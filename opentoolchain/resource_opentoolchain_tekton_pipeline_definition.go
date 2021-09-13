package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

func resourceOpenToolchainTektonPipelineDefinition() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage tekton pipeline definitions (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainTektonPipelineDefinitionCreate,
		ReadContext:   resourceOpenToolchainTektonPipelineDefinitionRead,
		DeleteContext: resourceOpenToolchainTektonPipelineDefinitionDelete,
		UpdateContext: resourceOpenToolchainTektonPipelineDefinitionUpdate,
		Schema: map[string]*schema.Schema{
			"guid": {
				Description: "The tekton pipeline definition `guid`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"toolchain_id": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"pipeline_id": {
				Description: "The tekton pipeline `guid`",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"env_id": {
				Description: "Environment ID, example: `ibm:yp:us-south`",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"input": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"github_integration_guid": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Required:    true,
						},
						"branch": {
							Description: "Github branch that contains tekton definition",
							Type:        schema.TypeString,
							Required:    true,
						},
						"path": {
							Description: "Path to tekton definition inside Github repository",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     ".tekton",
						},
					},
				},
			},
		},
	}
}

func resourceOpenToolchainTektonPipelineDefinitionCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	pipelineID := d.Get("pipeline_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	inputs := d.Get("input").(*schema.Set)

	config := m.(*ProviderConfig)
	c := config.OTClient

	definitionInputs, err := expandDefinitionInputs(c, &envID, &toolchainID, inputs.List())

	if err != nil {
		return diag.Errorf("Error creating pipeline definition inputs: %s", err)
	}

	options := &oc.CreateTektonPipelineDefinitionOptions{
		Inputs: definitionInputs,
		EnvID:  &envID,
		GUID:   &pipelineID,
	}

	// get definition ID first
	definition, _, err := c.CreateTektonPipelineDefinitionWithContext(ctx, options)

	if err != nil {
		return diag.Errorf("Error creating pipeline definition: %s", err)
	}

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:                 &pipelineID,
		EnvID:                &envID,
		PipelineDefinitionID: definition.Definition.ID,
		Inputs:               definition.Inputs,
	}

	// attach definition to pipeline
	_, _, err = c.PatchTektonPipelineWithContext(ctx, patchOptions)

	if err != nil {
		return diag.Errorf("Failed adding tekton pipeline definition: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", toolchainID, pipelineID, envID))
	return nil
}

func resourceOpenToolchainTektonPipelineDefinitionUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceOpenToolchainTektonPipelineDefinitionCreate(ctx, d, m)
}

func resourceOpenToolchainTektonPipelineDefinitionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	idParts := strings.Split(id, "/")

	toolchainID := idParts[0]
	pipelineID := idParts[1]
	envID := idParts[2]

	if len(idParts) < 3 {
		return diag.Errorf("Incorrect ID %s: ID should be a combination of toolchainID/pipelineID/envID", d.Id())
	}

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &pipelineID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading pipeline definition: %s", err)
	}

	d.Set("pipeline_id", pipelineID)
	d.Set("toolchain_id", toolchainID)
	d.Set("envID", envID)

	if err = d.Set("input", flattenPipelineInputs(pipeline.Inputs)); err != nil {
		return diag.Errorf("Error setting pipeline definition inputs: %s", err)
	}

	return nil
}

func flattenPipelineInputs(i []oc.TektonPipelineInput) []interface{} {
	var result []interface{}

	for _, in := range i {
		if *in.Type == "scm" {
			input := map[string]interface{}{
				"github_integration_guid": *in.ServiceInstanceID,
				"branch":                  *in.ScmSource.Branch,
				"path":                    *in.ScmSource.Path,
			}

			result = append(result, input)
		}
	}

	return result
}

// making additional call to get github integration URL, to simplify it for the user
func expandDefinitionInputs(c *oc.OpenToolchainV1, envID *string, toolchainID *string, inputs []interface{}) ([]oc.CreateTektonPipelineDefinitionParamsInputsItem, error) {
	result := make([]oc.CreateTektonPipelineDefinitionParamsInputsItem, len(inputs))

	for index, i := range inputs {
		input := i.(map[string]interface{})
		integrationGUID := input["github_integration_guid"].(string)
		branch := input["branch"].(string)
		path := input["path"].(string)

		ghInstance, _, err := c.GetServiceInstance(&oc.GetServiceInstanceOptions{
			GUID:        &integrationGUID,
			EnvID:       envID,
			ToolchainID: toolchainID,
		})

		if err != nil {
			return nil, err
		}

		result[index] = oc.CreateTektonPipelineDefinitionParamsInputsItem{
			Type:              getStringPtr("scm"),
			ServiceInstanceID: &integrationGUID,
			ScmSource: &oc.CreateTektonPipelineDefinitionParamsInputsItemScmSource{
				Path:            &path,
				URL:             getStringPtr(ghInstance.ServiceInstance.Parameters["repo_url"].(string)),
				Type:            getStringPtr("GitHub"),
				BlindConnection: getBoolPtr(false),
				Branch:          &branch,
			},
		}
	}

	return result, nil
}

func resourceOpenToolchainTektonPipelineDefinitionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	pipelineID := d.Get("pipeline_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:                 &pipelineID,
		EnvID:                &envID,
		PipelineDefinitionID: nil,
		Inputs:               []oc.TektonPipelineInput{},
	}

	// attach definition to pipeline
	_, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

	if err != nil {
		return diag.Errorf("Failed deleting tekton pipeline definition: %s", err)
	}

	return nil
}

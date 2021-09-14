package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	uuid "github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

const (
	pipelineServiceType = "pipeline"
	pipelineType        = "tekton"
)

func resourceOpenToolchainTektonPipeline() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage tekton pipeline (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainTektonPipelineCreate,
		ReadContext:   resourceOpenToolchainTektonPipelineRead,
		DeleteContext: resourceOpenToolchainTektonPipelineDelete,
		UpdateContext: resourceOpenToolchainTektonPipelineUpdate,
		Schema: map[string]*schema.Schema{
			"pipeline_id": {
				Description: "The tekton pipeline `guid`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"toolchain_id": {
				Description: "The toolchain `guid`",
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
			"name": {
				Description: "Pipeline name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"dashboard_url": {
				Description: "Pipeline dashboard URL",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"status": {
				Description: "Pipeline status",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"definition": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"github_integration_id": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Required:    true,
						},
						"github_url": {
							Description: "Github repository URL",
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

func resourceOpenToolchainTektonPipelineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	name := d.Get("name").(string)
	inputs := d.Get("definition").(*schema.Set)

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipelineUUID := uuid.NewString()
	// appending uuid temporarily to be able to retrieve pipeline guid once it is created
	pipelineName := fmt.Sprintf("%s/%s", name, pipelineUUID)

	options := &oc.CreateServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(pipelineServiceType),
		Parameters: &oc.CreateServiceInstanceParamsParameters{
			Name:       &pipelineName,
			Type:       getStringPtr(pipelineType),
			UIPipeline: getBoolPtr(true),
		},
	}

	_, _, err := c.CreateServiceInstanceWithContext(ctx, options)

	if err != nil {
		return diag.Errorf("Error creating tekton pipeline: %s", err)
	}

	// we have to get toolchain first, to be able to find pipeline ID
	// original POST API call does not provide it
	toolchain, _, err := c.GetToolchainWithContext(ctx, &oc.GetToolchainOptions{
		GUID:  &toolchainID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading toolchain: %s", err)
	}

	var instanceID string

	// find new pipeline instance
	if toolchain.Services != nil {
		for _, v := range toolchain.Services {
			if v.ServiceID != nil && *v.ServiceID == pipelineServiceType && v.Parameters != nil && v.Parameters["name"] == pipelineName && v.InstanceID != nil {
				instanceID = *v.InstanceID
				break
			}
		}
	}

	if instanceID == "" {
		// no way to cleanup since we don't know pipeline GUID
		return diag.Errorf("Unable to determine tekton pipeline GUID")
	}

	// update pipeline name, remove UUID suffix that was used to locate it
	_, err = c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		GUID:        &instanceID,
		EnvID:       &envID,
		ServiceID:   getStringPtr("pipeline"),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			Name:       &name, // removing UUID suffix
			Type:       getStringPtr(pipelineType),
			UIPipeline: getBoolPtr(true),
		},
	})

	if err != nil {
		// TODO: try deleting pipeline here to cleanup
		return diag.Errorf("Unable to update tekton pipeline name: %s", err)
	}

	definitionInputs := expandTektonPipelineDefinitionInputs(c, &envID, &toolchainID, inputs.List())

	definitionOptions := &oc.CreateTektonPipelineDefinitionOptions{
		Inputs: definitionInputs,
		EnvID:  &envID,
		GUID:   &instanceID,
	}

	// get definition ID first
	definition, _, err := c.CreateTektonPipelineDefinitionWithContext(ctx, definitionOptions)

	if err != nil {
		return diag.Errorf("Error creating pipeline definition: %s", err)
	}

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:                 &instanceID,
		EnvID:                &envID,
		PipelineDefinitionID: definition.Definition.ID,
		Inputs:               definition.Inputs,
		Worker: &oc.PatchTektonPipelineParamsWorker{
			// current pipeline defaults
			WorkerID:   getStringPtr("public"),
			WorkerType: getStringPtr("public"),
			WorkerName: getStringPtr("IBM Managed workers (Tekton Pipelines v0.20.1)"),
		},
	}

	_, _, err = c.PatchTektonPipelineWithContext(ctx, patchOptions)

	if err != nil {
		// TODO: try deleting pipeline here to cleanup
		return diag.Errorf("Unable to update tekton pipeline: %s", err)
	}

	d.Set("pipeline_id", instanceID)
	d.SetId(fmt.Sprintf("%s/%s", instanceID, envID))

	return resourceOpenToolchainTektonPipelineRead(ctx, d, m)
}

func resourceOpenToolchainTektonPipelineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	id := d.Id()
	idParts := strings.Split(id, "/")

	if len(idParts) < 2 {
		return diag.Errorf("Incorrect ID %s: ID should be a combination of pipelineID/envID", d.Id())
	}

	pipelineID := idParts[0]
	envID := idParts[1]

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &pipelineID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	if pipeline.Status != nil {
		d.Set("status", *pipeline.Status)
	}

	if pipeline.DashboardURL != nil {
		d.Set("dashboard_url", *pipeline.DashboardURL)
	}

	if pipeline.ToolchainID != nil {
		d.Set("toolchain_id", *pipeline.ToolchainID)
	}

	if err = d.Set("definition", flattenTektonPipelineDefinition(pipeline.Inputs)); err != nil {
		return diag.Errorf("Error setting pipeline definition inputs: %s", err)
	}

	return diags
}

func resourceOpenToolchainTektonPipelineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	pipelineID := d.Get("pipeline_id").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	_, err := c.DeleteServiceInstanceWithContext(ctx, &oc.DeleteServiceInstanceOptions{
		GUID:        &pipelineID,
		EnvID:       &envID,
		ToolchainID: &toolchainID,
	})

	if err != nil {
		return diag.Errorf("Error deleting tekton pipeline: %s", err)
	}

	d.SetId("")
	return diags
}

func resourceOpenToolchainTektonPipelineUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pipelineID := d.Get("pipeline_id").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	if d.HasChange("name") {
		name := d.Get("name").(string)

		_, err := c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
			ToolchainID: &toolchainID,
			GUID:        &pipelineID,
			EnvID:       &envID,
			ServiceID:   getStringPtr("pipeline"),
			Parameters: &oc.PatchServiceInstanceParamsParameters{
				Name:       &name,
				Type:       getStringPtr(pipelineType),
				UIPipeline: getBoolPtr(true),
			},
		})

		if err != nil {
			return diag.Errorf("Unable to update tekton pipeline name: %s", err)
		}
	}

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:  &pipelineID,
		EnvID: &envID,
	}

	if d.HasChange("definition") {
		inputs := d.Get("definition").(*schema.Set)
		definitionInputs := expandTektonPipelineDefinitionInputs(c, &envID, &toolchainID, inputs.List())

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

		patchOptions.PipelineDefinitionID = definition.Definition.ID
		patchOptions.Inputs = definition.Inputs
	}

	// add other conditions here
	if d.HasChange("definition") {
		_, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed updating tekton pipeline: %s", err)
		}
	}

	return resourceOpenToolchainTektonPipelineRead(ctx, d, m)
}

func expandTektonPipelineDefinitionInputs(c *oc.OpenToolchainV1, envID *string, toolchainID *string, inputs []interface{}) []oc.CreateTektonPipelineDefinitionParamsInputsItem {
	result := make([]oc.CreateTektonPipelineDefinitionParamsInputsItem, len(inputs))

	for index, i := range inputs {
		input := i.(map[string]interface{})
		integrationGUID := input["github_integration_id"].(string)
		branch := input["branch"].(string)
		path := input["path"].(string)
		url := input["github_url"].(string)

		result[index] = oc.CreateTektonPipelineDefinitionParamsInputsItem{
			Type:              getStringPtr("scm"),
			ServiceInstanceID: &integrationGUID,
			ScmSource: &oc.CreateTektonPipelineDefinitionParamsInputsItemScmSource{
				Path:            &path,
				URL:             &url,
				Type:            getStringPtr("GitHub"),
				BlindConnection: getBoolPtr(false),
				Branch:          &branch,
			},
		}
	}

	return result
}

func flattenTektonPipelineDefinition(d []oc.TektonPipelineInput) []interface{} {
	var result []interface{}

	for _, in := range d {
		if *in.Type == "scm" {
			input := map[string]interface{}{
				"github_integration_id": *in.ServiceInstanceID,
				"branch":                *in.ScmSource.Branch,
				"path":                  *in.ScmSource.Path,
				"github_url":            *in.ScmSource.URL,
			}

			result = append(result, input)
		}
	}

	return result
}

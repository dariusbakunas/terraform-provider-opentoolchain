package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
			"trigger": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "Trigger ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"enabled": {
							Description: "`true` if trigger should be active",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						"github_integration_id": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"github_url": {
							Description: "Github repository URL",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"name": {
							Description: "Trigger name",
							Type:        schema.TypeString,
							Required:    true,
						},
						"event_listener": {
							Description: "Event Listener name (from .tekton pipeline definition)",
							Type:        schema.TypeString,
							Required:    true,
						},
						"on_pull_request": {
							Description: "Trigger when pull request is opened or updated",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"on_pull_request_closed": {
							Description: "Trigger when pull request is closed",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"on_push": {
							Description: "Trigger when commit is pushed",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"branch": {
							Description: "GitHub branch",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"pattern": {
							Description: "GitHub branch pattern, if `branch` is not specified, otherwise setting is ignored",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"type": {
							Description:  "Trigger type",
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"scm", "manual"}, false),
							Required:     true,
						},
					},
				},
			},
			"text_env": {
				Description: "Pipeline environment text properties",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"secret_env": {
				Description: "Pipeline environment secret properties, use `{vault::vault_integration_name.VAULT_KEY}` with vault integration.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:  true,
				Sensitive: true,
			},
			"encrypted_secrets": {
				Type:        schema.TypeMap,
				Description: "Opentoolchain API does not return actual secret values, this is used internally to track changes to encrypted strings",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Sensitive: true,
				Computed:  true,
			},
		},
	}
}

func resourceOpenToolchainTektonPipelineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	name := d.Get("name").(string)
	inputs := d.Get("definition").(*schema.Set)
	triggers := d.Get("trigger").(*schema.Set)

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

	textEnv := d.Get("text_env").(map[string]interface{})
	secretEnv := d.Get("secret_env").(map[string]interface{})

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:                 &instanceID,
		EnvID:                &envID,
		EnvProperties:        expandTektonPipelineEnvProps(textEnv, secretEnv),
		PipelineDefinitionID: definition.Definition.ID,
		Inputs:               definition.Inputs,
		Triggers:             expandTektonPipelineTriggers(triggers.List()),
		Worker: &oc.PatchTektonPipelineParamsWorker{
			// current pipeline defaults
			WorkerID:   getStringPtr("public"),
			WorkerType: getStringPtr("public"),
			// TODO: should this be resource parameter? can we get a list of options?
			WorkerName: getStringPtr("IBM Managed workers (Tekton Pipelines v0.20.1)"),
		},
	}

	patchedPipeline, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

	if err != nil {
		// TODO: try deleting pipeline here to cleanup
		return diag.Errorf("Unable to update tekton pipeline: %s", err)
	}

	// TODO: move this to its onw fn
	encryptedSecrets := make(map[string]string)

	for _, v := range patchedPipeline.EnvProperties {
		if *v.Type == "SECURE" {
			encryptedSecrets[*v.Name] = *v.Value
		}
	}

	d.Set("encrypted_secrets", encryptedSecrets)

	d.Set("pipeline_id", instanceID)
	d.SetId(fmt.Sprintf("%s/%s", instanceID, envID))

	return resourceOpenToolchainTektonPipelineRead(ctx, d, m)
}

func resourceOpenToolchainTektonPipelineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	textEnv := getEnvMap(pipeline.EnvProperties, "TEXT")
	secretEnv := getEnvMap(pipeline.EnvProperties, "SECURE")

	if err := d.Set("text_env", textEnv); err != nil {
		return diag.Errorf("Error setting tekton pipeline text_env")
	}

	// there is no way to get actual secret values using current apis, it only provides encrypted strings
	// we can only detect if property is modified
	if env, ok := d.GetOk("secret_env"); ok {
		encryptedSecrets := d.Get("encrypted_secrets").(map[string]interface{})
		currentSecretEnvMap := env.(map[string]interface{})

		for k := range currentSecretEnvMap {
			if newVal, ok := secretEnv[k]; ok {
				if encryptedSecrets[k] != newVal {
					currentSecretEnvMap[k] = newVal // encrypted value changed, using encrypted string to force update
				}
			} else {
				delete(currentSecretEnvMap, k)
			}
		}

		if err := d.Set("secret_env", currentSecretEnvMap); err != nil {
			return diag.Errorf("Error setting pipeline secret_env: %s", err)
		}
	}

	encryptedSecrets := make(map[string]string)

	for _, v := range pipeline.EnvProperties {
		if *v.Type == "SECURE" {
			encryptedSecrets[*v.Name] = *v.Value
		}
	}

	if err := d.Set("encrypted_secrets", encryptedSecrets); err != nil {
		return diag.Errorf("Error setting pipeline encrypted_secrets: %s", err)
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

	if pipeline.Name != nil {
		d.Set("name", *pipeline.Name)
	}

	if err = d.Set("definition", flattenTektonPipelineDefinition(pipeline.Inputs)); err != nil {
		return diag.Errorf("Error setting pipeline definition inputs: %s", err)
	}

	if err = d.Set("trigger", flattenTektonPipelineTriggers(pipeline.Triggers)); err != nil {
		return diag.Errorf("Error setting pipeline triggers: %s", err)
	}

	return nil
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

	if d.HasChange("trigger") {
		triggers := d.Get("trigger").(*schema.Set)
		patchOptions.Triggers = expandTektonPipelineTriggers(triggers.List())
	}

	if d.HasChange("text_env") || d.HasChange("secret_env") {
		textEnv := d.Get("text_env").(map[string]interface{})
		secretEnv := d.Get("secret_env").(map[string]interface{})
		patchOptions.EnvProperties = expandTektonPipelineEnvProps(textEnv, secretEnv)
	}

	// add other conditions here
	if d.HasChange("definition") || d.HasChange("trigger") || d.HasChange("text_env") || d.HasChange("secret_env") {
		patchedPipeline, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed updating tekton pipeline: %s", err)
		}

		encryptedSecrets := make(map[string]string)

		for _, v := range patchedPipeline.EnvProperties {
			if *v.Type == "SECURE" {
				encryptedSecrets[*v.Name] = *v.Value
			}
		}

		d.Set("encrypted_secrets", encryptedSecrets)
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

func expandTektonPipelineEnvProps(text map[string]interface{}, secret map[string]interface{}) []oc.EnvProperty {
	var result []oc.EnvProperty

	if text != nil {
		for k, v := range text {
			value := v.(string)

			result = append(result, oc.EnvProperty{
				Name:  getStringPtr(k),
				Value: &value,
				Type:  getStringPtr("TEXT"),
			})
		}
	}

	if secret != nil {
		for k, v := range secret {
			value := v.(string)

			result = append(result, oc.EnvProperty{
				Name:  getStringPtr(k),
				Value: &value,
				Type:  getStringPtr("SECURE"),
			})
		}
	}

	return result
}

func expandTektonPipelineTriggers(t []interface{}) []oc.TektonPipelineTrigger {
	result := make([]oc.TektonPipelineTrigger, len(t))

	for index, trig := range t {
		trigger := trig.(map[string]interface{})
		name := trigger["name"].(string)
		eventListener := trigger["event_listener"].(string)
		triggerType := trigger["type"].(string)
		enabled := trigger["enabled"].(bool)

		result[index] = oc.TektonPipelineTrigger{
			ID:            getStringPtr(uuid.NewString()),
			Name:          &name,
			EventListener: &eventListener,
			Type:          &triggerType,
			Disabled:      getBoolPtr(!enabled),
		}

		if triggerType == "scm" {
			githubIntegrationID := trigger["github_integration_id"].(string)
			onPush := trigger["on_push"].(bool)
			onPR := trigger["on_pull_request"].(bool)
			onPRClosed := trigger["on_pull_request_closed"].(bool)
			branch := trigger["branch"].(string)
			pattern := trigger["pattern"].(string)
			url := trigger["github_url"].(string)

			result[index].ServiceInstanceID = &githubIntegrationID

			result[index].ScmSource = &oc.TektonPipelineTriggerScmSource{
				URL:     &url,
				Type:    getStringPtr("GitHub"), // TODO: what other types available?
				Branch:  &branch,
				Pattern: &pattern,
			}

			result[index].Events = &oc.TektonPipelineTriggerEvents{
				Push:              &onPush,
				PullRequest:       &onPR,
				PullRequestClosed: &onPRClosed,
			}
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

func flattenTektonPipelineTriggers(t []oc.TektonPipelineTrigger) []interface{} {
	var result []interface{}

	for _, trg := range t {
		trigger := map[string]interface{}{
			"id":             *trg.ID,
			"enabled":        !*trg.Disabled,
			"name":           *trg.Name,
			"event_listener": *trg.EventListener,
			"type":           *trg.Type,
		}

		if *trg.Type == "scm" {
			trigger["github_integration_id"] = *trg.ServiceInstanceID
			trigger["github_url"] = *trg.ScmSource.URL
			trigger["on_pull_request"] = *trg.Events.PullRequest
			trigger["on_pull_request_closed"] = *trg.Events.PullRequestClosed
			trigger["on_push"] = *trg.Events.Push
			trigger["branch"] = *trg.ScmSource.Branch
			trigger["pattern"] = *trg.ScmSource.Pattern
		}

		result = append(result, trigger)
	}

	return result
}

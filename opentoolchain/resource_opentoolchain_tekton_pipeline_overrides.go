package opentoolchain

import (
	"context"
	"fmt"
	"log"
	"strings"

	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpenToolchainTektonPipelineOverrides() *schema.Resource {
	return &schema.Resource{
		Description:   "Update *existing* tekton pipeline properties and triggers. If property exists, it will be updated in place, otherwise new one will be added. If trigger exists - it will be updated in place, otherwise it will be IGNORED (adding new triggers is not supported). When this resource is destroyed, original pipeline properties are restored. (WARN: using unpublished APIs)",
		CreateContext: resourceOpenToolchainTektonPipelineOverridesCreate,
		ReadContext:   resourceOpenToolchainTektonPipelineOverridesRead,
		DeleteContext: resourceOpenToolchainTektonPipelineOverridesDelete,
		UpdateContext: resourceOpenToolchainTektonPipelineOverridesUpdate,
		Schema: map[string]*schema.Schema{
			"guid": {
				Description: "The tekton pipeline `guid`",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
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
				Computed:    true,
			},
			"toolchain_guid": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"toolchain_crn": {
				Description: "The toolchain `crn`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"text_env": {
				Description: "Pipeline environment text properties that need to be updated",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"secret_env": {
				Description: "Pipeline environment secret properties that need to be updated, use `{vault::vault_integration_name.VAULT_KEY}` format with vault integration.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				//Sensitive: true,
			},
			"trigger": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "Trigger ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"github_integration_guid": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"type": {
							Description: "Trigger type",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"enabled": {
							Description: "Enable/disable the trigger",
							Type:        schema.TypeBool,
							Required:    true,
						},
						"name": {
							Description: "Trigger name, this is used for matching existing trigger",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"deleted_keys": {
				Description: "Any properties listed here will be deleted",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"new_keys": {
				Description: "Properties that were not part of original list (used internally)",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"original_properties": {
				Type:        schema.TypeList,
				Description: "Used internally to restore pipeline to it's original state once resource is deleted",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Sensitive: true,
				Computed:  true,
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

func resourceOpenToolchainTektonPipelineOverridesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	id := d.Id()
	idParts := strings.Split(id, "/")

	guid := idParts[0]
	envID := idParts[1]

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &guid,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	textEnv := getEnvMap(pipeline.EnvProperties, "TEXT")
	secretEnv := getEnvMap(pipeline.EnvProperties, "SECURE")

	if env, ok := d.GetOk("text_env"); ok {
		envMap := env.(map[string]interface{})
		for k := range envMap {
			if newVal, ok := textEnv[k]; ok {
				envMap[k] = newVal
			} else {
				// key no longer exists, delete to force update
				delete(envMap, k)
			}
		}
		d.Set("text_env", envMap)
	}

	if env, ok := d.GetOk("secret_env"); ok {
		encryptedSecrets := d.Get("encrypted_secrets")
		envMap := env.(map[string]interface{})
		for k := range envMap {
			if newVal, ok := secretEnv[k]; ok {
				if encryptedSecrets != nil {
					encrypted := encryptedSecrets.(map[string]interface{})
					// opentoolchain API does not return original secret values,
					// it returns encrypted strings instead, we save these during resource creation or update
					// check if encrypted secret did not change, to determine if update is required
					if encrypted[k] != newVal {
						envMap[k] = newVal // encrypted value changed, force update
					}
				} else {
					envMap[k] = newVal
				}
			} else {
				// key no longer exists, delete to force update
				delete(envMap, k)
			}
		}
		d.Set("secret_env", envMap)
	}

	if triggers, ok := d.GetOk("trigger"); ok {
		pipelineTriggerMap := make(map[string]oc.TektonPipelineTrigger)

		if pipeline.Triggers != nil {
			for _, t := range pipeline.Triggers {
				pipelineTriggerMap[*t.Name] = t
			}
		}

		var result []interface{}

		for _, t := range triggers.(*schema.Set).List() {
			tMap := t.(map[string]interface{})
			triggerName := tMap["name"].(string)

			if pipelineTrigger, ok := pipelineTriggerMap[triggerName]; ok {
				tMap["id"] = *pipelineTrigger.ID
				tMap["type"] = *pipelineTrigger.Type
				tMap["enabled"] = !*pipelineTrigger.Disabled

				if pipelineTrigger.ServiceInstanceID != nil {
					tMap["github_integration_guid"] = *pipelineTrigger.ServiceInstanceID
				}
			} else {
				log.Printf("[WARN] Trigger '%s' does not exist, it will be ignored", triggerName)
			}

			result = append(result, tMap)
		}

        d.Set("trigger", result)
	}

	// log.Printf("[DEBUG] Read tekton pipeline: %v", dbgPrint(pipeline))

	d.Set("name", *pipeline.Name)
	d.Set("toolchain_guid", *pipeline.ToolchainID)
	d.Set("toolchain_crn", *pipeline.ToolchainCRN)

	return diags
}

func resourceOpenToolchainTektonPipelineOverridesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:  &guid,
		EnvID: &envID,
	}

	// we have to read existing envProperties first
	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &guid,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	currentEnv := pipeline.EnvProperties

	textEnv, txtOk := d.GetOk("text_env")
	secretEnv, secOk := d.GetOk("secret_env")
	deletedKeys, delOk := d.GetOk("deleted_keys")
	triggers, trigOk := d.GetOk("trigger")

	originalProps, newKeys := createOriginalProps(currentEnv, textEnv, secretEnv, deletedKeys)
	d.Set("new_keys", newKeys)
	d.Set("original_properties", originalProps)

	if txtOk || secOk || delOk || trigOk {
		patchOptions.EnvProperties = makeEnvPatch(currentEnv, textEnv, secretEnv, deletedKeys, originalProps)

		if triggers != nil {
			patchOptions.Triggers = createTriggerPatch(triggers.(*schema.Set).List(), pipeline.Triggers)
		}

		// log.Printf("[DEBUG] Patching tekton pipeline: %v", dbgPrint(patchOptions))

		patchedPipeline, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed patching tekton pipeline: %s", err)
		}

		if patchedPipeline != nil {
			encryptedSecrets := make(map[string]string)

			for _, v := range patchedPipeline.EnvProperties {
				if *v.Type == "SECURE" {
					encryptedSecrets[*v.Name] = *v.Value
				}
			}

			d.Set("encrypted_secrets", encryptedSecrets)
		}
	}

	d.SetId(fmt.Sprintf("%s/%s", *pipeline.ID, envID))

	return resourceOpenToolchainTektonPipelineOverridesRead(ctx, d, m)
}

func resourceOpenToolchainTektonPipelineOverridesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:  &guid,
		EnvID: &envID,
	}

	originalProps := d.Get("original_properties")

	if originalProps != nil {
		// we have to read existing envProperties first
		pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
			GUID:  &guid,
			EnvID: &envID,
		})

		if err != nil {
			return diag.Errorf("Error reading tekton pipeline: %s", err)
		}

		currentEnv := pipeline.EnvProperties

		textEnv := d.Get("text_env")
		secretEnv := d.Get("secret_env")
		newKeys := d.Get("new_keys")

		var deletedKeys []interface{}

		originalMap := make(map[string]interface{})
		for _, p := range originalProps.([]interface{}) {
			prop := p.(map[string]interface{})
			originalMap[prop["name"].(string)] = p
		}

		if textEnv != nil {
			env := textEnv.(map[string]interface{})

			for k := range env {
				if _, ok := originalMap[k]; !ok {
					deletedKeys = append(deletedKeys, k)
				}
			}
		}

		if secretEnv != nil {
			env := secretEnv.(map[string]interface{})

			for k := range env {
				if _, ok := originalMap[k]; !ok {
					deletedKeys = append(deletedKeys, k)
				}
			}
		}

		// this will remove properties that were new
		if newKeys != nil {
			deletedKeys = append(deletedKeys, newKeys.([]interface{})...)
		}

		patchOptions.EnvProperties = makeEnvPatch(currentEnv, nil, nil, deletedKeys, originalProps)

		_, _, err = c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed deleting tekton pipeline: %s", err)
		}
	}

	d.SetId("")
	return diags
}

func resourceOpenToolchainTektonPipelineOverridesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange("text_env") || d.HasChange("secret_env") || d.HasChange("deleted_keys") || d.HasChange("trigger") {
		guid := d.Get("guid").(string)
		envID := d.Get("env_id").(string)

		config := m.(*ProviderConfig)
		c := config.OTClient

		// we have to read existing envProperties first
		pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
			GUID:  &guid,
			EnvID: &envID,
		})

		if err != nil {
			return diag.Errorf("Error reading tekton pipeline: %s", err)
		}

		currentEnv := pipeline.EnvProperties
		textEnv := d.Get("text_env")
		secretEnv := d.Get("secret_env")
		deletedKeys := d.Get("deleted_keys")
		originalProps := d.Get("original_properties")
		newKeys := d.Get("new_keys")
		triggers := d.Get("trigger")

		newOriginalProps, updatedNewKeys, deletedNewKeys := updateOriginalProps(currentEnv, textEnv, secretEnv, deletedKeys, newKeys, originalProps)

		// if new property is deleted, we need to make sure it is removed in patch payload
		if deletedNewKeys != nil {
		   if deletedKeys != nil {
		       deletedKeys = append(deletedKeys.([]interface{}), deletedNewKeys...)
           } else {
               deletedKeys = deletedNewKeys
           }
        }

		patchOptions := &oc.PatchTektonPipelineOptions{
			GUID:          &guid,
			EnvID:         &envID,
			EnvProperties: makeEnvPatch(currentEnv, textEnv, secretEnv, deletedKeys, newOriginalProps),
		}

		if triggers != nil {
			patchOptions.Triggers = createTriggerPatch(triggers.(*schema.Set).List(), pipeline.Triggers)
		}

		patchedPipeline, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed patching tekton pipeline: %s", err)
		}

		if patchedPipeline != nil {
			encryptedSecrets := make(map[string]string)

			for _, v := range patchedPipeline.EnvProperties {
				if *v.Type == "SECURE" {
					encryptedSecrets[*v.Name] = *v.Value
				}
			}

			d.Set("encrypted_secrets", encryptedSecrets)
		}

		// remove any values from original_properties that are no longer overridden
		props := cleanupOriginalProps(textEnv, secretEnv, deletedKeys, newOriginalProps)
		d.Set("original_properties", props)
		d.Set("new_keys", updatedNewKeys)
	}

	return resourceOpenToolchainTektonPipelineOverridesRead(ctx, d, m)
}

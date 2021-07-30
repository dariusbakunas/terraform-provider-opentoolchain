package opentoolchain

import (
	"context"
	"fmt"
	"strings"

	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpenToolchainPipelineProperties() *schema.Resource {
	return &schema.Resource{
		Description:   "Update tekton pipeline properties",
		CreateContext: resourceOpenToolchainPipelinePropertiesCreate,
		ReadContext:   resourceOpenToolchainPipelinePropertiesRead,
		DeleteContext: resourceOpenToolchainPipelinePropertiesDelete,
		UpdateContext: resourceOpenToolchainPipelinePropertiesUpdate,
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
				Description: "Pipeline environment secret properties that need to be updated, use `{vault::vault_integration_name.VAULT_KEY}` format with vault integration. Due to opentoolchain API limitation, direct string values will force update for every plan",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				//Sensitive: true,
			},
			"deleted_keys": {
				Description: "Any properties listed here will be deleted",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
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

func resourceOpenToolchainPipelinePropertiesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	// log.Printf("[DEBUG] Read tekton pipeline: %v", dbgPrint(pipeline))

	d.Set("name", *pipeline.Name)
	d.Set("toolchain_guid", *pipeline.ToolchainID)
	d.Set("toolchain_crn", *pipeline.ToolchainCRN)

	return diags
}

func resourceOpenToolchainPipelinePropertiesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

    var originalProps []interface{}
	for _, prop := range currentEnv {
	    originalProps = append(originalProps, map[string]interface{}{
	        "name": *prop.Name,
	        "value": *prop.Value,
	        "type": *prop.Type,
        })
    }

    d.Set("original_properties", originalProps)

	if txtOk || secOk || delOk {
		patchOptions.EnvProperties = makeEnvPatch(currentEnv, textEnv, secretEnv, deletedKeys)

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

	return resourceOpenToolchainPipelinePropertiesRead(ctx, d, m)
}

func resourceOpenToolchainPipelinePropertiesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
        var props []oc.EnvProperty

        for _, p := range originalProps.([]interface{}) {
            pMap := p.(map[string]interface{})
            props = append(props, oc.EnvProperty{
                Name: getStringPtr(pMap["name"].(string)),
                Type: getStringPtr(pMap["type"].(string)),
                Value: getStringPtr(pMap["value"].(string)),
            })
        }

        patchOptions.EnvProperties = props

        _, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

        if err != nil {
            return diag.Errorf("Failed deleting tekton pipeline: %s", err)
        }
    }

	d.SetId("")
	return diags
}

func resourceOpenToolchainPipelinePropertiesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	if d.HasChange("text_env") || d.HasChange("secret_env") || d.HasChange("deleted_keys") {
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

		patchOptions := &oc.PatchTektonPipelineOptions{
			GUID:          &guid,
			EnvID:         &envID,
			EnvProperties: makeEnvPatch(currentEnv, textEnv, secretEnv, deletedKeys),
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
	}

	return resourceOpenToolchainPipelinePropertiesRead(ctx, d, m)
}

func makeEnvPatch(currentEnv []oc.EnvProperty, textEnv interface{}, secretEnv interface{}, deletedKeys interface{}) []oc.EnvProperty {
	envMap := make(map[string]oc.EnvProperty)

	for _, p := range currentEnv {
		envMap[*p.Name] = p
	}

	if textEnv != nil {
		env := textEnv.(map[string]interface{})

		for k, v := range env {
			value := v.(string)

			envMap[k] = oc.EnvProperty{
				Name:  getStringPtr(k),
				Value: &value,
				Type:  getStringPtr("TEXT"),
			}
		}
	}

	// note: if secret has duplicate key as textEnv it will overwrite it
	if secretEnv != nil {
		env := secretEnv.(map[string]interface{})

		for k, v := range env {
			value := v.(string)

			envMap[k] = oc.EnvProperty{
				Name:  getStringPtr(k),
				Value: &value,
				Type:  getStringPtr("SECURE"),
			}
		}
	}

	if deletedKeys != nil {
		for _, key := range deletedKeys.([]interface{}) {
			delete(envMap, key.(string))
		}
	}

	var res []oc.EnvProperty

	for _, v := range envMap {
		res = append(res, v)
	}

	return res
}

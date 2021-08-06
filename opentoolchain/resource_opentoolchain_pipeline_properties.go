package opentoolchain

import (
	"context"
	"fmt"
	"sort"
	"strings"

	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpenToolchainPipelineProperties() *schema.Resource {
	return &schema.Resource{
		Description:        "Update *existing* tekton pipeline properties. If property exists, it will be updated in place, otherwise new one will be added. When this resource is destroyed, original pipeline properties are restored. (WARN: using unpublished APIs)",
		DeprecationMessage: "Use opentoolchain_tekton_pipeline_overrides resource instead",
		CreateContext:      resourceOpenToolchainPipelinePropertiesCreate,
		ReadContext:        resourceOpenToolchainPipelinePropertiesRead,
		DeleteContext:      resourceOpenToolchainPipelinePropertiesDelete,
		UpdateContext:      resourceOpenToolchainPipelinePropertiesUpdate,
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

	originalProps, newKeys := createOriginalProps(currentEnv, textEnv, secretEnv, deletedKeys)
	d.Set("new_keys", newKeys)
	d.Set("original_properties", originalProps)

	if txtOk || secOk || delOk {
		patchOptions.EnvProperties = makeEnvPatch(currentEnv, textEnv, secretEnv, deletedKeys, originalProps)

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

func resourceOpenToolchainPipelinePropertiesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange("text_env") || d.HasChange("secret_env") || d.HasChange("deleted_keys") {
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

		newOriginalProps, updatedNewKeys, _ := updateOriginalProps(currentEnv, textEnv, secretEnv, deletedKeys, newKeys, originalProps)

		patchOptions := &oc.PatchTektonPipelineOptions{
			GUID:          &guid,
			EnvID:         &envID,
			EnvProperties: makeEnvPatch(currentEnv, textEnv, secretEnv, deletedKeys, newOriginalProps),
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

	return resourceOpenToolchainPipelinePropertiesRead(ctx, d, m)
}

// compares source map keys or array of strings against target map keys
// returns a list of matched keys and new keys
func getKeyDiff(targetMap map[string]interface{}, source interface{}) (matchedKeys, newKeys []interface{}) {
    if m, ok := source.(map[string]interface{}); ok {
        for k := range m {
            if _, ok := targetMap[k]; ok {
                matchedKeys = append(matchedKeys, k)
                continue
            }

            newKeys = append(newKeys, k)
        }
    }

    if arr, ok := source.([]interface{}); ok {
        for _, k := range arr {
            if _, ok := targetMap[k.(string)]; ok {
                matchedKeys = append(matchedKeys, k)
                continue
            }

            newKeys = append(newKeys, k)
        }
    }

    return matchedKeys, newKeys
}

// we want to only retain properties that were mentioned in resource inputs, ignore the rest
// that way if some property is updated in UI and it was never overridden in terraform, we won't "restore" to the
// old value once this resource is destroyed
func createOriginalProps(currentEnv []oc.EnvProperty, textEnv interface{}, secretEnv interface{}, deletedKeys interface{}) (originalProps, newKeys []interface{}) {
	var existingKeys []interface{}

	envMap := make(map[string]interface{})

	for _, p := range currentEnv {
		envMap[*p.Name] = p
	}

	if textEnv != nil {
		env := textEnv.(map[string]interface{})
		e, n := getKeyDiff(envMap, env)
		existingKeys = append(existingKeys, e...)
		newKeys = append(newKeys, n...)
	}

	if secretEnv != nil {
		env := secretEnv.(map[string]interface{})
        e, n := getKeyDiff(envMap, env)
        existingKeys = append(existingKeys, e...)
        newKeys = append(newKeys, n...)
	}

	if deletedKeys != nil {
        e, n := getKeyDiff(envMap, deletedKeys)
        existingKeys = append(existingKeys, e...)
        newKeys = append(newKeys, n...) // not common, but nothing stops user from adding new property to deleted keys list
	}

	for _, k := range existingKeys {
		original := envMap[k.(string)].(oc.EnvProperty)
		originalProps = append(originalProps, map[string]interface{}{
			"name":  *original.Name,
			"value": *original.Value,
			"type":  *original.Type,
		})
	}

	sort.Slice(originalProps, func(i, j int) bool {
		a := originalProps[i].(map[string]interface{})["name"].(string)
		b := originalProps[j].(map[string]interface{})["name"].(string)
		return a < b
	})

	return originalProps, newKeys
}

// we need to update original properties if new key matching current properties is added to any resource inputs
func updateOriginalProps(currentEnv []oc.EnvProperty, textEnv interface{}, secretEnv interface{}, deletedKeys interface{}, newKeys interface{}, originalProps interface{}) (updatedOriginalProps, updatedNewKeys, deletedNewKeys []interface{}) {
	if originalProps == nil {
		updatedOriginalProps, updatedNewKeys = createOriginalProps(currentEnv, textEnv, secretEnv, deletedKeys)
		return updatedNewKeys, updatedNewKeys, nil
	}

	currentMap := make(map[string]oc.EnvProperty)
	originalMap := make(map[string]interface{})
	newKeyMap := make(map[string]interface{})

	allKeys := make(map[string]bool)

	for _, p := range currentEnv {
		currentMap[*p.Name] = p
	}

	if newKeys != nil {
		for _, k := range newKeys.([]interface{}) {
			newKeyMap[k.(string)] = true
		}
	}

	for _, p := range originalProps.([]interface{}) {
		prop := p.(map[string]interface{})
		originalMap[prop["name"].(string)] = p
	}

	// TODO: try removing some duplication between text and secret
	if textEnv != nil {
		env := textEnv.(map[string]interface{})

		for key := range env {
			allKeys[key] = true
			if _, ok := originalMap[key]; !ok {
				// if we're overriding new property, make sure to save it to originals, but only if this is not new key
				if current, ok := currentMap[key]; ok {
					if _, ok := newKeyMap[key]; !ok {
						originalMap[key] = map[string]interface{}{
							"name":  *current.Name,
							"value": *current.Value,
							"type":  *current.Type,
						}
					}
				} else {
					// this is new property, we need to update `new_keys` list to make sure we clean it up when resource is destroyed
					newKeyMap[key] = true
					//updatedNewKeys = append(updatedNewKeys, key)
				}
			}
		}
	}

	if secretEnv != nil {
		env := secretEnv.(map[string]interface{})

		for key := range env {
			allKeys[key] = true

			if _, ok := originalMap[key]; !ok {
				// if we're overriding new property, make sure to save it to originals, but only if this is not new key
				if current, ok := currentMap[key]; ok {
					if _, ok := newKeyMap[key]; !ok {
						originalMap[key] = map[string]interface{}{
							"name":  *current.Name,
							"value": *current.Value,
							"type":  *current.Type,
						}
					}
				} else {
					// this is new property, we need to update `new_keys` list to make sure we clean it up when resource is destroyed
					newKeyMap[key] = true
				}
			}
		}
	}

	if deletedKeys != nil {
		for _, k := range deletedKeys.([]interface{}) {
			key := k.(string)

			allKeys[key] = true

			if _, ok := originalMap[key]; !ok {
				// if we're deleting new property, make sure to save it to originals, but only if this is not new key
				if current, ok := currentMap[key]; ok {
					if _, ok := newKeyMap[key]; !ok {
						originalMap[key] = map[string]interface{}{
							"name":  *current.Name,
							"value": *current.Value,
							"type":  *current.Type,
						}
					}
				} else {
					newKeyMap[key] = true
				}
			}
		}
	}

	for k := range newKeyMap {
		if _, ok := allKeys[k]; !ok {
			// this new property was removed
			delete(newKeyMap, k)
			deletedNewKeys = append(deletedNewKeys, k)
		}
	}

	for _, original := range originalMap {
		updatedOriginalProps = append(updatedOriginalProps, original)
	}

	sort.Slice(updatedOriginalProps, func(i, j int) bool {
		a := updatedOriginalProps[i].(map[string]interface{})["name"].(string)
		b := updatedOriginalProps[j].(map[string]interface{})["name"].(string)
		return a < b
	})

	for k := range newKeyMap {
		updatedNewKeys = append(updatedNewKeys, k)
	}

	sort.Slice(updatedNewKeys, func(i, j int) bool {
		a := updatedNewKeys[i].(string)
		b := updatedNewKeys[j].(string)
		return a < b
	})

	return updatedOriginalProps, updatedNewKeys, deletedNewKeys
}

// apply partial patch to only properties that are mentioned in textEnv, secretEnv or deleted in deletedKeys
// restore originals if any inputs (overrides) are removed
func makeEnvPatch(currentEnv []oc.EnvProperty, textEnv interface{}, secretEnv interface{}, deletedKeys interface{}, originalProps interface{}) []oc.EnvProperty {
	envMap := make(map[string]oc.EnvProperty)

	for _, p := range currentEnv {
		envMap[*p.Name] = p
	}

	// restore originals first and then apply changes, that way we don't need to do diff
	if originalProps != nil {
		for _, p := range originalProps.([]interface{}) {
			prop := p.(map[string]interface{})

			key := prop["name"].(string)
			value := prop["value"].(string)
			propType := prop["type"].(string)

			envMap[key] = oc.EnvProperty{
				Name:  getStringPtr(key),
				Value: &value,
				Type:  getStringPtr(propType),
			}
		}
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

// we only want to keep original properties that are overridden, make sure this is last step in update method
func cleanupOriginalProps(textEnv interface{}, secretEnv interface{}, deletedKeys interface{}, originalProps interface{}) interface{} {
	var result []interface{}

	if originalProps == nil {
		return nil
	}

	allKeys := make(map[string]bool)

	if textEnv != nil {
		env := textEnv.(map[string]interface{})
		for k := range env {
			allKeys[k] = true
		}
	}

	if secretEnv != nil {
		env := secretEnv.(map[string]interface{})
		for k := range env {
			allKeys[k] = true
		}
	}

	if deletedKeys != nil {
		for _, key := range deletedKeys.([]interface{}) {
			allKeys[key.(string)] = true
		}
	}

	for _, p := range originalProps.([]interface{}) {
		prop := p.(map[string]interface{})
		key := prop["name"].(string)
		if _, ok := allKeys[key]; ok {
			result = append(result, p)
		}
	}

	return result
}

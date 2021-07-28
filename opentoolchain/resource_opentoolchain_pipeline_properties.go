package opentoolchain

import (
	"context"
	"fmt"
	"log"

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
		},
	}
}

func resourceOpenToolchainPipelinePropertiesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

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
				// key no longer exists? is it possible?
				delete(envMap, k)
			}
		}
		d.Set("text_env", envMap)
	}

	if env, ok := d.GetOk("secret_env"); ok {
		envMap := env.(map[string]interface{})
		for k := range envMap {
			if newVal, ok := secretEnv[k]; ok {
				envMap[k] = newVal
			} else {
				// key no longer exists? is it possible?
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

	if txtOk || secOk {
		patchOptions.EnvProperties = makeEnvPatch(currentEnv, textEnv, secretEnv)

		// log.Printf("[DEBUG] Patching tekton pipeline: %v", dbgPrint(patchOptions))

		_, _, err = c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed patching tekton pipeline: %s", err)
		}
	}

	d.SetId(fmt.Sprintf("%s/%s", *pipeline.ID, envID))

	return resourceOpenToolchainPipelinePropertiesRead(ctx, d, m)
}

func resourceOpenToolchainPipelinePropertiesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// NO-OP: implement once pipeline deletion apis are available
	var diags diag.Diagnostics
	d.SetId("")
	return diags
}

func resourceOpenToolchainPipelinePropertiesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	if d.HasChange("text_env") || d.HasChange("secret_env") {
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

		patchOptions := &oc.PatchTektonPipelineOptions{
			GUID:          &guid,
			EnvID:         &envID,
			EnvProperties: makeEnvPatch(currentEnv, textEnv, secretEnv),
		}

		_, _, err = c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed patching tekton pipeline: %s", err)
		}
	}

	return resourceOpenToolchainPipelinePropertiesRead(ctx, d, m)
}

func makeEnvPatch(currentEnv []oc.EnvProperty, textEnv interface{}, secretEnv interface{}) []oc.EnvProperty {
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
			if _, ok := envMap[k]; ok {
				log.Printf("[WARN] Secret property '%s' will overwrite matching text property", k)
			}

			value := v.(string)

			envMap[k] = oc.EnvProperty{
				Name:  getStringPtr(k),
				Value: &value,
				Type:  getStringPtr("SECURE"),
			}
		}
	}

	var res []oc.EnvProperty

	for _, v := range envMap {
		res = append(res, v)
	}

	return res
}

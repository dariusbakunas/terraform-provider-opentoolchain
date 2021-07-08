package opentoolchain

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
)

func dataSourceOpenToolchainTektonPipeline() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOpenToolchainTektonPipelineRead,
		Schema: map[string]*schema.Schema{
			"guid": {
				Description: "The tekton pipeline `guid`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"env_id": {
				Description: "Environment ID, example: `ibm:yp:us-south`",
				Type:        schema.TypeString,
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
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"secret_env": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataSourceOpenToolchainTektonPipelineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	log.Printf("[DEBUG] Read tekton pipeline: %+v", pipeline)

	d.Set("name", *pipeline.Name)
	d.Set("toolchain_guid", *pipeline.ToolchainID)
	d.Set("toolchain_crn", *pipeline.ToolchainCRN)

	d.SetId(fmt.Sprintf("%s:%s", *pipeline.ID, envID))

	d.Set("text_env", getEnvMap(pipeline.EnvProperties, "TEXT"))
	d.Set("secret_env", getEnvMap(pipeline.EnvProperties, "SECURE"))

	return diags
}

func getEnvMap(envProps []oc.EnvProperty, envType string) map[string]string {
	res := make(map[string]string)

	for _, prop := range envProps {
		if *prop.Type == envType {
			res[*prop.Name] = *prop.Value
		}
	}

	return res
}

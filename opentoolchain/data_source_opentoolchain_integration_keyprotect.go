package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenToolchainIntegrationKeyProtect() *schema.Resource {
	return &schema.Resource{
		Description: "Get IBM KeyProtect integration information (WARN: using undocumented APIs)",
		ReadContext: dataSourceOpenToolchainIntegrationKeyProtectRead,
		Schema: map[string]*schema.Schema{
			"toolchain_id": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"integration_id": {
				Description: "The integration `guid`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"env_id": {
				Description: "Environment ID, example: `ibm:yp:us-south`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"instance_region": {
				Description: "KeyProtect instance region, example: `ibm:yp:us-east`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"instance_name": {
				Description: "KeyProtect instance name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "Integration name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"resource_group": {
				Description: "The name of the resource group of KeyProtect instance",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceOpenToolchainIntegrationKeyProtectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	integrationID := d.Get("integration_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	svc, _, err := c.GetServiceInstanceWithContext(ctx, &oc.GetServiceInstanceOptions{
		EnvID:       &envID,
		ToolchainID: &toolchainID,
		GUID:        &integrationID,
	})

	if err != nil {
		return diag.Errorf("Error reading keyprotect service instance: %s", err)
	}

	if svc.ServiceInstance != nil && svc.ServiceInstance.Parameters != nil {
		params := svc.ServiceInstance.Parameters
		if n, ok := params["name"]; ok {
			d.Set("name", n.(string))
		}

		if r, ok := params["region"]; ok {
			d.Set("instance_region", r.(string))
		}

		if r, ok := params["resource-group"]; ok {
			d.Set("resource_group", r.(string))
		}

		if n, ok := params["instance-name"]; ok {
			d.Set("instance_name", n.(string))
		}

		// TODO: do we need `integration-status` ??
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", integrationID, toolchainID, envID))
	return nil
}

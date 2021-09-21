package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenToolchainIntegrationPagerDuty() *schema.Resource {
	return &schema.Resource{
		Description: "Get PagerDuty integration information (WARN: using undocumented APIs)",
		ReadContext: dataSourceOpenToolchainIntegrationPagerDutyRead,
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
			"service_id": {
				Description: "Name of PagerDuty service ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"service_url": {
				Description: "Name of PagerDuty service URL",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"service_name": {
				Description: "Name of PagerDuty service to post alerts to",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"primary_email": {
				Description: "The email address of the user to contact when alert is posted (required if `service_name` is specified)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"primary_phone_number": {
				Description: "The phone number of the user to contact when alert is posted. If national code is omitted, `+1` is set by default (required if `service_name` is specified)",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceOpenToolchainIntegrationPagerDutyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
		return diag.Errorf("Error reading pagerduty service instance: %s", err)
	}

	if svc.ServiceInstance != nil && svc.ServiceInstance.Parameters != nil {
		params := svc.ServiceInstance.Parameters

		if i, ok := params["service_id"]; ok {
			d.Set("service_id", i.(string))
		}

		if u, ok := params["service_url"]; ok {
			d.Set("service_url", u.(string))
		}

		if n, ok := params["service_name"]; ok {
			d.Set("service_name", n.(string))
		}

		if e, ok := params["user_email"]; ok {
			d.Set("primary_email", e.(string))
		}

		if p, ok := params["user_phone"]; ok {
			d.Set("primary_phone_number", p.(string))
		}
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", integrationID, toolchainID, envID))
	return nil
}

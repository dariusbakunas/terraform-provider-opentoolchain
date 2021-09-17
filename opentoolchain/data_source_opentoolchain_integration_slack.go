package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenToolchainIntegrationSlack() *schema.Resource {
	return &schema.Resource{
		Description: "Get IBM Slack integration information (WARN: using undocumented APIs)",
		ReadContext: dataSourceOpenToolchainIntegrationSlackRead,
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
			"channel_name": {
				Description: "Slack channel name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"team_name": {
				Description: "Slack team name, the phrase before `.slack.com`, for example if your team URL is https://team.slack.com, the team name is `team`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"events": {
				Description: "Events for which you want to receive notifications",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"pipeline_start": {
							Description: "Send slack notification when pipeline is started",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"pipeline_success": {
							Description: "Send slack notification when pipeline succeeds",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"pipeline_fail": {
							Description: "Send slack notification when pipeline fails",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"toolchain_bind": {
							Description: "Send slack notification when integration is created",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"toolchain_unbind": {
							Description: "Send slack notification when integration is removed",
							Type:        schema.TypeBool,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceOpenToolchainIntegrationSlackRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
		return diag.Errorf("Error reading slack service instance: %s", err)
	}

	if svc.ServiceInstance != nil && svc.ServiceInstance.Parameters != nil {
		params := svc.ServiceInstance.Parameters
		events := []interface{}{map[string]bool{
			"pipeline_start":   params["pipeline_start"].(bool),
			"pipeline_success": params["pipeline_success"].(bool),
			"pipeline_fail":    params["pipeline_fail"].(bool),
			"toolchain_bind":   params["toolchain_bind"].(bool),
			"toolchain_unbind": params["toolchain_unbind"].(bool),
		}}

		if err := d.Set("events", events); err != nil {
			return diag.Errorf("Error setting slack integration events: %s", err)
		}

		if n, ok := params["channel_name"]; ok {
			d.Set("channel_name", n.(string))
		}

		if t, ok := params["team_url"]; ok {
			d.Set("team_name", t.(string))
		}
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", integrationID, toolchainID, envID))
	return nil
}

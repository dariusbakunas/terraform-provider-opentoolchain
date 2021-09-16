package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"strings"
)

const (
	slackIntegrationServiceType = "slack"
)

func resourceOpenToolchainIntegrationSlack() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage IBM Slack integration (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainIntegrationSlackCreate,
		ReadContext:   resourceOpenToolchainIntegrationSlackRead,
		DeleteContext: resourceOpenToolchainIntegrationSlackDelete,
		UpdateContext: resourceOpenToolchainIntegrationSlackUpdate,
		Schema: map[string]*schema.Schema{
			"toolchain_id": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"integration_id": {
				Description: "The integration `guid`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"env_id": {
				Description: "Environment ID, example: `ibm:yp:us-south`",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"webhook_url": {
				Description: "Slack webhook URL, use `{vault::vault_integration_name.VAULT_KEY}` with vault integration.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"encrypted_webhook_url": {
				Description: "Since API only provides encrypted webook URL value, we can use that internally to track changes",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			"channel_name": {
				Description: "Slack channel name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"team_name": {
				Description: "Slack team name, the phrase before `.slack.com`, for example if your team URL is https://team.slack.com, the team name is `team`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"events": {
				Description: "Events for which you want to receive notifications",
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"pipeline_start": {
							Description: "Send slack notification when pipeline is started",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						"pipeline_success": {
							Description: "Send slack notification when pipeline succeeds",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						"pipeline_fail": {
							Description: "Send slack notification when pipeline fails",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						"toolchain_bind": {
							Description: "Send slack notification when integration is created",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						"toolchain_unbind": {
							Description: "Send slack notification when integration is removed",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
					},
				},
			},
		},
	}
}

func resourceOpenToolchainIntegrationSlackCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	teamName := d.Get("team_name").(string)
	channelName := d.Get("channel_name").(string)
	webhookURL := d.Get("webhook_url").(string)
	evt, evtOK := d.GetOk("events")

	events := map[string]bool{
		"pipeline_start":   true,
		"pipeline_success": true,
		"pipeline_fail":    true,
		"toolchain_bind":   true,
		"toolchain_unbind": true,
	}

	if evtOK {
		e := evt.([]interface{})[0].(map[string]interface{})

		events = map[string]bool{
			"pipeline_start":   e["pipeline_start"].(bool),
			"pipeline_success": e["pipeline_success"].(bool),
			"pipeline_fail":    e["pipeline_fail"].(bool),
			"toolchain_bind":   e["toolchain_bind"].(bool),
			"toolchain_unbind": e["toolchain_unbind"].(bool),
		}
	}

	config := m.(*ProviderConfig)
	c := config.OTClient

	integrationUUID := uuid.NewString()
	uuidChannelName := fmt.Sprintf("%s/%s", channelName, integrationUUID)

	options := &oc.CreateServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(slackIntegrationServiceType),
		Parameters: &oc.CreateServiceInstanceParamsParameters{
			ChannelName:     &uuidChannelName,
			TeamURL:         &teamName,
			APIToken:        &webhookURL,
			PipelineStart:   getBoolPtr(events["pipeline_start"]),
			PipelineSuccess: getBoolPtr(events["pipeline_success"]),
			PipelineFail:    getBoolPtr(events["pipeline_fail"]),
			ToolchainBind:   getBoolPtr(events["toolchain_bind"]),
			ToolchainUnbind: getBoolPtr(events["toolchain_unbind"]),
		},
	}

	_, _, err := c.CreateServiceInstanceWithContext(ctx, options)

	if err != nil {
		return diag.Errorf("Error creating Slack integration: %s", err)
	}

	toolchain, _, err := c.GetToolchainWithContext(ctx, &oc.GetToolchainOptions{
		GUID:  &toolchainID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading toolchain: %s", err)
	}

	var integrationID string

	// find new service instance
	if toolchain.Services != nil {
		for _, v := range toolchain.Services {
			if v.ServiceID != nil && *v.ServiceID == slackIntegrationServiceType && v.Parameters != nil && v.Parameters["channel_name"] == uuidChannelName && v.InstanceID != nil {
				integrationID = *v.InstanceID
				break
			}
		}
	}

	if integrationID == "" {
		// no way to cleanup since we don't know pipeline GUID
		return diag.Errorf("Unable to determine Slack integration GUID")
	}

	_, err = c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		GUID:        &integrationID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(slackIntegrationServiceType),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			APIToken:    &webhookURL,
			ChannelName: &channelName,
		},
	})

	if err != nil {
		return diag.Errorf("Unable to update Slack channel name: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", integrationID, toolchainID, envID))

	return resourceOpenToolchainIntegrationSlackRead(ctx, d, m)
}

func resourceOpenToolchainIntegrationSlackRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	idParts := strings.Split(id, "/")

	if len(idParts) < 3 {
		return diag.Errorf("Incorrect ID %s: ID should be a combination of integrationID/pipelineID/envID", d.Id())
	}

	integrationID := idParts[0]
	toolchainID := idParts[1]
	envID := idParts[2]

	d.Set("integration_id", integrationID)
	d.Set("toolchain_id", toolchainID)
	d.Set("env_id", envID)

	config := m.(*ProviderConfig)
	c := config.OTClient

	svc, resp, err := c.GetServiceInstanceWithContext(ctx, &oc.GetServiceInstanceOptions{
		EnvID:       &envID,
		ToolchainID: &toolchainID,
		GUID:        &integrationID,
	})

	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] Slack service instance '%s' is not found, removing it from state", integrationID)
			d.SetId("")
			return nil
		}

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

		if w, ok := params["api_token"]; ok {
			newValue := w.(string)
			currentValue := d.Get("encrypted_webhook_url").(string)

			if currentValue != "" && currentValue != newValue {
				d.Set("webhook_url", newValue) // force update
			}

			d.Set("encrypted_webhook_url", newValue)
		}

		if n, ok := params["channel_name"]; ok {
			d.Set("channel_name", n.(string))
		}

		if t, ok := params["team_url"]; ok {
			d.Set("team_name", t.(string))
		}
	}

	return nil
}

func resourceOpenToolchainIntegrationSlackDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	integrationID := d.Get("integration_id").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	_, err := c.DeleteServiceInstanceWithContext(ctx, &oc.DeleteServiceInstanceOptions{
		GUID:        &integrationID,
		EnvID:       &envID,
		ToolchainID: &toolchainID,
	})

	if err != nil {
		return diag.Errorf("Error deleting Slack integration: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceOpenToolchainIntegrationSlackUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	instanceID := d.Get("integration_id").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	webhookURL := d.Get("webhook_url").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	options := &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		GUID:        &instanceID,
		ServiceID:   getStringPtr(slackIntegrationServiceType),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			APIToken: &webhookURL, // seems to be mandatory for patching
		},
	}

	if d.HasChange("channel_name") {
		channelName := d.Get("channel_name").(string)
		options.Parameters.ChannelName = &channelName
	}

	if d.HasChange("team_name") {
		teamName := d.Get("team_name").(string)
		options.Parameters.TeamURL = &teamName
	}

	if d.HasChange("events") {
		evt := d.Get("events").([]interface{})
		events := evt[0].(map[string]interface{})

		options.Parameters.PipelineStart = getBoolPtr(events["pipeline_start"].(bool))
		options.Parameters.PipelineSuccess = getBoolPtr(events["pipeline_success"].(bool))
		options.Parameters.PipelineFail = getBoolPtr(events["pipeline_fail"].(bool))
		options.Parameters.ToolchainBind = getBoolPtr(events["toolchain_bind"].(bool))
		options.Parameters.ToolchainUnbind = getBoolPtr(events["toolchain_unbind"].(bool))
	}

	if d.HasChange("channel_name") || d.HasChange("team_name") || d.HasChange("events") || d.HasChange("events") || d.HasChange("webhook_url") {
		_, err := c.PatchServiceInstanceWithContext(ctx, options)

		if err != nil {
			return diag.Errorf("Unable to update Slack integration: %s", err)
		}
	}

	return resourceOpenToolchainIntegrationSlackRead(ctx, d, m)
}

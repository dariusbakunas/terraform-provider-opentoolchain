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
	pagerDutyIntegrationServiceType = "pagerduty"
)

func resourceOpenToolchainIntegrationPagerDuty() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage PagerDuty integration (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainIntegrationPagerDutyCreate,
		ReadContext:   resourceOpenToolchainIntegrationPagerDutyRead,
		DeleteContext: resourceOpenToolchainIntegrationPagerDutyDelete,
		UpdateContext: resourceOpenToolchainIntegrationPagerDutyUpdate,
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
			"api_key": {
				Description: "PagerDuty API key, use `{vault::vault_integration_name.API_KEY}` with vault integration.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"encrypted_api_key": {
				Description: "Since API only provides encrypted API key value, we can use that internally to track changes",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			"service_id": {
				Description: "Name of PagerDuty service ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"service_url": {
				Description:   "Name of PagerDuty service URL",
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"service_name", "primary_email", "primary_phone_number"},
			},
			"service_name": {
				Description:   "Name of PagerDuty service to post alerts to",
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"service_url"},
			},
			"primary_email": {
				Description:   "The email address of the user to contact when alert is posted (required if `service_name` is specified)",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"service_url"},
			},
			"primary_phone_number": {
				Description:   "The phone number of the user to contact when alert is posted. If national code is omitted, `+1` is set by default (required if `service_name` is specified)",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"service_url"},
			},
		},
	}
}

func resourceOpenToolchainIntegrationPagerDutyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	apiKey := d.Get("api_key").(string)
	serviceName := d.Get("service_name").(string)
	serviceURL := d.Get("service_url").(string)
	primaryEmail := d.Get("primary_email").(string)
	primaryPhoneNumber := d.Get("primary_phone_number").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	integrationUUID := uuid.NewString()
	// we don't want to use serviceName here, because this integration creates new service
	// in pagerduty if it does not exist
	uuidPrimaryEmail := fmt.Sprintf("%s/%s", primaryEmail, integrationUUID)
	uuidServiceURL := fmt.Sprintf("%s/%s", serviceURL, integrationUUID)

	keyType := "api"

	if serviceURL != "" {
		keyType = "service"
	}

	options := &oc.CreateServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(pagerDutyIntegrationServiceType),
		Parameters: &oc.CreateServiceInstanceParamsParameters{
			KeyType:     &keyType,
			ServiceName: &serviceName,
			ServiceURL:  &serviceURL,
			UserEmail:   &primaryEmail,
			UserPhone:   &primaryPhoneNumber,
		},
	}

	if keyType == "api" {
		options.Parameters.APIKey = &apiKey
		options.Parameters.UserEmail = &uuidPrimaryEmail // used to later locate the service
	} else {
		options.Parameters.ServiceKey = &apiKey
		options.Parameters.ServiceURL = &uuidServiceURL // used to later locate the service
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
			if v.ServiceID != nil && *v.ServiceID == pagerDutyIntegrationServiceType && v.Parameters != nil && (v.Parameters["user_email"] == uuidPrimaryEmail || v.Parameters["service_url"] == uuidServiceURL) && v.InstanceID != nil {
				integrationID = *v.InstanceID
				break
			}
		}
	}

	if integrationID == "" {
		// no way to cleanup since we don't know slack integration GUID
		return diag.Errorf("Unable to determine PagerDuty integration GUID")
	}

	_, err = c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		GUID:        &integrationID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(pagerDutyIntegrationServiceType),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			APIKey:     &apiKey,
			UserEmail:  &primaryEmail,
			ServiceURL: &serviceURL,
		},
	})

	if err != nil {
		// try cleaning up
		_, deleteErr := c.DeleteServiceInstanceWithContext(ctx, &oc.DeleteServiceInstanceOptions{
			GUID:        &integrationID,
			EnvID:       &envID,
			ToolchainID: &toolchainID,
		})

		if deleteErr != nil {
			return diag.Errorf("PagerDuty creation failed: %s, Unable to cleanup: %s", err, deleteErr)
		}

		return diag.Errorf("PagerDuty creation failed: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", integrationID, toolchainID, envID))

	return resourceOpenToolchainIntegrationPagerDutyRead(ctx, d, m)
}

func resourceOpenToolchainIntegrationPagerDutyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	idParts := strings.Split(id, "/")

	if len(idParts) < 3 {
		return diag.Errorf("Incorrect ID %s: ID should be a combination of integrationID/toolchainID/envID", d.Id())
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
			log.Printf("[WARN] PagerDuty service instance '%s' is not found, removing it from state", integrationID)
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error reading pagerduty service instance: %s", err)
	}

	if svc.ServiceInstance != nil && svc.ServiceInstance.Parameters != nil {
		params := svc.ServiceInstance.Parameters

		if a, ok := params["api_key"]; ok {
			newValue := a.(string)
			currentValue := d.Get("encrypted_api_key").(string)

			if currentValue != "" && currentValue != newValue {
				d.Set("api_key", newValue) // force update
			}

			d.Set("encrypted_api_key", newValue)
		}

		if s, ok := params["service_name"]; ok {
			d.Set("service_name", s.(string))
		}

		if s, ok := params["service_id"]; ok {
			d.Set("service_id", s.(string))
		}

		if s, ok := params["service_url"]; ok {
			d.Set("service_url", s.(string))
		}

		if e, ok := params["user_email"]; ok {
			d.Set("primary_email", e.(string))
		}

		if p, ok := params["user_phone"]; ok {
			d.Set("primary_phone_number", p.(string))
		}
	}

	return nil
}

func resourceOpenToolchainIntegrationPagerDutyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
		return diag.Errorf("Error deleting PagerDuty integration: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceOpenToolchainIntegrationPagerDutyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	instanceID := d.Get("integration_id").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	apiKey := d.Get("api_key").(string)
	serviceName := d.Get("service_name").(string)
	serviceURL := d.Get("service_url").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	keyType := "api"

	if serviceURL != "" {
		keyType = "service"
	}

	options := &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		GUID:        &instanceID,
		ServiceID:   getStringPtr(pagerDutyIntegrationServiceType),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			ServiceName: &serviceName,
			ServiceURL:  &serviceURL,
			KeyType:     &keyType,
		},
	}

	if keyType == "api" {
		options.Parameters.APIKey = &apiKey
	} else {
		options.Parameters.ServiceKey = &apiKey
	}

	if d.HasChange("primary_email") {
		primaryEmail := d.Get("primary_email").(string)
		options.Parameters.UserEmail = &primaryEmail
	}

	if d.HasChange("primary_phone_number") {
		primaryPhoneNumber := d.Get("primary_phone_number").(string)
		options.Parameters.UserPhone = &primaryPhoneNumber
	}

	if d.HasChange("primary_email") || d.HasChange("primary_phone_number") || d.HasChange("api_key") {
		_, err := c.PatchServiceInstanceWithContext(ctx, options)

		if err != nil {
			return diag.Errorf("Unable to update PagerDuty integration: %s", err)
		}
	}

	return resourceOpenToolchainIntegrationPagerDutyRead(ctx, d, m)
}

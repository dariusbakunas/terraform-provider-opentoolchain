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
	keyProtectIntegrationServiceType = "keyprotect"
)

func resourceOpenToolchainIntegrationKeyProtect() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage IBM KeyProtect integration (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainIntegrationKeyProtectCreate,
		ReadContext:   resourceOpenToolchainIntegrationKeyProtectRead,
		DeleteContext: resourceOpenToolchainIntegrationKeyProtectDelete,
		UpdateContext: resourceOpenToolchainIntegrationKeyProtectUpdate,
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
			"instance_region": {
				Description: "KeyProtect instance region, example: `ibm:yp:us-east`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"instance_name": {
				Description: "KeyProtect instance name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "Integration name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"resource_group": {
				Description: "The name of the resource group of KeyProtect instance",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func resourceOpenToolchainIntegrationKeyProtectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	name := d.Get("name").(string)
	instanceName := d.Get("instance_name").(string)
	instanceRegion := d.Get("instance_region").(string)
	resourceGroup := d.Get("resource_group").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	integrationUUID := uuid.NewString()
	uuidName := fmt.Sprintf("%s/%s", name, integrationUUID)

	options := &oc.CreateServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(keyProtectIntegrationServiceType),
		Parameters: &oc.CreateServiceInstanceParamsParameters{
			InstanceName:  &instanceName,
			Name:          &uuidName,
			Region:        &instanceRegion,
			ResourceGroup: &resourceGroup,
		},
	}

	_, _, err := c.CreateServiceInstanceWithContext(ctx, options)

	if err != nil {
		return diag.Errorf("Error creating KeyProtect integration: %s", err)
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
			if v.ServiceID != nil && *v.ServiceID == keyProtectIntegrationServiceType && v.Parameters != nil && v.Parameters["name"] == uuidName && v.InstanceID != nil {
				integrationID = *v.InstanceID
				break
			}
		}
	}

	if integrationID == "" {
		// no way to cleanup since we don't know pipeline GUID
		return diag.Errorf("Unable to determine KeyProtect integration GUID")
	}

	_, err = c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		GUID:        &integrationID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(keyProtectIntegrationServiceType),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			Name: &name,
		},
	})

	if err != nil {
		return diag.Errorf("Unable to update Github URL: %s", err)
	}

	d.Set("integration_id", integrationID)
	d.SetId(fmt.Sprintf("%s/%s/%s", integrationID, toolchainID, envID))

	return resourceOpenToolchainIntegrationKeyProtectRead(ctx, d, m)
}

func resourceOpenToolchainIntegrationKeyProtectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
			log.Printf("[WARN] KeyProtect service instance '%s' is not found, removing it from state", integrationID)
			d.SetId("")
			return nil
		}

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

	return nil
}

func resourceOpenToolchainIntegrationKeyProtectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
		return diag.Errorf("Error deleting KeyProtect integration: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceOpenToolchainIntegrationKeyProtectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	instanceID := d.Get("integration_id").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	options := &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		GUID:        &instanceID,
		ServiceID:   getStringPtr(keyProtectIntegrationServiceType),
		Parameters:  &oc.PatchServiceInstanceParamsParameters{},
	}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		options.Parameters.Name = &name
	}

	if d.HasChange("instance_name") {
		name := d.Get("instance_name").(string)
		options.Parameters.InstanceName = &name
	}

	if d.HasChange("instance_region") {
		region := d.Get("instance_region").(string)
		options.Parameters.Region = &region
	}

	if d.HasChange("resource_group") {
		group := d.Get("resource_group").(string)
		options.Parameters.ResourceGroup = &group
	}

	// list all change conditions here
	if d.HasChange("name") || d.HasChange("instance_name") || d.HasChange("instance_region") || d.HasChange("resource_group") {
		_, err := c.PatchServiceInstanceWithContext(ctx, options)

		if err != nil {
			return diag.Errorf("Unable to update KeyProtect integration: %s", err)
		}
	}

	return resourceOpenToolchainIntegrationKeyProtectRead(ctx, d, m)
}

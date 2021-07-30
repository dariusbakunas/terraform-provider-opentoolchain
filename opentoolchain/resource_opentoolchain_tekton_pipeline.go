package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	uuid "github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

const (
	pipelineType = "tekton"
)

func resourceOpenToolchainTektonPipeline() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage tekton pipeline (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainTektonPipelineCreate,
		ReadContext:   resourceOpenToolchainTektonPipelineRead,
		DeleteContext: resourceOpenToolchainTektonPipelineDelete,
		UpdateContext: resourceOpenToolchainTektonPipelineUpdate,
		Schema: map[string]*schema.Schema{
			"guid": {
				Description: "The tekton pipeline `guid`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"toolchain_id": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
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
				Required:    true,
			},
			"dashboard_url": {
				Description: "Pipeline dashboard URL",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"status": {
				Description: "Pipeline status",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceOpenToolchainTektonPipelineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	name := d.Get("name").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipelineUUID := uuid.NewString()
	// appending uuid temporarily to be able to retrieve pipeline guid once it is created
	pipelineName := fmt.Sprintf("%s/%s", name, pipelineUUID)

	options := &oc.CreateServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		ServiceID:   getStringPtr("pipeline"),
		Parameters: &oc.CreateServiceInstanceParamsParameters{
			Name:       &pipelineName,
			Type:       getStringPtr(pipelineType),
			UIPipeline: getBoolPtr(true),
		},
	}

	_, _, err := c.CreateServiceInstanceWithContext(ctx, options)

	if err != nil {
		return diag.Errorf("Error creating tekton pipeline: %s", err)
	}

	toolchain, _, err := c.GetToolchainWithContext(ctx, &oc.GetToolchainOptions{
		GUID:  &toolchainID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading toolchain: %s", err)
	}

	var instanceID string

	// find new pipeline instance
	if toolchain.Services != nil {
		for _, v := range toolchain.Services {
			if v.ServiceID != nil && *v.ServiceID == "pipeline" && v.Parameters != nil && v.Parameters["name"] == pipelineName && v.InstanceID != nil {
				instanceID = *v.InstanceID
				break
			}
		}
	}

	if instanceID == "" {
		// no way to cleanup since we don't know pipeline GUID
		return diag.Errorf("Unable to determine tekton pipeline GUID")
	}

	_, err = c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		GUID:        &instanceID,
		EnvID:       &envID,
		ServiceID:   getStringPtr("pipeline"),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			Name:       &name, // removing UUID suffix
			Type:       getStringPtr(pipelineType),
			UIPipeline: getBoolPtr(true),
		},
	})

	if err != nil {
		// TODO: try deleting pipeline here to cleanup
		return diag.Errorf("Unable to update tekton pipeline name: %s", err)
	}

	// update worker
	_, _, err = c.PatchTektonPipelineWithContext(ctx, &oc.PatchTektonPipelineOptions{
		GUID:  &instanceID,
		EnvID: &envID,
		Worker: &oc.PatchTektonPipelineParamsWorker{
			// current pipeline defaults
			WorkerID:   getStringPtr("public"),
			WorkerType: getStringPtr("public"),
			WorkerName: getStringPtr("IBM Managed workers (Tekton Pipelines v0.20.1)"),
		},
	})

	if err != nil {
		// TODO: try deleting pipeline here to cleanup
		return diag.Errorf("Unable to update tekton pipeline worker: %s", err)
	}

	d.Set("guid", instanceID)
	d.SetId(fmt.Sprintf("%s/%s", instanceID, envID))

	return resourceOpenToolchainTektonPipelineRead(ctx, d, m)
}

func resourceOpenToolchainTektonPipelineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	id := d.Id()
	idParts := strings.Split(id, "/")

	pipelineID := idParts[0]
	envID := idParts[1]

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &pipelineID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	if pipeline.Status != nil {
		d.Set("status", *pipeline.Status)
	}

	if pipeline.DashboardURL != nil {
		d.Set("dashboard_url", *pipeline.DashboardURL)
	}

	return diags
}

func resourceOpenToolchainTektonPipelineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	pipelineID := d.Get("guid").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	_, err := c.DeleteServiceInstanceWithContext(ctx, &oc.DeleteServiceInstanceOptions{
		GUID:        &pipelineID,
		EnvID:       &envID,
		ToolchainID: &toolchainID,
	})

	if err != nil {
		return diag.Errorf("Error deleting tekton pipeline: %s", err)
	}

	d.SetId("")
	return diags
}

func resourceOpenToolchainTektonPipelineUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pipelineID := d.Get("guid").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	if d.HasChange("name") {
		name := d.Get("name").(string)

		_, err := c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
			ToolchainID: &toolchainID,
			GUID:        &pipelineID,
			EnvID:       &envID,
			ServiceID:   getStringPtr("pipeline"),
			Parameters: &oc.PatchServiceInstanceParamsParameters{
				Name:       &name,
				Type:       getStringPtr(pipelineType),
				UIPipeline: getBoolPtr(true),
			},
		})

		if err != nil {
			return diag.Errorf("Unable to update tekton pipeline name: %s", err)
		}
	}

	return resourceOpenToolchainTektonPipelineRead(ctx, d, m)
}

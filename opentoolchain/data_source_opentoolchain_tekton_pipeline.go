package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

func dataSourceOpenToolchainTektonPipeline() *schema.Resource {
	return &schema.Resource{
		Description: "Get tekton pipeline information",
		ReadContext: dataSourceOpenToolchainTektonPipelineRead,
		Schema: map[string]*schema.Schema{
			"pipeline_id": {
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
			"definition": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"github_integration_id": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"github_url": {
							Description: "Github repository URL",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"branch": {
							Description: "Github branch that contains tekton definition",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"path": {
							Description: "Path to tekton definition inside Github repository",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"trigger": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "Trigger ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"enabled": {
							Description: "`true` if trigger should be active",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"github_integration_id": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"github_url": {
							Description: "Github repository URL",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "Trigger name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"event_listener": {
							Description: "Event Listener name (from .tekton pipeline definition)",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"on_pull_request": {
							Description: "Trigger when pull request is opened or updated",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"on_pull_request_closed": {
							Description: "Trigger when pull request is closed",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"on_push": {
							Description: "Trigger when commit is pushed",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"branch": {
							Description: "GitHub branch",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"pattern": {
							Description: "GitHub branch pattern, if `branch` is not specified, otherwise setting is ignored",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"type": {
							Description: "Trigger type",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"text_env": {
				Description: "Pipeline environment text properties",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
	}
}

func dataSourceOpenToolchainTektonPipelineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	pipelineID := d.Get("pipeline_id").(string)

	envIDParts := strings.Split(envID, ":")
	region := envIDParts[len(envIDParts)-1]

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:   &pipelineID,
		Region: &region,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	textEnv := getEnvMap(pipeline.EnvProperties, "TEXT")

	if err := d.Set("text_env", textEnv); err != nil {
		return diag.Errorf("Error setting tekton pipeline text_env")
	}

	if pipeline.Status != nil {
		d.Set("status", *pipeline.Status)
	}

	if pipeline.DashboardURL != nil {
		d.Set("dashboard_url", *pipeline.DashboardURL)
	}

	if pipeline.Name != nil {
		d.Set("name", *pipeline.Name)
	}

	if err = d.Set("definition", flattenTektonPipelineDefinition(pipeline.Inputs)); err != nil {
		return diag.Errorf("Error setting pipeline definition inputs: %s", err)
	}

	if err = d.Set("trigger", flattenTektonPipelineTriggers(pipeline.Triggers)); err != nil {
		return diag.Errorf("Error setting pipeline triggers: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s", pipelineID, envID))

	return nil
}

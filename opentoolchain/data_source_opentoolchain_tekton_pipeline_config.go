package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"strings"
)

func dataSourceOpenToolchainTektonPipelineConfig() *schema.Resource {
	return &schema.Resource{
		Description: "Get tekton pipeline configuration",
		ReadContext: dataSourceOpenToolchainTektonPipelineConfigRead,
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
				Description: "Pipeline environment text properties",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"secret_env": {
				Description: "Pipeline environment secret properties",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:  true,
				Sensitive: true,
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
						"github_integration_guid": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"scm": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"branch": {
										Description: "Github branch for scm triggers",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"pattern": {
										Description: "Github branch pattern for scm triggers",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"type": {
										Description: "SCM type",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"url": {
										Description: "Github url for scm triggers",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"hook_id": {
										Description: "Hook ID for scm triggers",
										Type:        schema.TypeString,
										Computed:    true,
									},
								},
							},
						},
						"type": {
							Description: "Trigger type",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"enabled": {
							Description: "Enable/disable the trigger",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"event_listener": {
							Description: "",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "Trigger name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"events": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"pull_request": {
										Description: "When pull request is opened or updated",
										Type:        schema.TypeBool,
										Computed:    true,
									},
									"pull_request_closed": {
										Description: "When pull request is closed",
										Type:        schema.TypeBool,
										Computed:    true,
									},
									"push": {
										Description: "When commit is pushed",
										Type:        schema.TypeBool,
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceOpenToolchainTektonPipelineConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	envIDParts := strings.Split(envID, ":")
	region := envIDParts[len(envIDParts)-1]

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:   &guid,
		Region: &region,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	log.Printf("[DEBUG] Read tekton pipeline: %+v", pipeline)

	d.Set("name", *pipeline.Name)
	d.Set("toolchain_guid", *pipeline.ToolchainID)
	d.Set("toolchain_crn", *pipeline.ToolchainCRN)

	if err := d.Set("text_env", getEnvMap(pipeline.EnvProperties, "TEXT")); err != nil {
		return diag.Errorf("Error setting tekton pipline text_env: %s", err)
	}

	if err := d.Set("secret_env", getEnvMap(pipeline.EnvProperties, "SECURE")); err != nil {
		return diag.Errorf("Error setting tekton pipline secret_env: %s", err)
	}

	if err := d.Set("trigger", flattenPipelineTriggers(pipeline.Triggers)); err != nil {
		return diag.Errorf("Error setting tekton pipline trigger: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s", *pipeline.ID, envID))

	return diags
}

func flattenPipelineTriggers(triggers []oc.TektonPipelineTrigger) []interface{} {
	var result []interface{}

	if triggers == nil {
		return result
	}

	for _, t := range triggers {
		trigger := map[string]interface{}{
			"id":             *t.ID,
			"scm":            flattenTriggerSCM(t.ScmSource),
			"type":           *t.Type,
			"enabled":        !*t.Disabled,
			"event_listener": *t.EventListener,
			"events":         flattenTriggerEvents(t.Events),
		}

		if t.Name != nil {
			trigger["name"] = *t.Name
		}

		if t.ServiceInstanceID != nil {
			trigger["github_integration_guid"] = *t.ServiceInstanceID
		}

		result = append(result, trigger)
	}

	return result
}

func flattenTriggerSCM(scm *oc.TektonPipelineTriggerScmSource) []interface{} {
	if scm == nil {
		return []interface{}{}
	}

	s := map[string]interface{}{}

	if scm.Branch != nil {
		s["branch"] = *scm.Branch
	}

	if scm.Pattern != nil {
		s["pattern"] = *scm.Pattern
	}

	if scm.Type != nil {
		s["type"] = *scm.Type
	}

	if scm.URL != nil {
		s["url"] = *scm.URL
	}

	// TODO: find workaround for handling both string and int
	//if scm.HookID != nil {
	//	s["hook_id"] = *scm.HookID
	//}

	return []interface{}{s}
}

func flattenTriggerEvents(events *oc.TektonPipelineTriggerEvents) []interface{} {
	if events == nil {
		return []interface{}{}
	}

	e := map[string]interface{}{}

	if events.Push != nil {
		e["push"] = *events.Push
	}

	if events.PullRequest != nil {
		e["pull_request"] = *events.PullRequest
	}

	if events.PullRequestClosed != nil {
		e["pull_request_closed"] = *events.PullRequestClosed
	}

	return []interface{}{e}
}

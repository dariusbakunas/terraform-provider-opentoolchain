package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strings"
)

func resourceOpenToolchainTektonPipelineTriggers() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage tekton pipeline triggers (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainTektonPipelineTriggersCreate,
		ReadContext:   resourceOpenToolchainTektonPipelineTriggersRead,
		DeleteContext: resourceOpenToolchainTektonPipelineTriggerDelete,
		UpdateContext: resourceOpenToolchainTektonPipelineTriggersUpdate,
		Schema: map[string]*schema.Schema{
			"pipeline_id": {
				Description: "The tekton pipeline `guid`",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"env_id": {
				Description: "Environment ID, example: `ibm:yp:us-south`",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"trigger": {
				Type:     schema.TypeSet,
				Required: true,
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
							Optional:    true,
							Default:     true,
						},
						"github_integration_id": {
							Description: "Github integration ID",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"github_url": {
							Description: "Github repository URL",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"name": {
							Description: "Trigger name",
							Type:        schema.TypeString,
							Required:    true,
						},
						"event_listener": {
							Description: "Event Listener name (from .tekton pipeline definition)",
							Type:        schema.TypeString,
							Required:    true,
						},
						"on_pull_request": {
							Description: "Trigger when pull request is opened or updated",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"on_pull_request_closed": {
							Description: "Trigger when pull request is closed",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"on_push": {
							Description: "Trigger when commit is pushed",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"branch": {
							Description:   "GitHub branch",
							Type:          schema.TypeString,
							Optional:      true,
						},
						"pattern": {
							Description:   "GitHub branch pattern, if `branch` is not specified, otherwise setting is ignored",
							Type:          schema.TypeString,
							Optional:      true,
						},
						"type": {
							Description:  "Trigger type",
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"scm", "manual"}, false),
							Required:     true,
						},
					},
				},
			},
		},
	}
}

func resourceOpenToolchainTektonPipelineTriggersCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	pipelineID := d.Get("pipeline_id").(string)
	triggers := d.Get("trigger").(*schema.Set)

	config := m.(*ProviderConfig)
	c := config.OTClient

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:     &pipelineID,
		EnvID:    &envID,
		Triggers: expandPipelineTriggers(triggers.List()),
	}

	_, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

	if err != nil {
		return diag.Errorf("Failed adding tekton pipeline triggers: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s", pipelineID, envID))

	return nil
}

func resourceOpenToolchainTektonPipelineTriggersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	idParts := strings.Split(id, "/")

	pipelineID := idParts[0]
	envID := idParts[1]

	if len(idParts) < 2 {
		return diag.Errorf("Incorrect ID %s: ID should be a combination of pipelineID/envID", d.Id())
	}

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &pipelineID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading pipeline triggers: %s", err)
	}

	d.Set("pipeline_id", pipelineID)
	d.Set("envID", envID)

	if err = d.Set("trigger", flattenTektonPipelineTriggers(pipeline.Triggers)); err != nil {
		return diag.Errorf("Error setting pipeline definition inputs: %s", err)
	}

	return nil
}

func resourceOpenToolchainTektonPipelineTriggerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	pipelineID := d.Get("pipeline_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:     &pipelineID,
		EnvID:    &envID,
		Triggers: []oc.TektonPipelineTrigger{},
	}

	_, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

	if err != nil {
		return diag.Errorf("Failed deleting tekton pipeline triggers: %s", err)
	}

	d.SetId("")

	return nil
}

func resourceOpenToolchainTektonPipelineTriggersUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	pipelineID := d.Get("pipeline_id").(string)
	triggers := d.Get("trigger").(*schema.Set)

	config := m.(*ProviderConfig)
	c := config.OTClient

	patchOptions := &oc.PatchTektonPipelineOptions{
		GUID:     &pipelineID,
		EnvID:    &envID,
		Triggers: expandPipelineTriggers(triggers.List()),
	}

	_, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

	if err != nil {
		return diag.Errorf("Failed updating tekton pipeline triggers: %s", err)
	}

	return nil
}

func expandPipelineTriggers(t []interface{}) []oc.TektonPipelineTrigger {
	result := make([]oc.TektonPipelineTrigger, len(t))

	for index, trig := range t {
		trigger := trig.(map[string]interface{})
		name := trigger["name"].(string)
		eventListener := trigger["event_listener"].(string)
		triggerType := trigger["type"].(string)
		enabled := trigger["enabled"].(bool)

		result[index] = oc.TektonPipelineTrigger{
			ID:            getStringPtr(uuid.NewString()),
			Name:          &name,
			EventListener: &eventListener,
			Type:          &triggerType,
			Disabled:      getBoolPtr(!enabled),
		}

		if triggerType == "scm" {
			githubIntegrationID := trigger["github_integration_id"].(string)
			onPush := trigger["on_push"].(bool)
			onPR := trigger["on_pull_request"].(bool)
			onPRClosed := trigger["on_pull_request_closed"].(bool)
			branch := trigger["branch"].(string)
			pattern := trigger["pattern"].(string)
			url := trigger["github_url"].(string)

			result[index].ServiceInstanceID = &githubIntegrationID

			result[index].ScmSource = &oc.TektonPipelineTriggerScmSource{
				URL:     &url,
				Type:    getStringPtr("GitHub"), // TODO: what other types available?
				Branch:  &branch,
				Pattern: &pattern,
			}

			result[index].Events = &oc.TektonPipelineTriggerEvents{
				Push:              &onPush,
				PullRequest:       &onPR,
				PullRequestClosed: &onPRClosed,
			}
		}
	}

	return result
}

func flattenTektonPipelineTriggers(t []oc.TektonPipelineTrigger) []interface{} {
	var result []interface{}

	for _, trg := range t {
		trigger := map[string]interface{}{
			"id":             *trg.ID,
			"enabled":        !*trg.Disabled,
			"name":           *trg.Name,
			"event_listener": *trg.EventListener,
			"type":           *trg.Type,
		}

		if *trg.Type == "scm" {
			trigger["github_integration_id"] = *trg.ServiceInstanceID
			trigger["github_url"] = *trg.ScmSource.URL
			trigger["on_pull_request"] = *trg.Events.PullRequest
			trigger["on_pull_request_closed"] = *trg.Events.PullRequestClosed
			trigger["on_push"] = *trg.Events.Push
			trigger["branch"] = *trg.ScmSource.Branch
			trigger["pattern"] = *trg.ScmSource.Pattern
		}

		result = append(result, trigger)
	}

	return result
}

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

func resourceOpenToolchainPipelineTriggers() *schema.Resource {
	return &schema.Resource{
		Description:        "Update *existing* tekton pipeline triggers. If trigger exists, it will be updated in place, otherwise trigger will be ignored (adding new triggers is not supported). (WARN: using unpublished APIs) (DEPRECATED)",
		DeprecationMessage: "Use opentoolchain_tekton_pipeline_overrides resource instead",
		CreateContext:      resourceOpenToolchainPipelineTriggersCreate,
		ReadContext:        resourceOpenToolchainPipelineTriggersRead,
		UpdateContext:      resourceOpenToolchainPipelineTriggersUpdate,
		DeleteContext:      resourceOpenToolchainPipelineTriggersDelete,
		Schema: map[string]*schema.Schema{
			"guid": {
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
				Optional: true,
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
						"type": {
							Description: "Trigger type",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"enabled": {
							Description: "Enable/disable the trigger",
							Type:        schema.TypeBool,
							Required:    true,
						},
						"name": {
							Description: "Trigger name, this is used for matching existing trigger",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func resourceOpenToolchainPipelineTriggersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	id := d.Id()
	idParts := strings.Split(id, "/")

	guid := idParts[0]
	envID := idParts[1]

	config := m.(*ProviderConfig)
	c := config.OTClient

	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &guid,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	if triggers, ok := d.GetOk("trigger"); ok {
		pipelineTriggerMap := make(map[string]oc.TektonPipelineTrigger)

		if pipeline.Triggers != nil {
			for _, t := range pipeline.Triggers {
				pipelineTriggerMap[*t.Name] = t
			}
		}

		var result []interface{}

		for _, t := range triggers.(*schema.Set).List() {
			tMap := t.(map[string]interface{})
			triggerName := tMap["name"].(string)

			if pipelineTrigger, ok := pipelineTriggerMap[triggerName]; ok {
				tMap["id"] = *pipelineTrigger.ID
				tMap["type"] = *pipelineTrigger.Type
				tMap["enabled"] = !*pipelineTrigger.Disabled

				if pipelineTrigger.ServiceInstanceID != nil {
					tMap["github_integration_guid"] = *pipelineTrigger.ServiceInstanceID
				}
			} else {
				log.Printf("[WARN] Trigger '%s' does not exist, it will be ignored", triggerName)
			}

			result = append(result, tMap)
		}

		d.Set("trigger", result)
	}

	return diags
}

func resourceOpenToolchainPipelineTriggersCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	// we have to read existing triggers first
	pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
		GUID:  &guid,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading tekton pipeline: %s", err)
	}

	if triggers, ok := d.GetOk("trigger"); ok {
		patchOptions := &oc.PatchTektonPipelineOptions{
			GUID:     &guid,
			EnvID:    &envID,
			Triggers: createTriggerPatch(triggers.(*schema.Set).List(), pipeline.Triggers),
		}

		_, _, err := c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed patching tekton pipeline: %s", err)
		}
	}

	d.SetId(fmt.Sprintf("%s/%s", *pipeline.ID, envID))

	return resourceOpenToolchainPipelineTriggersRead(ctx, d, m)
}

func resourceOpenToolchainPipelineTriggersUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange("trigger") {
		guid := d.Get("guid").(string)
		envID := d.Get("env_id").(string)

		config := m.(*ProviderConfig)
		c := config.OTClient

		// we have to read existing envProperties first
		pipeline, _, err := c.GetTektonPipelineWithContext(ctx, &oc.GetTektonPipelineOptions{
			GUID:  &guid,
			EnvID: &envID,
		})

		if err != nil {
			return diag.Errorf("Error reading tekton pipeline: %s", err)
		}

		triggers := d.Get("trigger")

		patchOptions := &oc.PatchTektonPipelineOptions{
			GUID:     &guid,
			EnvID:    &envID,
			Triggers: createTriggerPatch(triggers.(*schema.Set).List(), pipeline.Triggers),
		}

		_, _, err = c.PatchTektonPipelineWithContext(ctx, patchOptions)

		if err != nil {
			return diag.Errorf("Failed patching tekton pipeline: %s", err)
		}
	}

	return resourceOpenToolchainPipelineTriggersRead(ctx, d, m)
}

func resourceOpenToolchainPipelineTriggersDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// TODO: add restore to original
	d.SetId("")
	return diags
}

func createTriggerPatch(triggers []interface{}, currentPipelineTriggers []oc.TektonPipelineTrigger) []oc.TektonPipelineTrigger {
	if currentPipelineTriggers == nil {
		return nil
	}

	var result []oc.TektonPipelineTrigger
	triggerMap := make(map[string]map[string]interface{})

	for _, t := range triggers {
		tMap := t.(map[string]interface{})
		triggerMap[tMap["name"].(string)] = tMap
	}

	for _, t := range currentPipelineTriggers {
		if existing, ok := triggerMap[*t.Name]; ok {
			t.Disabled = getBoolPtr(!existing["enabled"].(bool))
			pattern := existing["pattern"].(string)
			branch := existing["branch"].(string)

			// we should not be setting branch or pattern for non-scm triggers
			// this should be sufficient check
			if t.ScmSource != nil {
				if pattern != "" && branch != "" {
					log.Printf("[WARN] Both trigger (%s) branch and pattern were set, branch setting will be used: %s", *t.Name, branch)
				}

				if pattern != "" {
					t.ScmSource.Pattern = getStringPtr(pattern)
					t.ScmSource.Branch = nil
				}

				if branch != "" {
					t.ScmSource.Branch = getStringPtr(branch)
					t.ScmSource.Pattern = nil
				}
			} else {
				log.Printf("[WARN] Trying to set branch/pattern for non scm trigger: %s", *t.Name)
			}
		}

		result = append(result, t)
	}

	return result
}

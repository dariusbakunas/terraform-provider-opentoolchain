package opentoolchain

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	oc "github.ibm.com/dbakuna/opentoolchain-go-sdk/opentoolchainv1"
)

func dataSourceOpenToolchainToolchain() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOpenToolchainToolchainRead,
		Schema: map[string]*schema.Schema{
			"guid": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"env_id": {
				Description: "Environment ID, example: `ibm:yp:us-south`",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "Toolchain name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"description": {
				Description: "Toolchain description",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"key": {
				Description: "Toolchain key",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"template": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"getting_started": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"services_total": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"url": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"source": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"locale": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"services": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"broker_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"service_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"lifecycle_messaging_webhook_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceOpenToolchainToolchainRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	c := m.(*oc.OpenToolchainV1)

	toolchain, _, err := c.GetToolchainWithContext(ctx, &oc.GetToolchainOptions{
		GUID:  getStringPtr(guid),
		EnvID: getStringPtr(envID),
	})

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Read toolchain: %+v", toolchain)

	d.Set("name", *toolchain.Name)
	d.Set("description", *toolchain.Description)
	d.Set("key", *toolchain.Key)
	d.Set("template", flattenToolchainTemplate(toolchain.Template))
	d.Set("services", flattenToolchainServices(toolchain.Services))
	d.Set("tags", toolchain.Tags)
	d.Set("lifecycle_messaging_webhook_id", *toolchain.LifecycleMessagingWebhookID)

	d.SetId(*toolchain.ToolchainGUID)
	return diags
}

func flattenToolchainTemplate(tpl *oc.ToolchainTemplate) []interface{} {
	if tpl == nil {
		return []interface{}{}
	}

	mTpl := map[string]interface{}{}
	mTpl["getting_started"] = *tpl.GettingStarted
	mTpl["services_total"] = *tpl.ServicesTotal
	mTpl["name"] = *tpl.Name
	mTpl["type"] = *tpl.Type
	mTpl["url"] = *tpl.URL
	mTpl["source"] = *tpl.Source
	mTpl["locale"] = *tpl.Locale

	return []interface{}{mTpl}
}

func flattenToolchainServices(svcs []oc.Service) []interface{} {
	var s []interface{}

	for _, svc := range svcs {
		service := map[string]interface{}{
			"broker_id":  *svc.BrokerID,
			"service_id": *svc.ServiceID,
		}

		s = append(s, service)
	}

	return s
}

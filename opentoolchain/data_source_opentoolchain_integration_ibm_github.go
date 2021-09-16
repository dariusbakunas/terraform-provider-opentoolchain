package opentoolchain

import (
	"context"
	"fmt"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenToolchainIntegrationIBMGithub() *schema.Resource {
	return &schema.Resource{
		Description: "Get IBM Github integration information (WARN: using undocumented APIs)",
		ReadContext: dataSourceOpenToolchainIntegrationIBMGithubRead,
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
			"repo_url": {
				Description: "Github repository url",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"private": {
				Description: "`true` if repository is private",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"enable_issues": {
				Description: "`true` if lightweight issue tracking is enabled",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"enable_traceability": {
				Description: "`true` if tracking for deployment of code changes is enabled",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func dataSourceOpenToolchainIntegrationIBMGithubRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
		return diag.Errorf("Error reading github service instance: %s", err)
	}

	if svc.ServiceInstance != nil && svc.ServiceInstance.Parameters != nil {
		params := svc.ServiceInstance.Parameters

		if u, ok := params["repo_url"]; ok {
			url := u.(string)
			d.Set("repo_url", url)
		}

		if p, ok := params["private_repo"]; ok {
			private := p.(bool)
			d.Set("private", private)
		}

		if h, ok := params["has_issues"]; ok {
			hasIssues := h.(bool)
			d.Set("enable_issues", hasIssues)
		}

		if e, ok := params["enable_traceability"]; ok {
			enableTraceability := e.(bool)
			d.Set("enable_traceability", enableTraceability)
		}
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", integrationID, toolchainID, envID))

	return nil
}

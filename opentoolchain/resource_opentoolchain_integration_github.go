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
	githubIntegrationServiceType = "github_integrated"
)

func resourceOpenToolchainIntegrationGithub() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage IBM Github integration (WARN: using undocumented APIs)",
		CreateContext: resourceOpenToolchainIntegrationGithubCreate,
		ReadContext:   resourceOpenToolchainIntegrationGithubRead,
		DeleteContext: resourceOpenToolchainIntegrationGithubDelete,
		UpdateContext: resourceOpenToolchainIntegrationGithubUpdate,
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
			"repo_url": {
				Description: "Github repository url",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == new {
						return true
					}
					if strings.TrimSuffix(old, ".git") == strings.TrimSuffix(new, ".git") {
						return true
					}
					return false
				},
			},
			"private": {
				Description: "Set `true` if repository is private",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"enable_issues": {
				Description: "Enable lightweight issue tracking",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"enable_traceability": {
				Description: "Track deployment of code changes",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func resourceOpenToolchainIntegrationGithubCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)
	repoURL := d.Get("repo_url").(string)
	private := d.Get("private").(bool)
	enableIssues := d.Get("enable_issues").(bool)
	enableTraceability := d.Get("enable_traceability").(bool)

	config := m.(*ProviderConfig)
	c := config.OTClient

	integrationUUID := uuid.NewString()
	uuidURL := fmt.Sprintf("%s/%s", repoURL, integrationUUID)

	options := &oc.CreateServiceInstanceOptions{
		ToolchainID: &toolchainID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(githubIntegrationServiceType),
		Parameters: &oc.CreateServiceInstanceParamsParameters{
			Authorized:         getStringPtr("integrated"),
			GitID:              getStringPtr("integrated"),
			Legal:              getBoolPtr(true),
			RepoURL:            &uuidURL,
			Type:               getStringPtr("link"),
			PrivateRepo:        &private,
			HasIssues:          &enableIssues,
			EnableTraceability: &enableTraceability,
		},
	}

	_, _, err := c.CreateServiceInstanceWithContext(ctx, options)

	if err != nil {
		return diag.Errorf("Error creating Github integration: %s", err)
	}

	toolchain, _, err := c.GetToolchainWithContext(ctx, &oc.GetToolchainOptions{
		GUID:  &toolchainID,
		EnvID: &envID,
	})

	if err != nil {
		return diag.Errorf("Error reading toolchain: %s", err)
	}

	var instanceID string

	// find new service instance
	if toolchain.Services != nil {
		for _, v := range toolchain.Services {
			if v.ServiceID != nil && *v.ServiceID == githubIntegrationServiceType && v.Parameters != nil && v.Parameters["repo_url"] == uuidURL && v.InstanceID != nil {
				instanceID = *v.InstanceID
				break
			}
		}
	}

	if instanceID == "" {
		// no way to cleanup since we don't know pipeline GUID
		return diag.Errorf("Unable to determine Github integration GUID")
	}

	_, err = c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
		ToolchainID: &toolchainID,
		GUID:        &instanceID,
		EnvID:       &envID,
		ServiceID:   getStringPtr(githubIntegrationServiceType),
		Parameters: &oc.PatchServiceInstanceParamsParameters{
			RepoURL: &repoURL,
		},
	})

	if err != nil {
		return diag.Errorf("Unable to update Github URL: %s", err)
	}

	d.Set("integration_id", instanceID)
	d.SetId(fmt.Sprintf("%s/%s/%s", instanceID, toolchainID, envID))

	return resourceOpenToolchainIntegrationGithubRead(ctx, d, m)
}

func resourceOpenToolchainIntegrationGithubRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	idParts := strings.Split(id, "/")

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
			log.Printf("[WARN] Github service instance '%s' is not found, removing it from state", integrationID)
			d.SetId("")
			return nil
		}

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

	return nil
}

func resourceOpenToolchainIntegrationGithubDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

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
		return diag.Errorf("Error deleting Github integration: %s", err)
	}

	d.SetId("")
	return diags
}

func resourceOpenToolchainIntegrationGithubUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	instanceID := d.Get("integration_id").(string)
	envID := d.Get("env_id").(string)
	toolchainID := d.Get("toolchain_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	if d.HasChange("private") || d.HasChange("enable_issues") || d.HasChange("enable_traceability") {
		private := d.Get("private").(bool)
		enableIssues := d.Get("enable_issues").(bool)
		enableTraceability := d.Get("enable_traceability").(bool)

		_, err := c.PatchServiceInstanceWithContext(ctx, &oc.PatchServiceInstanceOptions{
			ToolchainID: &toolchainID,
			GUID:        &instanceID,
			EnvID:       &envID,
			ServiceID:   getStringPtr(githubIntegrationServiceType),
			Parameters: &oc.PatchServiceInstanceParamsParameters{
				PrivateRepo:        &private,
				HasIssues:          &enableIssues,
				EnableTraceability: &enableTraceability,
			},
		})

		if err != nil {
			return diag.Errorf("Unable to update Github integration: %s", err)
		}
	}

	return resourceOpenToolchainIntegrationGithubRead(ctx, d, m)
}

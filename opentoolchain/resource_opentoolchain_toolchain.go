package opentoolchain

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOpenToolchainToolchain() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOpenToolchainToolchainCreate,
		ReadContext:   resourceOpenToolchainToolchainRead,
		DeleteContext: resourceOpenToolchainToolchainDelete,
		UpdateContext: resourceOpenToolchainToolchainUpdate,
		Schema: map[string]*schema.Schema{
			"guid": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"crn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"env_id": {
				Description: "Environment ID, example: `ibm:yp:us-south`",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"template_branch": {
				Description: "The Git branch name that the template will be read from",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"template_repository": {
				Description: "The Git repository that the template will be read from (leave empty if using without the template)",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "https://github.com/open-toolchain/empty-toolchain",
			},
			"repository_token": {
				Description: "If you are using a private GitHub or GitLab repository to host your template repo you will need to provide a personal access token",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"resource_group_id": {
				Description: "The GUID of resource group where toolchain will be created.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Description:  "Toolchain name",
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9-,._]+$`), "must contain only letters, numbers, hyphens, ',', '.' or '_'"),
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
			"template_properties": {
				Description: "Additional properties that are used by the template (leave empty if using without the template)",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				ForceNew: true,
			},
			"url": {
				Description: "Toolchain URL",
				Type:        schema.TypeString,
				Computed:    true,
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
						"instance_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"parameters": {
							Type: schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed: true,
						},
					},
				},
			},
			"tags": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
		},
	}
}

func resourceOpenToolchainToolchainRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	guid := d.Id()
	envID := d.Get("env_id").(string)

	envIDParts := strings.Split(envID, ":")
	region := envIDParts[len(envIDParts)-1]

	config := m.(*ProviderConfig)
	c := config.OTClient

	response, _, err := c.GetToolchainWithContext(ctx, &oc.GetToolchainOptions{
		GUID:    getStringPtr(guid),
		Region:  &region,
		Include: getStringPtr("fields,services"),
	})

	if err != nil {
		return diag.Errorf("Error reading toolchain: %s", err)
	}

	if len(response.Items) == 0 {
		return diag.Errorf("No toolchain found with GUID: %s", guid)
	}

	toolchain := response.Items[0]

	log.Printf("[DEBUG] Read toolchain: %+v", toolchain)

	d.Set("name", *toolchain.Name)
	d.Set("description", *toolchain.Description)
	d.Set("key", *toolchain.Key)
	d.Set("crn", *toolchain.CRN)
	d.Set("services", flattenToolchainServices(toolchain.Services))
	//d.Set("template", flattenToolchainTemplate(toolchain.Template))
	// d.Set("lifecycle_messaging_webhook_id", *toolchain.LifecycleMessagingWebhookID)

	u, err := url.Parse("https://cloud.ibm.com")

	if err != nil {
		return diag.Errorf("Unable to parse base service url: %s", err)
	}

	u.Path = path.Join(u.Path, fmt.Sprintf("/devops/toolchains/%s", *toolchain.ToolchainGUID))

	d.Set("url", fmt.Sprintf("%s?env_id=%s", u.String(), envID))
	return diags
}

func resourceOpenToolchainToolchainCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)

	envIDParts := strings.Split(envID, ":")
	region := envIDParts[len(envIDParts)-1]

	input := &oc.CreateToolchainOptions{
		EnvID:           getStringPtr(envID),
		Autocreate:      getBoolPtr(true),
		Repository:      getStringPtr(d.Get("template_repository").(string)),
		ResourceGroupID: getStringPtr(d.Get("resource_group_id").(string)),
	}

	config := m.(*ProviderConfig)
	c := config.OTClient
	t := config.TagClient

	if branch, ok := d.GetOk("template_branch"); ok {
		input.Branch = getStringPtr(branch.(string))
	}

	if repositoryToken, ok := d.GetOk("repository_token"); ok {
		input.SetProperty("repository_token", repositoryToken.(string))
	}

	if tplProps, ok := d.GetOk("template_properties"); ok {
		props := tplProps.(map[string]interface{})
		reserved := map[string]bool{
			"env_id":              true,
			"autocreate":          true,
			"template_repository": true,
			//"repository_token":    true, // TODO: uncomment on next major release (breaking change)
			"template_branch":   true,
			"resource_group_id": true,
			"name":              true,
		}

		for k, v := range props {
			if !reserved[k] {
				input.SetProperty(k, v)
			}
		}
	}

	log.Printf("[DEBUG] Creating toolchain: %+v", input)

	resp, err := c.CreateToolchainWithContext(ctx, input)

	if err != nil && resp.StatusCode != 302 {
		if result, ok := resp.GetResultAsMap(); ok {
			errDetails := ""

			if description, ok := result["description"]; ok {
				errDetails = description.(string)
			}

			if details, ok := result["details"]; ok {
				errDetails = errDetails + " " + details.(string)
			}

			return diag.Errorf("Error creating toolchain: %s - %s", err, errDetails)
		} else {
			return diag.Errorf("Error creating toolchain: %s", err)
		}
	}

	location := resp.Headers.Get("Location")

	if location == "" {
		return diag.Errorf("Failed getting new toolchain GUID, location header missing")
	}

	guid := extractGuid(location)
	d.Set("guid", guid)

	// TODO: this will need envID for import support (breaking change)
	d.SetId(guid)

	if name, ok := d.GetOk("name"); ok {
		// name was specified, try to use patch method to update it
		_, err := c.PatchToolchainWithContext(ctx, &oc.PatchToolchainOptions{
			Region: &region,
			GUID:   &guid,
			Name:   getStringPtr(name.(string)),
		})

		if err != nil {
			log.Printf("[WARN] Failed to update toolchain name: %s", err)

			// try to cleanup
			_, deleteErr := c.DeleteToolchainWithContext(ctx, &oc.DeleteToolchainOptions{
				Region: &region,
				GUID:   &guid,
			})

			if deleteErr != nil {
				return diag.Errorf("Failed to update toolchain name, unable to cleanup: %s", deleteErr)
			}

			return diag.Errorf("Failed to update toolchain name: %s", err)
		}
	}

	tags := expandStringList(d.Get("tags").(*schema.Set).List())
	crn, err := getCRN(ctx, d, m)

	if err != nil {
		return diag.Errorf("Error reading toolchain CRN: %s", err)
	}

	if len(tags) > 0 {
		log.Printf("[DEBUG] Setting toolchain tags: %v, %s", tags, crn)
		_, resp, err = t.AttachTagWithContext(ctx, &globaltaggingv1.AttachTagOptions{
			Resources: []globaltaggingv1.Resource{
				{ResourceID: getStringPtr(crn)},
			},
			TagNames: tags,
		})

		if err != nil {
			log.Printf("[DEBUG] Error setting toolchain tags: %s", resp)
			return diag.Errorf("Error setting toolchain tags: %s", err)
		}
	}

	return resourceOpenToolchainToolchainRead(ctx, d, m)
}

func getCRN(ctx context.Context, d *schema.ResourceData, m interface{}) (string, error) {
	guid := d.Get("guid").(string)
	envID := d.Get("env_id").(string)

	config := m.(*ProviderConfig)
	c := config.OTClient

	envIDParts := strings.Split(envID, ":")
	region := envIDParts[len(envIDParts)-1]

	response, _, err := c.GetToolchainWithContext(ctx, &oc.GetToolchainOptions{
		GUID:    getStringPtr(guid),
		Region:  &region,
		Include: getStringPtr("fields"),
	})

	if err != nil {
		return "", err
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no toolchain found with GUID: %s", guid)
	}

	toolchain := response.Items[0]

	log.Printf("[DEBUG] Read toolchain: %+v", toolchain)
	return *toolchain.CRN, nil
}

func extractGuid(location string) string {
	re := regexp.MustCompile(`\/(?P<guid>[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12})`)
	match := re.FindStringSubmatch(location)
	result := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return result["guid"]
}

func resourceOpenToolchainToolchainDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	envID := d.Get("env_id").(string)
	guid := d.Get("guid").(string)
	config := m.(*ProviderConfig)
	c := config.OTClient

	envIDParts := strings.Split(envID, ":")
	region := envIDParts[len(envIDParts)-1]

	log.Printf("[DEBUG] Deleting toolchain: %s", d.Id())

	_, err := c.DeleteToolchainWithContext(ctx, &oc.DeleteToolchainOptions{
		Region: &region,
		GUID:   &guid,
	})

	if err != nil {
		return diag.Errorf("Error deleting toolchain: %s", err)
	}

	return diags
}

func resourceOpenToolchainToolchainUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	guid := d.Get("guid").(string)
	crn := d.Get("crn").(string)

	envIDParts := strings.Split(envID, ":")
	region := envIDParts[len(envIDParts)-1]

	config := m.(*ProviderConfig)
	c := config.OTClient
	t := config.TagClient

	if d.HasChange("name") {
		name := d.Get("name")

		_, err := c.PatchToolchainWithContext(ctx, &oc.PatchToolchainOptions{
			Region: &region,
			GUID:   &guid,
			Name:   getStringPtr(name.(string)),
		})

		if err != nil {
			return diag.Errorf("Error updating toolchain nane: %s", err)
		}
	}

	if d.HasChange("tags") {
		o, n := d.GetChange("tags")

		oldTags := expandStringList(o.(*schema.Set).List())
		newTags := expandStringList(n.(*schema.Set).List())

		removed, added := sliceDiff(oldTags, newTags)

		if len(added) > 0 {
			log.Printf("[DEBUG] Adding toolchain tags: %v, %s", added, crn)
			_, resp, err := t.AttachTagWithContext(ctx, &globaltaggingv1.AttachTagOptions{
				Resources: []globaltaggingv1.Resource{
					{ResourceID: getStringPtr(crn)},
				},
				TagNames: added,
			})

			if err != nil {
				log.Printf("[DEBUG] Error setting toolchain tags: %s", resp)
				return diag.Errorf("Error setting toolchain tags: %s", err)
			}
		}

		if len(removed) > 0 {
			log.Printf("[DEBUG] Removing toolchain tags: %v, %s", removed, crn)
			_, resp, err := t.DetachTagWithContext(ctx, &globaltaggingv1.DetachTagOptions{
				Resources: []globaltaggingv1.Resource{
					{ResourceID: getStringPtr(crn)},
				},
				TagNames: removed,
			})

			if err != nil {
				log.Printf("[DEBUG] Error setting toolchain tags: %s", resp)
				return diag.Errorf("Error setting toolchain tags: %s", err)
			}
		}
	}

	return resourceOpenToolchainToolchainRead(ctx, d, m)
}

func sliceDiff(o, n []string) (removed, added []string) {
	oMap := make(map[string]bool)
	nMap := make(map[string]bool)

	for _, v := range o {
		oMap[v] = true
	}

	for _, v := range n {
		nMap[v] = true
	}

	for _, v := range n {
		if !oMap[v] {
			added = append(added, v)
		}
	}

	for _, v := range o {
		if !nMap[v] {
			removed = append(removed, v)
		}
	}

	return
}

package opentoolchain

import (
	"context"
	"log"
	"regexp"

	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOpenToolchainToolchain() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOpenToolchainToolchainCreate,
		ReadContext:   dataSourceOpenToolchainToolchainRead, // reusing data source read, same schema
		DeleteContext: resourceOpenToolchainToolchainDelete,
		UpdateContext: resourceOpenToolchainToolchainUpdate,
		Schema: map[string]*schema.Schema{
			"guid": {
				Description: "The toolchain `guid`",
				Type:        schema.TypeString,
				Computed:    true,
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
				Description: "The Git repository that the template will be read from",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "https://github.com/open-toolchain/empty-toolchain",
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
				Description: "Additional properties that are used by the template",
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
		},
	}
}

func resourceOpenToolchainToolchainCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)

	input := &oc.CreateToolchainOptions{
		EnvID:           getStringPtr(envID),
		Autocreate:      getBoolPtr(true),
		Repository:      getStringPtr(d.Get("template_repository").(string)),
		ResourceGroupID: getStringPtr(d.Get("resource_group_id").(string)),
	}

	c := m.(*oc.OpenToolchainV1)

	if branch, ok := d.GetOk("template_branch"); ok {
		input.Branch = getStringPtr(branch.(string))
	}

	if tplProps, ok := d.GetOk("template_properties"); ok {
		props := tplProps.(map[string]interface{})
		reserved := map[string]bool{
			"env_id":              true,
			"autocreate":          true,
			"template_repository": true,
			"template_branch":     true,
			"resource_group_id":   true,
			"name":                true,
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

	if name, ok := d.GetOk("name"); ok {
		// name was specified, try to use patch method to update it
		_, err := c.PatchToolchainWithContext(ctx, &oc.PatchToolchainOptions{
			EnvID: &envID,
			GUID:  &guid,
			Name:  getStringPtr(name.(string)),
		})

		if err != nil {
			log.Printf("[WARN] Failed to update toolchain name: %s", err)

			// try to cleanup
			_, deleteErr := c.DeleteToolchainWithContext(ctx, &oc.DeleteToolchainOptions{
				EnvID: getStringPtr(envID),
				GUID:  getStringPtr(guid),
			})

			if deleteErr != nil {
				return diag.Errorf("Failed to update toolchain name, unable to cleanup: %s", deleteErr)
			}

			return diag.Errorf("Failed to update toolchain name: %s", err)
		}
	}

	return dataSourceOpenToolchainToolchainRead(ctx, d, m)
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
	c := m.(*oc.OpenToolchainV1)

	log.Printf("[DEBUG] Deleting toolchain: %s", d.Id())

	_, err := c.DeleteToolchainWithContext(ctx, &oc.DeleteToolchainOptions{
		EnvID: getStringPtr(envID),
		GUID:  getStringPtr(guid),
	})

	if err != nil {
		return diag.Errorf("Error deleting toolchain: %s", err)
	}

	return diags
}

func resourceOpenToolchainToolchainUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	envID := d.Get("env_id").(string)
	guid := d.Get("guid").(string)
	c := m.(*oc.OpenToolchainV1)

	if d.HasChange("name") {
		name := d.Get("name")

		_, err := c.PatchToolchainWithContext(ctx, &oc.PatchToolchainOptions{
			EnvID: &envID,
			GUID:  &guid,
			Name:  getStringPtr(name.(string)),
		})

		if err != nil {
			return diag.Errorf("Error updating toolchain nane: %s", err)
		}
	}

	return dataSourceOpenToolchainToolchainRead(ctx, d, m)
}

package opentoolchain

import (
	"context"
	"net/url"
	"path"

	"github.com/IBM/go-sdk-core/core"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"iam_api_key": {
				Type:        schema.TypeString,
				Description: "The IBM Cloud IAM api key used to retrieve IAM access token if `iam_access_token` is not specified",
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("IAM_API_KEY", nil),
			},
			"iam_access_token": {
				Type:        schema.TypeString,
				Description: "The IBM Cloud Identity and Access Management token used to access Open Toolchain APIs",
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("IAM_ACCESS_TOKEN", nil),
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Open Toolchain API base URL (for example 'https://cloud.ibm.com')",
				Default:     "https://cloud.ibm.com",
			},
			"iam_base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "IBM IAM base URL",
				Default:     "https://iam.cloud.ibm.com",
			},
			"api_max_retry": {
				Description: "Maximum number of retries for AppID api requests, set to 0 to disable",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     3,
			},
		},
		ResourcesMap: map[string]*schema.Resource{},
		DataSourcesMap: map[string]*schema.Resource{
			"opentoolchain_toolchain": dataSourceOpenToolchainToolchain(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	var iamApiKey, iamAccessToken string

	if apiKey, ok := d.GetOk("iam_api_key"); ok {
		iamApiKey = apiKey.(string)
	}

	if accessToken, ok := d.GetOk("iam_access_token"); ok {
		iamAccessToken = accessToken.(string)
	}

	options := &oc.OpenToolchainV1Options{
		URL: d.Get("base_url").(string),
	}

	if iamAccessToken == "" {
		if iamApiKey == "" {
			return nil, diag.Errorf("iam_api_key or iam_access_token must be specified")
		}

		iamBaseURL := d.Get("iam_base_url").(string)

		u, err := url.Parse(iamBaseURL)

		if err != nil {
			return nil, diag.Errorf("failed parsing iam_base_url")
		}

		u.Path = path.Join(u.Path, "/identity/token")

		options.Authenticator = &core.IamAuthenticator{
			ApiKey: iamApiKey,
			URL:    u.String(),
		}
	} else {
		options.Authenticator = &core.BearerTokenAuthenticator{
			BearerToken: iamAccessToken,
		}
	}

	client, err := oc.NewOpenToolchainV1(options)

	if err != nil {
		return nil, diag.FromErr(err)
	}

	client.EnableRetries(d.Get("api_max_retry").(int), 0) // 0 delay - using client defaults

	return client, diags
}

package opentoolchain

import (
	"context"
	"net/http"
	"net/url"
	"path"

	"github.com/IBM/go-sdk-core/core"
	// v5core "github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	oc "github.com/dariusbakunas/opentoolchain-go-sdk/opentoolchainv1"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ProviderConfig struct {
	OTClient  *oc.OpenToolchainV1
	TagClient *globaltaggingv1.GlobalTaggingV1
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"iam_api_key": {
				Type:        schema.TypeString,
				Description: "The IBM Cloud IAM api key used to retrieve IAM access token if `iam_access_token` is not specified",
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"IC_API_KEY", "IBMCLOUD_API_KEY", "IAM_API_KEY"}, nil),
			},
			"iam_access_token": {
				Type:        schema.TypeString,
				Description: "The IBM Cloud Identity and Access Management token used to access Open Toolchain APIs",
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"IC_IAM_TOKEN", "IBMCLOUD_IAM_TOKEN", "IAM_ACCESS_TOKEN"}, nil),
			},
			"tags_base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Global Tagging service base URL",
				Default:     "https://tags.global-search-tagging.cloud.ibm.com",
			},
			"iam_base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "IBM IAM base URL",
				Default:     "https://iam.cloud.ibm.com",
			},
			// "api_max_retry": {
			// 	Description: "Maximum number of retries for AppID api requests, set to 0 to disable",
			// 	Type:        schema.TypeInt,
			// 	Optional:    true,
			// 	Default:     3,
			// },
		},
		ResourcesMap: map[string]*schema.Resource{
			"opentoolchain_integration_github":        resourceOpenToolchainIntegrationGithub(),
			"opentoolchain_integration_ibm_github":    resourceOpenToolchainIntegrationIBMGithub(),
			"opentoolchain_integration_keyprotect":    resourceOpenToolchainIntegrationKeyProtect(),
			"opentoolchain_integration_pagerduty":     resourceOpenToolchainIntegrationPagerDuty(),
			"opentoolchain_integration_slack":         resourceOpenToolchainIntegrationSlack(),
			"opentoolchain_toolchain":                 resourceOpenToolchainToolchain(),
			"opentoolchain_pipeline_properties":       resourceOpenToolchainPipelineProperties(),
			"opentoolchain_pipeline_triggers":         resourceOpenToolchainPipelineTriggers(),
			"opentoolchain_tekton_pipeline":           resourceOpenToolchainTektonPipeline(),
			"opentoolchain_tekton_pipeline_overrides": resourceOpenToolchainTektonPipelineOverrides(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"opentoolchain_toolchain":              dataSourceOpenToolchainToolchain(),
			"opentoolchain_integration_github":     dataSourceOpenToolchainIntegrationGithub(),
			"opentoolchain_integration_ibm_github": dataSourceOpenToolchainIntegrationIBMGithub(),
			"opentoolchain_integration_keyprotect": dataSourceOpenToolchainIntegrationKeyProtect(),
			"opentoolchain_integration_pagerduty":  dataSourceOpenToolchainIntegrationPagerDuty(),
			"opentoolchain_integration_slack":      dataSourceOpenToolchainIntegrationSlack(),
			"opentoolchain_pipeline_properties":    dataSourceOpenToolchainPipelineProperties(),
			"opentoolchain_pipeline_triggers":      dataSourceOpenToolchainPipelineTriggers(),
			"opentoolchain_tekton_pipeline":        dataSourceOpenToolchainTektonPipeline(),
			"opentoolchain_tekton_pipeline_config": dataSourceOpenToolchainTektonPipelineConfig(),
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

	otClientOptions := &oc.OpenToolchainV1Options{
		URL: "https://", // SDK appends 'devops-api.{region}.devops.cloud.ibm.com/v1/toolchains/{guid}' to all requests, due to openapi generator limitation, unable to override
	}

	tagClientOptions := &globaltaggingv1.GlobalTaggingV1Options{
		URL: d.Get("tags_base_url").(string),
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

		otClientOptions.Authenticator = &core.IamAuthenticator{
			ApiKey: iamApiKey,
			URL:    u.String(),
		}

		tagClientOptions.Authenticator = otClientOptions.Authenticator
	} else {
		otClientOptions.Authenticator = &core.BearerTokenAuthenticator{
			BearerToken: iamAccessToken,
		}

		tagClientOptions.Authenticator = otClientOptions.Authenticator
	}

	otClient, err := oc.NewOpenToolchainV1(otClientOptions)

	if err != nil {
		return nil, diag.FromErr(err)
	}

	// have to disable redirects, toolchain creation redirects to toolchain page
	httpClient := cleanhttp.DefaultPooledClient()
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// v5core.GetLogger().SetLogLevel(v5core.LevelDebug)

	otClient.Service.Client = httpClient

	// do not allow retries for reason above
	//client.EnableRetries(d.Get("api_max_retry").(int), 0) // 0 delay - using client defaults

	tagClient, err := globaltaggingv1.NewGlobalTaggingV1(tagClientOptions)

	if err != nil {
		return nil, diag.FromErr(err)
	}

	return &ProviderConfig{
		OTClient:  otClient,
		TagClient: tagClient,
	}, diags
}

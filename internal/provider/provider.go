package provider

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure AlphaKeyProvider satisfies various provider interfaces.
var _ provider.Provider = &AlphaKeyProvider{}

// AlphaKeyProvider defines the provider implementation.
type AlphaKeyProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// AlphaKeyProviderModel describes the provider data model.
type AlphaKeyProviderModel struct {
	BaseURL        types.String `tfsdk:"base_url"`
	APIToken       types.String `tfsdk:"api_token"`
	RequestTimeout types.Int64  `tfsdk:"request_timeout"`
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AlphaKeyProvider{
			version: version,
		}
	}
}

func (p *AlphaKeyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "alphakey"
	resp.Version = p.version
}

func (p *AlphaKeyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키(AlphaKey) IDaaS Terraform Provider",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Description: "알파키 API 기본 URL. 환경 변수 ALPHAKEY_BASE_URL로 대체 가능합니다.",
				Optional:    true,
			},
			"api_token": schema.StringAttribute{
				Description: "알파키 API 인증 토큰. 환경 변수 ALPHAKEY_API_TOKEN으로 대체 가능합니다.",
				Optional:    true,
				Sensitive:   true,
			},
			"request_timeout": schema.Int64Attribute{
				Description: "HTTP 요청 타임아웃(초). 기본값은 30초입니다.",
				Optional:    true,
			},
		},
	}
}

func (p *AlphaKeyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config AlphaKeyProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine base_url: Schema value takes priority, then environment variable
	baseURL := ""
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	} else {
		baseURL = os.Getenv("ALPHAKEY_BASE_URL")
	}

	if baseURL == "" {
		resp.Diagnostics.AddError(
			"알파키 API URL 누락",
			"base_url이 Provider 설정 또는 환경 변수 ALPHAKEY_BASE_URL에 지정되지 않았습니다. "+
				"Provider 블록에 base_url을 설정하거나 ALPHAKEY_BASE_URL 환경 변수를 설정해주세요.",
		)
	}

	// Determine api_token: Schema value takes priority, then environment variable
	apiToken := ""
	if !config.APIToken.IsNull() && !config.APIToken.IsUnknown() {
		apiToken = config.APIToken.ValueString()
	} else {
		apiToken = os.Getenv("ALPHAKEY_API_TOKEN")
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"알파키 API 토큰 누락",
			"api_token이 Provider 설정 또는 환경 변수 ALPHAKEY_API_TOKEN에 지정되지 않았습니다. "+
				"Provider 블록에 api_token을 설정하거나 ALPHAKEY_API_TOKEN 환경 변수를 설정해주세요.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine request timeout
	timeout := int64(30)
	if !config.RequestTimeout.IsNull() && !config.RequestTimeout.IsUnknown() {
		timeout = config.RequestTimeout.ValueInt64()
	}

	// Create AlphaKeyClient instance
	client := &AlphaKeyClient{
		BaseURL:  baseURL,
		APIToken: apiToken,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}

	// Make client available to resources and data sources
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *AlphaKeyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewAppResource,
		NewAppContractResource,
		NewUserGroupResource,
		NewAppGroupResource,
		NewUserAppResource,
		NewBlockedIPResource,
		NewAdminResource,
		NewMFAPolicyResource,
		NewPasswordExpiryResource,
	}
}

func (p *AlphaKeyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewUsersDataSource,
		NewAppDataSource,
		NewAppsDataSource,
		NewUserGroupsDataSource,
		NewAppGroupsDataSource,
		NewDepartmentsDataSource,
		NewAppCategoriesDataSource,
	}
}

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AppDataSource{}

type AppDataSource struct {
	client *AlphaKeyClient
}

type AppDataSourceModel struct {
	SaasID         types.String `tfsdk:"saas_id"`
	SaasName       types.String `tfsdk:"saas_name"`
	CategoryName   types.String `tfsdk:"category_name"`
	LinkYn         types.String `tfsdk:"link_yn"`
	LinkMethodName types.String `tfsdk:"link_method_name"`
	SaasURL        types.String `tfsdk:"saas_url"`
}

func NewAppDataSource() datasource.DataSource {
	return &AppDataSource{}
}

func (d *AppDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *AppDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱(SaaS) 정보를 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"saas_id": schema.StringAttribute{
				Description: "조회할 앱 ID",
				Required:    true,
			},
			"saas_name": schema.StringAttribute{
				Description: "앱 이름",
				Computed:    true,
			},
			"category_name": schema.StringAttribute{
				Description: "카테고리명",
				Computed:    true,
			},
			"link_yn": schema.StringAttribute{
				Description: "연동 여부",
				Computed:    true,
			},
			"link_method_name": schema.StringAttribute{
				Description: "연동 방식명",
				Computed:    true,
			},
			"saas_url": schema.StringAttribute{
				Description: "앱 URL",
				Computed:    true,
			},
		},
	}
}

func (d *AppDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*AlphaKeyClient)
	if !ok {
		resp.Diagnostics.AddError("잘못된 Provider 데이터", "예상하지 못한 Provider 데이터 타입입니다.")
		return
	}
	d.client = client
}

func (d *AppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AppDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]string{"saasId": state.SaasID.ValueString()}
	apiResp, err := d.client.Post(ctx, "/iam/v1/service/info/basic/detail", body)
	if err != nil {
		resp.Diagnostics.AddError("앱 조회 실패", err.Error())
		return
	}

	var data struct {
		SaasName       string `json:"saasName"`
		CategoryName   string `json:"categoryName"`
		LinkYn         string `json:"linkYn"`
		LinkMethodName string `json:"linkMethodName"`
		SaasURL        string `json:"saasUrl"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.SaasName = types.StringValue(data.SaasName)
	state.CategoryName = types.StringValue(data.CategoryName)
	state.LinkYn = types.StringValue(data.LinkYn)
	state.LinkMethodName = types.StringValue(data.LinkMethodName)
	state.SaasURL = types.StringValue(data.SaasURL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AppsDataSource{}

type AppsDataSource struct {
	client *AlphaKeyClient
}

type AppsDataSourceModel struct {
	Keyword types.String   `tfsdk:"keyword"`
	Apps    []AppItemModel `tfsdk:"apps"`
}

type AppItemModel struct {
	SaasID         types.String `tfsdk:"saas_id"`
	SaasName       types.String `tfsdk:"saas_name"`
	CategoryName   types.String `tfsdk:"category_name"`
	LinkYn         types.String `tfsdk:"link_yn"`
	LinkMethodName types.String `tfsdk:"link_method_name"`
	SaasURL        types.String `tfsdk:"saas_url"`
}

func NewAppsDataSource() datasource.DataSource {
	return &AppsDataSource{}
}

func (d *AppsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apps"
}

func (d *AppsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱(SaaS) 목록을 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"keyword": schema.StringAttribute{
				Description: "검색 키워드",
				Optional:    true,
			},
			"apps": schema.ListNestedAttribute{
				Description: "앱 목록",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"saas_id": schema.StringAttribute{
							Description: "앱 ID",
							Computed:    true,
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
				},
			},
		},
	}
}

func (d *AppsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AppsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{}
	if !state.Keyword.IsNull() && !state.Keyword.IsUnknown() {
		body["keyword"] = state.Keyword.ValueString()
	}

	apiResp, err := d.client.Post(ctx, "/iam/v1/service/list", body)
	if err != nil {
		resp.Diagnostics.AddError("앱 목록 조회 실패", err.Error())
		return
	}

	var data struct {
		List []struct {
			SaasID         string `json:"saasId"`
			SaasName       string `json:"saasName"`
			CategoryName   string `json:"categoryName"`
			LinkYn         string `json:"linkYn"`
			LinkMethodName string `json:"linkMethodName"`
			SaasURL        string `json:"saasUrl"`
		} `json:"list"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.Apps = make([]AppItemModel, len(data.List))
	for i, a := range data.List {
		state.Apps[i] = AppItemModel{
			SaasID:         types.StringValue(a.SaasID),
			SaasName:       types.StringValue(a.SaasName),
			CategoryName:   types.StringValue(a.CategoryName),
			LinkYn:         types.StringValue(a.LinkYn),
			LinkMethodName: types.StringValue(a.LinkMethodName),
			SaasURL:        types.StringValue(a.SaasURL),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

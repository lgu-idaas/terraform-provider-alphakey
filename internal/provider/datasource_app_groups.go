package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AppGroupsDataSource{}

type AppGroupsDataSource struct {
	client *AlphaKeyClient
}

type AppGroupsDataSourceModel struct {
	Keyword types.String       `tfsdk:"keyword"`
	Groups  []AppGroupItemModel `tfsdk:"groups"`
}

type AppGroupItemModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Desc types.String `tfsdk:"desc"`
}

func NewAppGroupsDataSource() datasource.DataSource {
	return &AppGroupsDataSource{}
}

func (d *AppGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_groups"
}

func (d *AppGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱 그룹 목록을 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"keyword": schema.StringAttribute{
				Description: "검색 키워드",
				Optional:    true,
			},
			"groups": schema.ListNestedAttribute{
				Description: "앱 그룹 목록",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "그룹 ID",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "그룹명",
							Computed:    true,
						},
						"desc": schema.StringAttribute{
							Description: "그룹 설명",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *AppGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AppGroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{}
	if !state.Keyword.IsNull() && !state.Keyword.IsUnknown() {
		body["keyword"] = state.Keyword.ValueString()
	}

	apiResp, err := d.client.Post(ctx, "/iam/v1/service/group/list", body)
	if err != nil {
		resp.Diagnostics.AddError("앱 그룹 목록 조회 실패", err.Error())
		return
	}

	var data struct {
		List []struct {
			GroupID   string `json:"saasGroupId"`
			GroupName string `json:"saasGroupName"`
			GroupDesc string `json:"saasGroupDesc"`
		} `json:"list"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.Groups = make([]AppGroupItemModel, len(data.List))
	for i, g := range data.List {
		state.Groups[i] = AppGroupItemModel{
			ID:   types.StringValue(g.GroupID),
			Name: types.StringValue(g.GroupName),
			Desc: types.StringValue(g.GroupDesc),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

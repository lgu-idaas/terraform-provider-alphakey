package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UsersDataSource{}

type UsersDataSource struct {
	client *AlphaKeyClient
}

type UsersDataSourceModel struct {
	Keyword  types.String      `tfsdk:"keyword"`
	DeptName types.String      `tfsdk:"dept_name"`
	Users    []UserItemModel   `tfsdk:"users"`
}

type UserItemModel struct {
	UserID        types.String `tfsdk:"user_id"`
	UserName      types.String `tfsdk:"user_name"`
	DeptName      types.String `tfsdk:"dept_name"`
	Position      types.String `tfsdk:"position"`
	WorkStateCode types.String `tfsdk:"work_state_code"`
	UserStateCode types.String `tfsdk:"user_state_code"`
}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 사용자 목록을 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"keyword": schema.StringAttribute{
				Description: "검색 키워드 (이름/아이디/부서)",
				Optional:    true,
			},
			"dept_name": schema.StringAttribute{
				Description: "부서명 필터",
				Optional:    true,
			},
			"users": schema.ListNestedAttribute{
				Description: "사용자 목록",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.StringAttribute{
							Description: "사용자 ID",
							Computed:    true,
						},
						"user_name": schema.StringAttribute{
							Description: "사용자 이름",
							Computed:    true,
						},
						"dept_name": schema.StringAttribute{
							Description: "부서명",
							Computed:    true,
						},
						"position": schema.StringAttribute{
							Description: "직위",
							Computed:    true,
						},
						"work_state_code": schema.StringAttribute{
							Description: "재직 상태 코드",
							Computed:    true,
						},
						"user_state_code": schema.StringAttribute{
							Description: "사용자 상태 코드",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state UsersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{}
	if !state.Keyword.IsNull() && !state.Keyword.IsUnknown() {
		body["keyword"] = state.Keyword.ValueString()
	}
	if !state.DeptName.IsNull() && !state.DeptName.IsUnknown() {
		body["deptName"] = state.DeptName.ValueString()
	}

	apiResp, err := d.client.Post(ctx, "/iam/v1/user/list", body)
	if err != nil {
		resp.Diagnostics.AddError("사용자 목록 조회 실패", err.Error())
		return
	}

	var data struct {
		Users []struct {
			UserID        string `json:"userId"`
			UserName      string `json:"userName"`
			DeptName      string `json:"deptName"`
			Position      string `json:"position"`
			WorkStateCode string `json:"workStateCode"`
			UserStateCode string `json:"userStateCode"`
		} `json:"users"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.Users = make([]UserItemModel, len(data.Users))
	for i, u := range data.Users {
		state.Users[i] = UserItemModel{
			UserID:        types.StringValue(u.UserID),
			UserName:      types.StringValue(u.UserName),
			DeptName:      types.StringValue(u.DeptName),
			Position:      types.StringValue(u.Position),
			WorkStateCode: types.StringValue(u.WorkStateCode),
			UserStateCode: types.StringValue(u.UserStateCode),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}

type UserDataSource struct {
	client *AlphaKeyClient
}

type UserDataSourceModel struct {
	UserID        types.String `tfsdk:"user_id"`
	UserName      types.String `tfsdk:"user_name"`
	DeptName      types.String `tfsdk:"dept_name"`
	Position      types.String `tfsdk:"position"`
	WorkStateCode types.String `tfsdk:"work_state_code"`
	UserStateCode types.String `tfsdk:"user_state_code"`
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 사용자 정보를 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "조회할 사용자 ID",
				Required:    true,
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
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]string{"userId": state.UserID.ValueString()}
	apiResp, err := d.client.Post(ctx, "/iam/v1/user/info/basic/detail", body)
	if err != nil {
		resp.Diagnostics.AddError("사용자 조회 실패", err.Error())
		return
	}

	var data struct {
		UserName      string `json:"userName"`
		DeptName      string `json:"deptName"`
		Position      string `json:"position"`
		WorkStateCode string `json:"workStateCode"`
		UserStateCode string `json:"userStateCode"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.UserName = types.StringValue(data.UserName)
	state.DeptName = types.StringValue(data.DeptName)
	state.Position = types.StringValue(data.Position)
	state.WorkStateCode = types.StringValue(data.WorkStateCode)
	state.UserStateCode = types.StringValue(data.UserStateCode)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

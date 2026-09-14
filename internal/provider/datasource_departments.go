package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DepartmentsDataSource{}

type DepartmentsDataSource struct {
	client *AlphaKeyClient
}

type DepartmentsDataSourceModel struct {
	Keyword     types.String         `tfsdk:"keyword"`
	Departments []DepartmentItemModel `tfsdk:"departments"`
}

type DepartmentItemModel struct {
	DeptID   types.String `tfsdk:"dept_id"`
	DeptName types.String `tfsdk:"dept_name"`
}

func NewDepartmentsDataSource() datasource.DataSource {
	return &DepartmentsDataSource{}
}

func (d *DepartmentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_departments"
}

func (d *DepartmentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 부서 목록을 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"keyword": schema.StringAttribute{
				Description: "검색 키워드",
				Optional:    true,
			},
			"departments": schema.ListNestedAttribute{
				Description: "부서 목록",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"dept_id": schema.StringAttribute{
							Description: "부서 ID",
							Computed:    true,
						},
						"dept_name": schema.StringAttribute{
							Description: "부서명",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DepartmentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DepartmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DepartmentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{}
	if !state.Keyword.IsNull() && !state.Keyword.IsUnknown() {
		body["keyword"] = state.Keyword.ValueString()
	}

	apiResp, err := d.client.Post(ctx, "/iam/v1/user/dept/tree/list", body)
	if err != nil {
		resp.Diagnostics.AddError("부서 목록 조회 실패", err.Error())
		return
	}

	var data struct {
		List []struct {
			DeptID   string `json:"deptId"`
			DeptName string `json:"deptName"`
		} `json:"list"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.Departments = make([]DepartmentItemModel, len(data.List))
	for i, dept := range data.List {
		state.Departments[i] = DepartmentItemModel{
			DeptID:   types.StringValue(dept.DeptID),
			DeptName: types.StringValue(dept.DeptName),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

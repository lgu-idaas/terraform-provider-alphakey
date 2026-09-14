package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AppCategoriesDataSource{}

type AppCategoriesDataSource struct {
	client *AlphaKeyClient
}

type AppCategoriesDataSourceModel struct {
	Categories []AppCategoryItemModel `tfsdk:"categories"`
}

type AppCategoryItemModel struct {
	CategoryID   types.String `tfsdk:"category_id"`
	CategoryName types.String `tfsdk:"category_name"`
}

func NewAppCategoriesDataSource() datasource.DataSource {
	return &AppCategoriesDataSource{}
}

func (d *AppCategoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_categories"
}

func (d *AppCategoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱 카테고리 목록을 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"categories": schema.ListNestedAttribute{
				Description: "앱 카테고리 목록",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"category_id": schema.StringAttribute{
							Description: "카테고리 ID",
							Computed:    true,
						},
						"category_name": schema.StringAttribute{
							Description: "카테고리명",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *AppCategoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppCategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AppCategoriesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := d.client.Post(ctx, "/iam/v1/service/category/list", nil)
	if err != nil {
		resp.Diagnostics.AddError("앱 카테고리 목록 조회 실패", err.Error())
		return
	}

	var data struct {
		List []struct {
			CategoryID   string `json:"categoryId"`
			CategoryName string `json:"categoryName"`
		} `json:"list"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.Categories = make([]AppCategoryItemModel, len(data.List))
	for i, cat := range data.List {
		state.Categories[i] = AppCategoryItemModel{
			CategoryID:   types.StringValue(cat.CategoryID),
			CategoryName: types.StringValue(cat.CategoryName),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

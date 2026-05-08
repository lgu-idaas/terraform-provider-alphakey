package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &AppResource{}
	_ resource.ResourceWithImportState = &AppResource{}
)

type AppResource struct {
	client *AlphaKeyClient
}

type AppResourceModel struct {
	ID             types.String `tfsdk:"id"`
	LgSaasID      types.String `tfsdk:"lg_saas_id"`
	SaasName       types.String `tfsdk:"saas_name"`
	CategoryName   types.String `tfsdk:"category_name"`
	LinkYn         types.String `tfsdk:"link_yn"`
	LinkMethodName types.String `tfsdk:"link_method_name"`
}

func NewAppResource() resource.Resource {
	return &AppResource{}
}

func (r *AppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱(SaaS) 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "앱 고유 ID (API에서 자동 생성)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"lg_saas_id": schema.StringAttribute{
				Description: "LG SaaS ID",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"saas_name": schema.StringAttribute{
				Description: "SaaS 이름",
				Computed:    true,
			},
			"category_name": schema.StringAttribute{
				Description: "카테고리 이름",
				Computed:    true,
			},
			"link_yn": schema.StringAttribute{
				Description: "연동 여부",
				Computed:    true,
			},
			"link_method_name": schema.StringAttribute{
				Description: "연동 방식 이름",
				Computed:    true,
			},
		},
	}
}

func (r *AppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*AlphaKeyClient)
	if !ok {
		resp.Diagnostics.AddError("잘못된 Provider 데이터", "Provider에서 전달된 데이터가 *AlphaKeyClient 타입이 아닙니다.")
		return
	}
	r.client = client
}

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"lgSaasId": plan.LgSaasID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/service/create", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 생성 실패", err.Error())
		}
		return
	}

	var respData struct {
		SaasID string `json:"saasId"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "앱 생성 응답에서 saasId를 파싱할 수 없습니다: "+err.Error())
		return
	}

	plan.ID = types.StringValue(respData.SaasID)
	plan.SaasName = types.StringValue("")
	plan.CategoryName = types.StringValue("")
	plan.LinkYn = types.StringValue("")
	plan.LinkMethodName = types.StringValue("")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"saasId": state.ID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/service/info/basic/detail", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		SaasID         string `json:"saasId"`
		LgSaasID       string `json:"lgSaasId"`
		SaasName       string `json:"saasName"`
		CategoryName   string `json:"categoryName"`
		LinkYn         string `json:"linkYn"`
		LinkMethodName string `json:"linkMethodName"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "앱 조회 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	state.ID = types.StringValue(respData.SaasID)
	state.LgSaasID = types.StringValue(respData.LgSaasID)
	state.SaasName = types.StringValue(respData.SaasName)
	state.CategoryName = types.StringValue(respData.CategoryName)
	state.LinkYn = types.StringValue(respData.LinkYn)
	state.LinkMethodName = types.StringValue(respData.LinkMethodName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"saasId": state.ID.ValueString(),
	}

	_, err := r.client.Post(ctx, "/iam/v1/service/delete", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 삭제 실패", err.Error())
		}
		return
	}
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

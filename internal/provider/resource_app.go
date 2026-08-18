package provider

import (
	"context"
	"encoding/json"
	"fmt"

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
		"lgSaasIds": []string{plan.LgSaasID.ValueString()},
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

	// 응답: {"list": [{"saasId": "...", ...}]}
	var respData struct {
		List []struct {
			SaasID         string `json:"saasId"`
			SaasName       string `json:"saasName"`
			CategoryName   string `json:"categoryName"`
			LinkYn         string `json:"linkYn"`
			LinkMethodName string `json:"linkMethodName"`
		} `json:"list"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "앱 생성 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}
	if len(respData.List) == 0 {
		resp.Diagnostics.AddError("앱 생성 실패", "응답에 생성된 앱 정보가 없습니다.")
		return
	}

	app := respData.List[0]
	plan.ID = types.StringValue(app.SaasID)
	plan.SaasName = types.StringValue(app.SaasName)
	plan.CategoryName = types.StringValue(app.CategoryName)
	plan.LinkYn = types.StringValue(app.LinkYn)
	plan.LinkMethodName = types.StringValue(app.LinkMethodName)

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

	apiResp, err := r.client.Post(ctx, "/iam/v1/service/basic/detail", body)
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
		SaasAppID      string `json:"saasAppId"`
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
	if respData.SaasAppID != "" && respData.SaasAppID != "0" {
		state.LgSaasID = types.StringValue(respData.SaasAppID)
	}
	state.SaasName = types.StringValue(respData.SaasName)
	state.CategoryName = types.StringValue(respData.CategoryName)
	state.LinkYn = types.StringValue(respData.LinkYn)
	state.LinkMethodName = types.StringValue(respData.LinkMethodName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// lg_saas_id 변경 시 RequiresReplace이므로 Update는 호출되지 않음
	_ = fmt.Sprintf("no-op")
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

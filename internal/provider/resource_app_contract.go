package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &AppContractResource{}
)

type AppContractResource struct {
	client *AlphaKeyClient
}

type AppContractResourceModel struct {
	ID            types.String `tfsdk:"id"`
	SaasID        types.String `tfsdk:"saas_id"`
	UserID        types.String `tfsdk:"user_id"`
	SaasFreeYn    types.String `tfsdk:"saas_free_yn"`
	SaasStartDate types.String `tfsdk:"saas_start_date"`
	SaasEndDate   types.String `tfsdk:"saas_end_date"`
	EndAnnounceYn types.String `tfsdk:"end_announce_yn"`
	Price         types.String `tfsdk:"price"`
	PriceCode     types.String `tfsdk:"price_code"`
	PriceTypeCode types.String `tfsdk:"price_type_code"`
	UserCountYn   types.String `tfsdk:"user_count_yn"`
	SaasCount     types.Int64  `tfsdk:"saas_count"`
	Memo          types.String `tfsdk:"memo"`
}

func NewAppContractResource() resource.Resource {
	return &AppContractResource{}
}

func (r *AppContractResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_contract"
}

func (r *AppContractResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱 계약정보 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "리소스 ID (saas_id와 동일)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"saas_id": schema.StringAttribute{
				Description: "SaaS ID",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "담당자 사용자 ID",
				Required:    true,
			},
			"saas_free_yn": schema.StringAttribute{
				Description: "무료 여부 (Y/N)",
				Required:    true,
			},
			"saas_start_date": schema.StringAttribute{
				Description: "계약 시작일",
				Required:    true,
			},
			"saas_end_date": schema.StringAttribute{
				Description: "계약 종료일",
				Required:    true,
			},
			"end_announce_yn": schema.StringAttribute{
				Description: "만료 알림 여부 (Y/N)",
				Required:    true,
			},
			"price": schema.StringAttribute{
				Description: "가격",
				Optional:    true,
			},
			"price_code": schema.StringAttribute{
				Description: "가격 코드",
				Optional:    true,
			},
			"price_type_code": schema.StringAttribute{
				Description: "가격 유형 코드",
				Optional:    true,
			},
			"user_count_yn": schema.StringAttribute{
				Description: "사용자 수 제한 여부 (Y/N)",
				Optional:    true,
			},
			"saas_count": schema.Int64Attribute{
				Description: "SaaS 사용자 수",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"memo": schema.StringAttribute{
				Description: "메모",
				Optional:    true,
			},
		},
	}
}

func (r *AppContractResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppContractResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AppContractResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(&plan)

	_, err := r.client.Post(ctx, "/iam/v1/service/contract/create", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 계약정보 생성 실패", err.Error())
		}
		return
	}

	plan.ID = plan.SaasID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppContractResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AppContractResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"saasId": state.ID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/service/contract/detail", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 계약정보 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		SaasID        string `json:"saasId"`
		UserID        string `json:"userId"`
		SaasFreeYn    string `json:"saasFreeYn"`
		SaasStartDate string `json:"saasStartDate"`
		SaasEndDate   string `json:"saasEndDate"`
		EndAnnounceYn string `json:"endAnnounceYn"`
		Price         string `json:"price"`
		PriceCode     string `json:"priceCode"`
		PriceTypeCode string `json:"priceTypeCode"`
		UserCountYn   string `json:"userCountYn"`
		SaasCount     *int64 `json:"saasCount"`
		Memo          string `json:"memo"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "앱 계약정보 조회 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	state.ID = types.StringValue(respData.SaasID)
	state.SaasID = types.StringValue(respData.SaasID)
	state.UserID = types.StringValue(respData.UserID)
	state.SaasFreeYn = types.StringValue(respData.SaasFreeYn)
	state.SaasStartDate = types.StringValue(respData.SaasStartDate)
	state.SaasEndDate = types.StringValue(respData.SaasEndDate)
	state.EndAnnounceYn = types.StringValue(respData.EndAnnounceYn)

	if respData.Price != "" {
		state.Price = types.StringValue(respData.Price)
	} else {
		state.Price = types.StringNull()
	}
	if respData.PriceCode != "" {
		state.PriceCode = types.StringValue(respData.PriceCode)
	} else {
		state.PriceCode = types.StringNull()
	}
	if respData.PriceTypeCode != "" {
		state.PriceTypeCode = types.StringValue(respData.PriceTypeCode)
	} else {
		state.PriceTypeCode = types.StringNull()
	}
	if respData.UserCountYn != "" {
		state.UserCountYn = types.StringValue(respData.UserCountYn)
	} else {
		state.UserCountYn = types.StringNull()
	}
	if respData.SaasCount != nil {
		state.SaasCount = types.Int64Value(*respData.SaasCount)
	} else {
		state.SaasCount = types.Int64Null()
	}
	if respData.Memo != "" {
		state.Memo = types.StringValue(respData.Memo)
	} else {
		state.Memo = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AppContractResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AppContractResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(&plan)

	_, err := r.client.Post(ctx, "/iam/v1/service/contract/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 계약정보 수정 실패", err.Error())
		}
		return
	}

	plan.ID = plan.SaasID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppContractResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Contract is deleted when the app is deleted; no separate delete API
}

func (r *AppContractResource) buildBody(model *AppContractResourceModel) map[string]interface{} {
	body := map[string]interface{}{
		"saasId":        model.SaasID.ValueString(),
		"userId":        model.UserID.ValueString(),
		"saasFreeYn":    model.SaasFreeYn.ValueString(),
		"saasStartDate": model.SaasStartDate.ValueString(),
		"saasEndDate":   model.SaasEndDate.ValueString(),
		"endAnnounceYn": model.EndAnnounceYn.ValueString(),
	}

	if !model.Price.IsNull() && !model.Price.IsUnknown() {
		body["price"] = model.Price.ValueString()
	}
	if !model.PriceCode.IsNull() && !model.PriceCode.IsUnknown() {
		body["priceCode"] = model.PriceCode.ValueString()
	}
	if !model.PriceTypeCode.IsNull() && !model.PriceTypeCode.IsUnknown() {
		body["priceTypeCode"] = model.PriceTypeCode.ValueString()
	}
	if !model.UserCountYn.IsNull() && !model.UserCountYn.IsUnknown() {
		body["userCountYn"] = model.UserCountYn.ValueString()
	}
	if !model.SaasCount.IsNull() && !model.SaasCount.IsUnknown() {
		body["saasCount"] = model.SaasCount.ValueInt64()
	}
	if !model.Memo.IsNull() && !model.Memo.IsUnknown() {
		body["memo"] = model.Memo.ValueString()
	}

	return body
}

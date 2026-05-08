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
	_ resource.Resource                = &AppGroupResource{}
	_ resource.ResourceWithImportState = &AppGroupResource{}
)

type AppGroupResource struct {
	client *AlphaKeyClient
}

type AppGroupResourceModel struct {
	ID            types.String `tfsdk:"id"`
	SaasGroupName types.String `tfsdk:"saas_group_name"`
	SaasGroupDesc types.String `tfsdk:"saas_group_desc"`
	OfficerIDs    types.Set    `tfsdk:"officer_ids"`
	SaasIDs       types.Set    `tfsdk:"saas_ids"`
}

func NewAppGroupResource() resource.Resource {
	return &AppGroupResource{}
}

func (r *AppGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_group"
}

func (r *AppGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱 그룹 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "앱 그룹 고유 ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"saas_group_name": schema.StringAttribute{
				Description: "앱 그룹 이름",
				Required:    true,
			},
			"saas_group_desc": schema.StringAttribute{
				Description: "앱 그룹 설명",
				Required:    true,
			},
			"officer_ids": schema.SetAttribute{
				Description: "그룹 담당자 사용자 ID 목록",
				Optional:    true,
				ElementType: types.StringType,
			},
			"saas_ids": schema.SetAttribute{
				Description: "그룹에 포함된 SaaS ID 목록",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *AppGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AppGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"saasGroupName": plan.SaasGroupName.ValueString(),
		"saasGroupDesc": plan.SaasGroupDesc.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/service/group/create", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 그룹 생성 실패", err.Error())
		}
		return
	}

	var respData struct {
		SaasGroupID string `json:"saasGroupId"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "앱 그룹 생성 응답에서 saasGroupId를 파싱할 수 없습니다: "+err.Error())
		return
	}

	plan.ID = types.StringValue(respData.SaasGroupID)
	groupID := respData.SaasGroupID

	// Add officers
	officerIDs := extractStringSet(ctx, plan.OfficerIDs)
	for _, oid := range officerIDs {
		officerBody := map[string]interface{}{
			"saasGroupId": groupID,
			"userId":      oid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/group/officer/add", officerBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 그룹 담당자 추가 실패", err.Error())
			}
			return
		}
	}

	// Add saas
	saasIDs := extractStringSet(ctx, plan.SaasIDs)
	for _, sid := range saasIDs {
		saasBody := map[string]interface{}{
			"saasGroupId": groupID,
			"saasId":      sid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/group/saas/add", saasBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 그룹 SaaS 추가 실패", err.Error())
			}
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AppGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"saasGroupId": state.ID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/service/group/detail", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 그룹 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		SaasGroupID   string `json:"saasGroupId"`
		SaasGroupName string `json:"saasGroupName"`
		SaasGroupDesc string `json:"saasGroupDesc"`
		Officers      []struct {
			UserID string `json:"userId"`
		} `json:"officers"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "앱 그룹 조회 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	state.ID = types.StringValue(respData.SaasGroupID)
	state.SaasGroupName = types.StringValue(respData.SaasGroupName)
	state.SaasGroupDesc = types.StringValue(respData.SaasGroupDesc)

	var officerIDs []string
	for _, o := range respData.Officers {
		officerIDs = append(officerIDs, o.UserID)
	}
	state.OfficerIDs = buildStringSet(ctx, officerIDs)

	// Read saas list
	saasResp, err := r.client.Post(ctx, "/iam/v1/service/group/saas/list", body)
	if err == nil {
		var saasData struct {
			List []struct {
				SaasID string `json:"saasId"`
			} `json:"list"`
		}
		if json.Unmarshal(saasResp.Data, &saasData) == nil {
			var saasIDs []string
			for _, s := range saasData.List {
				saasIDs = append(saasIDs, s.SaasID)
			}
			state.SaasIDs = buildStringSet(ctx, saasIDs)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AppGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AppGroupResourceModel
	var state AppGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.ID.ValueString()

	// Update group info
	body := map[string]interface{}{
		"saasGroupId":   groupID,
		"saasGroupName": plan.SaasGroupName.ValueString(),
		"saasGroupDesc": plan.SaasGroupDesc.ValueString(),
	}

	_, err := r.client.Post(ctx, "/iam/v1/service/group/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 그룹 수정 실패", err.Error())
		}
		return
	}

	// Diff officer_ids
	oldOfficers := extractStringSet(ctx, state.OfficerIDs)
	newOfficers := extractStringSet(ctx, plan.OfficerIDs)
	toAddOfficers, toRemoveOfficers := diffSets(oldOfficers, newOfficers)
	for _, oid := range toAddOfficers {
		officerBody := map[string]interface{}{
			"saasGroupId": groupID,
			"userId":      oid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/group/officer/add", officerBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 그룹 담당자 추가 실패", err.Error())
			}
			return
		}
	}
	for _, oid := range toRemoveOfficers {
		officerBody := map[string]interface{}{
			"saasGroupId": groupID,
			"userId":      oid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/group/officer/remove", officerBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 그룹 담당자 제거 실패", err.Error())
			}
			return
		}
	}

	// Diff saas_ids
	oldSaas := extractStringSet(ctx, state.SaasIDs)
	newSaas := extractStringSet(ctx, plan.SaasIDs)
	toAddSaas, toRemoveSaas := diffSets(oldSaas, newSaas)
	for _, sid := range toAddSaas {
		saasBody := map[string]interface{}{
			"saasGroupId": groupID,
			"saasId":      sid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/group/saas/add", saasBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 그룹 SaaS 추가 실패", err.Error())
			}
			return
		}
	}
	for _, sid := range toRemoveSaas {
		saasBody := map[string]interface{}{
			"saasGroupId": groupID,
			"saasId":      sid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/group/saas/remove", saasBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 그룹 SaaS 제거 실패", err.Error())
			}
			return
		}
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AppGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"saasGroupId": state.ID.ValueString(),
	}

	_, err := r.client.Post(ctx, "/iam/v1/service/group/delete", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 그룹 삭제 실패", err.Error())
		}
		return
	}
}

func (r *AppGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &UserGroupResource{}
	_ resource.ResourceWithImportState = &UserGroupResource{}
)

type UserGroupResource struct {
	client *AlphaKeyClient
}

type UserGroupResourceModel struct {
	ID            types.String `tfsdk:"id"`
	UserGroupName types.String `tfsdk:"user_group_name"`
	UserGroupDesc types.String `tfsdk:"user_group_desc"`
	UserIDs       types.Set    `tfsdk:"user_ids"`
	OfficerIDs    types.Set    `tfsdk:"officer_ids"`
	SaasIDs       types.Set    `tfsdk:"saas_ids"`
}

func NewUserGroupResource() resource.Resource {
	return &UserGroupResource{}
}

func (r *UserGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (r *UserGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 사용자 그룹 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "사용자 그룹 고유 ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_group_name": schema.StringAttribute{
				Description: "사용자 그룹 이름",
				Required:    true,
			},
			"user_group_desc": schema.StringAttribute{
				Description: "사용자 그룹 설명",
				Optional:    true,
			},
			"user_ids": schema.SetAttribute{
				Description: "그룹 멤버 사용자 ID 목록",
				Optional:    true,
				ElementType: types.StringType,
			},
			"officer_ids": schema.SetAttribute{
				Description: "그룹 담당자 사용자 ID 목록",
				Optional:    true,
				ElementType: types.StringType,
			},
			"saas_ids": schema.SetAttribute{
				Description: "그룹에 연결된 SaaS ID 목록",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *UserGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"userGroupName": plan.UserGroupName.ValueString(),
	}
	if !plan.UserGroupDesc.IsNull() && !plan.UserGroupDesc.IsUnknown() {
		body["userGroupDesc"] = plan.UserGroupDesc.ValueString()
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/user/group/create", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 그룹 생성 실패", err.Error())
		}
		return
	}

	var respData struct {
		UserGroupID string `json:"userGroupId"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "사용자 그룹 생성 응답에서 userGroupId를 파싱할 수 없습니다: "+err.Error())
		return
	}

	plan.ID = types.StringValue(respData.UserGroupID)
	groupID := respData.UserGroupID

	// Add members (API requires userIds as array)
	userIDs := extractStringSet(ctx, plan.UserIDs)
	if len(userIDs) > 0 {
		memberBody := map[string]interface{}{
			"userGroupId": groupID,
			"userIds":     userIDs,
		}
		_, err := r.client.Post(ctx, "/iam/v1/user/group/member/add", memberBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("그룹 멤버 추가 실패", err.Error())
			}
			return
		}
	}

	// Add officers
	officerIDs := extractStringSet(ctx, plan.OfficerIDs)
	if len(officerIDs) > 0 {
		officerBody := map[string]interface{}{
			"userGroupId": groupID,
			"userIds":     officerIDs,
		}
		_, err := r.client.Post(ctx, "/iam/v1/user/group/member/add", officerBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("그룹 담당자 추가 실패", err.Error())
			}
			return
		}
	}

	// Add saas
	saasIDs := extractStringSet(ctx, plan.SaasIDs)
	for _, sid := range saasIDs {
		saasBody := map[string]interface{}{
			"userGroupId": groupID,
			"saasId":      sid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/user/group/saas/add", saasBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("그룹 SaaS 추가 실패", err.Error())
			}
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"userGroupId": state.ID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/user/group/detail", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 그룹 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		UserGroupID   string `json:"userGroupId"`
		UserGroupName string `json:"userGroupName"`
		UserGroupDesc string `json:"userGroupDesc"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "사용자 그룹 조회 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	state.ID = types.StringValue(respData.UserGroupID)
	state.UserGroupName = types.StringValue(respData.UserGroupName)
	if respData.UserGroupDesc != "" {
		state.UserGroupDesc = types.StringValue(respData.UserGroupDesc)
	} else {
		state.UserGroupDesc = types.StringNull()
	}

	// Read members
	memberResp, err := r.client.Post(ctx, "/iam/v1/user/group/member/list", body)
	if err == nil {
		var memberData struct {
			List []struct {
				UserID    string `json:"userId"`
				OfficerYn string `json:"officerYn"`
			} `json:"list"`
		}
		if json.Unmarshal(memberResp.Data, &memberData) == nil {
			var userIDs []string
			var officerIDs []string
			for _, m := range memberData.List {
				if m.OfficerYn == "Y" {
					officerIDs = append(officerIDs, m.UserID)
				} else {
					userIDs = append(userIDs, m.UserID)
				}
			}
			state.UserIDs = buildStringSet(ctx, userIDs)
			state.OfficerIDs = buildStringSet(ctx, officerIDs)
		}
	}

	// Read saas list
	saasResp, err := r.client.Post(ctx, "/iam/v1/user/group/saas/list", body)
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

func (r *UserGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserGroupResourceModel
	var state UserGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.ID.ValueString()

	// Update group info
	body := map[string]interface{}{
		"userGroupId":   groupID,
		"userGroupName": plan.UserGroupName.ValueString(),
	}
	if !plan.UserGroupDesc.IsNull() && !plan.UserGroupDesc.IsUnknown() {
		body["userGroupDesc"] = plan.UserGroupDesc.ValueString()
	}

	_, err := r.client.Post(ctx, "/iam/v1/user/group/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 그룹 수정 실패", err.Error())
		}
		return
	}

	// Diff user_ids
	oldUsers := extractStringSet(ctx, state.UserIDs)
	newUsers := extractStringSet(ctx, plan.UserIDs)
	r.diffGroupMembers(ctx, groupID, oldUsers, newUsers, "N", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Diff officer_ids
	oldOfficers := extractStringSet(ctx, state.OfficerIDs)
	newOfficers := extractStringSet(ctx, plan.OfficerIDs)
	r.diffGroupMembers(ctx, groupID, oldOfficers, newOfficers, "Y", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Diff saas_ids
	oldSaas := extractStringSet(ctx, state.SaasIDs)
	newSaas := extractStringSet(ctx, plan.SaasIDs)
	toAddSaas, toRemoveSaas := diffSets(oldSaas, newSaas)
	for _, sid := range toAddSaas {
		saasBody := map[string]interface{}{
			"userGroupId": groupID,
			"saasId":      sid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/user/group/saas/add", saasBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("그룹 SaaS 추가 실패", err.Error())
			}
			return
		}
	}
	for _, sid := range toRemoveSaas {
		saasBody := map[string]interface{}{
			"userGroupId": groupID,
			"saasId":      sid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/user/group/saas/remove", saasBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("그룹 SaaS 제거 실패", err.Error())
			}
			return
		}
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.ID.ValueString()

	// Remove all members first
	allMembers := extractStringSet(ctx, state.UserIDs)
	allMembers = append(allMembers, extractStringSet(ctx, state.OfficerIDs)...)
	for _, uid := range allMembers {
		memberBody := map[string]interface{}{
			"userGroupId": groupID,
			"userId":      uid,
		}
		_, _ = r.client.Post(ctx, "/iam/v1/user/group/member/remove", memberBody)
	}

	// Delete group
	body := map[string]interface{}{
		"userGroupId": groupID,
	}
	_, err := r.client.Post(ctx, "/iam/v1/user/group/delete", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 그룹 삭제 실패", err.Error())
		}
		return
	}
}

func (r *UserGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *UserGroupResource) diffGroupMembers(ctx context.Context, groupID string, oldIDs, newIDs []string, officerYn string, diags *diag.Diagnostics) {
	toAdd, toRemove := diffSets(oldIDs, newIDs)

	for _, uid := range toAdd {
		memberBody := map[string]interface{}{
			"userGroupId": groupID,
			"userId":      uid,
		}
		if officerYn == "Y" {
			memberBody["officerYn"] = "Y"
		}
		_, err := r.client.Post(ctx, "/iam/v1/user/group/member/add", memberBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				diags.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				diags.AddError("그룹 멤버 추가 실패", err.Error())
			}
			return
		}
	}

	for _, uid := range toRemove {
		memberBody := map[string]interface{}{
			"userGroupId": groupID,
			"userId":      uid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/user/group/member/remove", memberBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				diags.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				diags.AddError("그룹 멤버 제거 실패", err.Error())
			}
			return
		}
	}
}

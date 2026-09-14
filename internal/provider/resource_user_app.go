package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &UserAppResource{}
)

type UserAppResource struct {
	client *AlphaKeyClient
}

type UserAppResourceModel struct {
	ID      types.String `tfsdk:"id"`
	SaasID  types.String `tfsdk:"saas_id"`
	UserIDs types.Set    `tfsdk:"user_ids"`
}

func NewUserAppResource() resource.Resource {
	return &UserAppResource{}
}

func (r *UserAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_app"
}

func (r *UserAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 앱 사용자 권한 리소스를 관리합니다.",
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
			"user_ids": schema.SetAttribute{
				Description: "권한을 부여할 사용자 ID 목록",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *UserAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userIDs := extractStringSet(ctx, plan.UserIDs)

	body := map[string]interface{}{
		"saasId":  plan.SaasID.ValueString(),
		"userIds": userIDs,
	}

	_, err := r.client.Post(ctx, "/iam/v1/service/user/grant", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 사용자 권한 부여 실패", err.Error())
		}
		return
	}

	plan.ID = plan.SaasID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"saasId": state.ID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/service/user/list", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 사용자 목록 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		List []struct {
			UserID string `json:"userId"`
		} `json:"list"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "앱 사용자 목록 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	// Filter to only managed user_ids
	managedIDs := extractStringSet(ctx, state.UserIDs)
	managedMap := make(map[string]bool, len(managedIDs))
	for _, id := range managedIDs {
		managedMap[id] = true
	}

	var currentIDs []string
	for _, u := range respData.List {
		if managedMap[u.UserID] {
			currentIDs = append(currentIDs, u.UserID)
		}
	}

	state.UserIDs = buildStringSet(ctx, currentIDs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserAppResourceModel
	var state UserAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	saasID := state.ID.ValueString()
	oldIDs := extractStringSet(ctx, state.UserIDs)
	newIDs := extractStringSet(ctx, plan.UserIDs)
	toAdd, toRemove := diffSets(oldIDs, newIDs)

	if len(toAdd) > 0 {
		body := map[string]interface{}{
			"saasId":  saasID,
			"userIds": toAdd,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/user/grant", body)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 사용자 권한 부여 실패", err.Error())
			}
			return
		}
	}

	if len(toRemove) > 0 {
		body := map[string]interface{}{
			"saasId":  saasID,
			"userIds": toRemove,
		}
		_, err := r.client.Post(ctx, "/iam/v1/service/user/revoke", body)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("앱 사용자 권한 회수 실패", err.Error())
			}
			return
		}
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userIDs := extractStringSet(ctx, state.UserIDs)

	body := map[string]interface{}{
		"saasId":  state.ID.ValueString(),
		"userIds": userIDs,
	}

	_, err := r.client.Post(ctx, "/iam/v1/service/user/revoke", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("앱 사용자 권한 회수 실패", err.Error())
		}
		return
	}
}

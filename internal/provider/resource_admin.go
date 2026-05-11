package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &AdminResource{}
)

type AdminResource struct {
	client *AlphaKeyClient
}

type AdminResourceModel struct {
	ID           types.String `tfsdk:"id"`
	UserIDs      types.Set    `tfsdk:"user_ids"`
	ValidateFrom types.String `tfsdk:"validate_from"`
	ValidateTo   types.String `tfsdk:"validate_to"`
}

func NewAdminResource() resource.Resource {
	return &AdminResource{}
}

func (r *AdminResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin"
}

func (r *AdminResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 일반 관리자 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "리소스 ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_ids": schema.SetAttribute{
				Description: "관리자로 지정할 사용자 ID 목록",
				Required:    true,
				ElementType: types.StringType,
			},
			"validate_from": schema.StringAttribute{
				Description: "관리자 권한 시작일 (yyyy.MM.dd)",
				Required:    true,
				Validators: []validator.String{
					ValidateDateFormat(),
				},
			},
			"validate_to": schema.StringAttribute{
				Description: "관리자 권한 종료일 (yyyy.MM.dd)",
				Required:    true,
				Validators: []validator.String{
					ValidateDateFormat(),
				},
			},
		},
	}
}

func (r *AdminResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AdminResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AdminResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userIDs := extractStringSet(ctx, plan.UserIDs)

	body := map[string]interface{}{
		"userIds":      userIDs,
		"validateFrom": plan.ValidateFrom.ValueString(),
		"validateTo":   plan.ValidateTo.ValueString(),
	}

	_, err := r.client.Post(ctx, "/iam/v1/settings/normaladmin/add", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("관리자 추가 실패", err.Error())
		}
		return
	}

	// Use sorted first user_id as resource ID for deterministic ordering
	if len(userIDs) > 0 {
		sorted := make([]string, len(userIDs))
		copy(sorted, userIDs)
		sort.Strings(sorted)
		plan.ID = types.StringValue(sorted[0])
	} else {
		plan.ID = types.StringValue("admin")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AdminResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AdminResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Query admin list to verify managed admins still exist
	userIDs := extractStringSet(ctx, state.UserIDs)
	existingIDs := make([]string, 0, len(userIDs))

	for _, uid := range userIDs {
		body := map[string]interface{}{
			"adminId": uid,
		}
		_, err := r.client.Post(ctx, "/iam/v1/settings/admin/detail", body)
		if err != nil {
			if HandleNotFound(err) {
				// This admin no longer exists, skip it
				continue
			}
			// For other errors, report but don't fail the whole read
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("관리자 조회 실패", err.Error())
			}
			return
		}
		existingIDs = append(existingIDs, uid)
	}

	// If no admins exist anymore, remove the resource from state
	if len(existingIDs) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with only existing admin IDs
	state.UserIDs = buildStringSet(ctx, existingIDs)

	// Update ID based on sorted existing IDs
	sorted := make([]string, len(existingIDs))
	copy(sorted, existingIDs)
	sort.Strings(sorted)
	state.ID = types.StringValue(sorted[0])

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AdminResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AdminResourceModel
	var state AdminResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldIDs := extractStringSet(ctx, state.UserIDs)
	newIDs := extractStringSet(ctx, plan.UserIDs)

	toAdd, toRemove := diffSets(oldIDs, newIDs)

	// Remove only the users that are no longer in the set
	if len(toRemove) > 0 {
		deleteBody := map[string]interface{}{
			"userIds": toRemove,
		}
		_, err := r.client.Post(ctx, "/iam/v1/settings/normaladmin/delete", deleteBody)
		if err != nil {
			// Only ignore "not found" errors (admin already removed)
			if !HandleNotFound(err) {
				if apiErr, ok := err.(*APIError); ok {
					resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
				} else {
					resp.Diagnostics.AddError("관리자 삭제 실패", err.Error())
				}
				return
			}
		}
	}

	// Add only the new users
	if len(toAdd) > 0 {
		addBody := map[string]interface{}{
			"userIds":      toAdd,
			"validateFrom": plan.ValidateFrom.ValueString(),
			"validateTo":   plan.ValidateTo.ValueString(),
		}
		_, err := r.client.Post(ctx, "/iam/v1/settings/normaladmin/add", addBody)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok {
				resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
			} else {
				resp.Diagnostics.AddError("관리자 추가 실패", err.Error())
			}
			return
		}
	}

	// If validate dates changed but no user changes, update all existing users
	if len(toAdd) == 0 && len(toRemove) == 0 {
		if state.ValidateFrom.ValueString() != plan.ValidateFrom.ValueString() ||
			state.ValidateTo.ValueString() != plan.ValidateTo.ValueString() {
			// Re-add all users with new dates (API overwrites dates)
			body := map[string]interface{}{
				"userIds":      newIDs,
				"validateFrom": plan.ValidateFrom.ValueString(),
				"validateTo":   plan.ValidateTo.ValueString(),
			}
			_, err := r.client.Post(ctx, "/iam/v1/settings/normaladmin/add", body)
			if err != nil {
				if apiErr, ok := err.(*APIError); ok {
					resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
				} else {
					resp.Diagnostics.AddError("관리자 업데이트 실패", err.Error())
				}
				return
			}
		}
	}

	// Update ID based on sorted new IDs
	sorted := make([]string, len(newIDs))
	copy(sorted, newIDs)
	sort.Strings(sorted)
	if len(sorted) > 0 {
		plan.ID = types.StringValue(sorted[0])
	} else {
		plan.ID = state.ID
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AdminResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AdminResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userIDs := extractStringSet(ctx, state.UserIDs)

	body := map[string]interface{}{
		"userIds": userIDs,
	}

	_, err := r.client.Post(ctx, "/iam/v1/settings/normaladmin/delete", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("관리자 삭제 실패", err.Error())
		}
		return
	}
}

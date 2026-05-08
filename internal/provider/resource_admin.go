package provider

import (
	"context"

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

	// Use first user_id as resource ID
	if len(userIDs) > 0 {
		plan.ID = types.StringValue(userIDs[0])
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

	// Admin resource is write-only; preserve state as-is
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

	// Remove old admins
	oldIDs := extractStringSet(ctx, state.UserIDs)
	if len(oldIDs) > 0 {
		deleteBody := map[string]interface{}{
			"userIds": oldIDs,
		}
		_, _ = r.client.Post(ctx, "/iam/v1/settings/normaladmin/delete", deleteBody)
	}

	// Add new admins
	newIDs := extractStringSet(ctx, plan.UserIDs)
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
			resp.Diagnostics.AddError("관리자 추가 실패", err.Error())
		}
		return
	}

	plan.ID = state.ID
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

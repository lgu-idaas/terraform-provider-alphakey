package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &PasswordExpiryResource{}
)

type PasswordExpiryResource struct {
	client *AlphaKeyClient
}

type PasswordExpiryResourceModel struct {
	ID               types.String `tfsdk:"id"`
	PwdResetTime     types.Int64  `tfsdk:"pwd_reset_time"`
	PwdResetAlarmTime types.Int64  `tfsdk:"pwd_reset_alarm_time"`
}

func NewPasswordExpiryResource() resource.Resource {
	return &PasswordExpiryResource{}
}

func (r *PasswordExpiryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_expiry"
}

func (r *PasswordExpiryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 비밀번호 유효기간 설정 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "리소스 ID (singleton: password_expiry)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pwd_reset_time": schema.Int64Attribute{
				Description: "비밀번호 유효기간(일)",
				Required:    true,
				Validators: []validator.Int64{
					ValidatePositiveInt(),
				},
			},
			"pwd_reset_alarm_time": schema.Int64Attribute{
				Description: "비밀번호 만료 알림 기간(일)",
				Required:    true,
				Validators: []validator.Int64{
					ValidatePositiveInt(),
				},
			},
		},
	}
}

func (r *PasswordExpiryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PasswordExpiryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PasswordExpiryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"pwdResetTime":      plan.PwdResetTime.ValueInt64(),
		"pwdResetAlarmTime": plan.PwdResetAlarmTime.ValueInt64(),
	}

	_, err := r.client.Post(ctx, "/iam/v1/settings/policy/login/pwd/reset/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("비밀번호 유효기간 설정 실패", err.Error())
		}
		return
	}

	plan.ID = types.StringValue("password_expiry")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PasswordExpiryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PasswordExpiryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/settings/policy/login/pwd/reset/detail", nil)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("비밀번호 유효기간 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		PwdResetTime      int64 `json:"pwdResetTime"`
		PwdResetAlarmTime int64 `json:"pwdResetAlarmTime"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "비밀번호 유효기간 조회 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	state.ID = types.StringValue("password_expiry")
	state.PwdResetTime = types.Int64Value(respData.PwdResetTime)
	state.PwdResetAlarmTime = types.Int64Value(respData.PwdResetAlarmTime)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PasswordExpiryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PasswordExpiryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"pwdResetTime":      plan.PwdResetTime.ValueInt64(),
		"pwdResetAlarmTime": plan.PwdResetAlarmTime.ValueInt64(),
	}

	_, err := r.client.Post(ctx, "/iam/v1/settings/policy/login/pwd/reset/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("비밀번호 유효기간 수정 실패", err.Error())
		}
		return
	}

	plan.ID = types.StringValue("password_expiry")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PasswordExpiryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Singleton resource - cannot be deleted, just removed from state
}

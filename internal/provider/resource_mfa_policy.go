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
	_ resource.Resource = &MFAPolicyResource{}
)

type MFAPolicyResource struct {
	client *AlphaKeyClient
}

type MFAPolicyResourceModel struct {
	ID        types.String `tfsdk:"id"`
	AuthType  types.String `tfsdk:"auth_type"`
	UseYn     types.String `tfsdk:"use_yn"`
	FailCnt   types.Int64  `tfsdk:"fail_cnt"`
	TryCnt    types.Int64  `tfsdk:"try_cnt"`
	LimitTime types.Int64  `tfsdk:"limit_time"`
	RegMaxCnt types.Int64  `tfsdk:"reg_max_cnt"`
}

func NewMFAPolicyResource() resource.Resource {
	return &MFAPolicyResource{}
}

func (r *MFAPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mfa_policy"
}

func (r *MFAPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 MFA 인증수단 정책 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "리소스 ID (auth_type과 동일)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auth_type": schema.StringAttribute{
				Description: "인증수단 타입 (sms, email, otp, push, fido, passkey, easyauth)",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					ValidateAuthType(),
				},
			},
			"use_yn": schema.StringAttribute{
				Description: "사용 여부 (Y/N)",
				Required:    true,
			},
			"fail_cnt": schema.Int64Attribute{
				Description: "실패 허용 횟수",
				Required:    true,
			},
			"try_cnt": schema.Int64Attribute{
				Description: "시도 허용 횟수",
				Required:    true,
			},
			"limit_time": schema.Int64Attribute{
				Description: "제한 시간(초)",
				Required:    true,
			},
			"reg_max_cnt": schema.Int64Attribute{
				Description: "최대 등록 수",
				Required:    true,
			},
		},
	}
}

func (r *MFAPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MFAPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MFAPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"authType":  plan.AuthType.ValueString(),
		"useYn":     plan.UseYn.ValueString(),
		"failCnt":   plan.FailCnt.ValueInt64(),
		"tryCnt":    plan.TryCnt.ValueInt64(),
		"limitTime": plan.LimitTime.ValueInt64(),
		"regMaxCnt": plan.RegMaxCnt.ValueInt64(),
	}

	_, err := r.client.Post(ctx, "/mfa/v1/settings/policy/authenticators/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("MFA 정책 생성 실패", err.Error())
		}
		return
	}

	plan.ID = plan.AuthType

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MFAPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MFAPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Post(ctx, "/mfa/v1/settings/policy/authenticators/list", nil)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("MFA 정책 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		List []struct {
			AuthType  string `json:"authType"`
			UseYn     string `json:"useYn"`
			FailCnt   int64  `json:"failCnt"`
			TryCnt    int64  `json:"tryCnt"`
			LimitTime int64  `json:"limitTime"`
			RegMaxCnt int64  `json:"regMaxCnt"`
		} `json:"list"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "MFA 정책 목록 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	authType := state.AuthType.ValueString()
	found := false
	for _, p := range respData.List {
		if p.AuthType == authType {
			state.UseYn = types.StringValue(p.UseYn)
			state.FailCnt = types.Int64Value(p.FailCnt)
			state.TryCnt = types.Int64Value(p.TryCnt)
			state.LimitTime = types.Int64Value(p.LimitTime)
			state.RegMaxCnt = types.Int64Value(p.RegMaxCnt)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MFAPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MFAPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"authType":  plan.AuthType.ValueString(),
		"useYn":     plan.UseYn.ValueString(),
		"failCnt":   plan.FailCnt.ValueInt64(),
		"tryCnt":    plan.TryCnt.ValueInt64(),
		"limitTime": plan.LimitTime.ValueInt64(),
		"regMaxCnt": plan.RegMaxCnt.ValueInt64(),
	}

	_, err := r.client.Post(ctx, "/mfa/v1/settings/policy/authenticators/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("MFA 정책 수정 실패", err.Error())
		}
		return
	}

	plan.ID = plan.AuthType

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MFAPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// MFA policy cannot be deleted, only disabled
	// On destroy, we set use_yn to "N"
	var state MFAPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"authType":  state.AuthType.ValueString(),
		"useYn":     "N",
		"failCnt":   state.FailCnt.ValueInt64(),
		"tryCnt":    state.TryCnt.ValueInt64(),
		"limitTime": state.LimitTime.ValueInt64(),
		"regMaxCnt": state.RegMaxCnt.ValueInt64(),
	}

	_, _ = r.client.Post(ctx, "/mfa/v1/settings/policy/authenticators/update", body)
}

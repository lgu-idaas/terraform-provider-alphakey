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
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

type UserResource struct {
	client *AlphaKeyClient
}

type UserResourceModel struct {
	ID        types.String `tfsdk:"id"`
	DeptName  types.String `tfsdk:"dept_name"`
	DeptID    types.String `tfsdk:"dept_id"`
	LastName  types.String `tfsdk:"last_name"`
	FirstName types.String `tfsdk:"first_name"`
	Email     types.String `tfsdk:"email"`
	Mobile    types.String `tfsdk:"mobile"`
	IDStateYn types.String `tfsdk:"id_state_yn"`
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 사용자 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "사용자 고유 ID (이메일 기반, API에서 자동 생성)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dept_name": schema.StringAttribute{
				Description: "부서명 (최대 15자)",
				Required:    true,
			},
			"dept_id": schema.StringAttribute{
				Description: "부서 아이디 (선택)",
				Optional:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "성 (최대 10자, 공백/특수문자 불가)",
				Required:    true,
			},
			"first_name": schema.StringAttribute{
				Description: "이름 (최대 10자, 공백/특수문자 불가)",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "이메일 (최대 50자)",
				Required:    true,
			},
			"mobile": schema.StringAttribute{
				Description: "전화번호 (010-1234-5678 형식)",
				Required:    true,
			},
			"id_state_yn": schema.StringAttribute{
				Description: "ID 사용여부 (Y: 활성화 메일 전송, N: 미사용)",
				Required:    true,
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"deptName":  plan.DeptName.ValueString(),
		"lastName":  plan.LastName.ValueString(),
		"firstName": plan.FirstName.ValueString(),
		"email":     plan.Email.ValueString(),
		"mobile":    plan.Mobile.ValueString(),
		"idStateYn": plan.IDStateYn.ValueString(),
	}
	if !plan.DeptID.IsNull() && !plan.DeptID.IsUnknown() {
		body["deptId"] = plan.DeptID.ValueString()
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/user/create", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 생성 실패", err.Error())
		}
		return
	}

	var respData struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	plan.ID = types.StringValue(respData.UserID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"userId": state.ID.ValueString(),
	}
	apiResp, err := r.client.Post(ctx, "/iam/v1/user/info/basic/detail", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 조회 실패", err.Error())
		}
		return
	}

	// API returns: userId, userName, deptName, idStateYn (no lastName/firstName/email/mobile)
	// We only update fields that the API returns; keep plan values for the rest.
	var respData struct {
		UserID    string `json:"userId"`
		DeptName  string `json:"deptName"`
		IDStateYn string `json:"idStateYn"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", err.Error())
		return
	}

	state.ID = types.StringValue(respData.UserID)
	if respData.DeptName != "" {
		state.DeptName = types.StringValue(respData.DeptName)
	}
	if respData.IDStateYn != "" {
		state.IDStateYn = types.StringValue(respData.IDStateYn)
	}
	// lastName, firstName, email, mobile are NOT returned by the detail API
	// Keep existing state values to prevent unnecessary updates

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	var state UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"userId":    state.ID.ValueString(),
		"deptName":  plan.DeptName.ValueString(),
		"lastName":  plan.LastName.ValueString(),
		"firstName": plan.FirstName.ValueString(),
		"mobile":    plan.Mobile.ValueString(),
	}
	if !plan.DeptID.IsNull() && !plan.DeptID.IsUnknown() {
		body["deptId"] = plan.DeptID.ValueString()
	}

	_, err := r.client.Post(ctx, "/iam/v1/user/info/update", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 수정 실패", err.Error())
		}
		return
	}

	// Handle id_state_yn changes
	oldIDState := state.IDStateYn.ValueString()
	newIDState := plan.IDStateYn.ValueString()
	if oldIDState != newIDState {
		idBody := map[string]interface{}{"userIds": []string{state.ID.ValueString()}}
		if oldIDState == "N" && newIDState == "Y" {
			_, err := r.client.Post(ctx, "/iam/v1/user/id/grant", idBody)
			if err != nil {
				if apiErr, ok := err.(*APIError); ok {
					resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
				} else {
					resp.Diagnostics.AddError("사용자 ID 생성 실패", err.Error())
				}
				return
			}
		} else if oldIDState == "Y" && newIDState == "N" {
			_, err := r.client.Post(ctx, "/iam/v1/user/id/revoke", idBody)
			if err != nil {
				if apiErr, ok := err.(*APIError); ok {
					resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
				} else {
					resp.Diagnostics.AddError("사용자 ID 회수 실패", err.Error())
				}
				return
			}
		}
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	revokeBody := map[string]interface{}{"userIds": []string{state.ID.ValueString()}}
	deleteBody := map[string]interface{}{"userId": state.ID.ValueString()}

	// Revoke ID first (ignore "already revoked" errors)
	_, revokeErr := r.client.Post(ctx, "/iam/v1/user/id/revoke", revokeBody)
	if revokeErr != nil {
		if apiErr, ok := revokeErr.(*APIError); ok {
			if apiErr.Code != "E1040606" && apiErr.Code != "E1040112" {
				resp.Diagnostics.AddWarning("사용자 ID 회수 중 오류", apiErr.Error())
			}
		}
	}

	// Delete user
	_, err := r.client.Post(ctx, "/iam/v1/user/delete", deleteBody)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 삭제 실패", err.Error())
		}
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

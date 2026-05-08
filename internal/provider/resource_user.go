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

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

// UserResource defines the resource implementation.
type UserResource struct {
	client *AlphaKeyClient
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	ID        types.String `tfsdk:"id"`
	DeptName  types.String `tfsdk:"dept_name"`
	LastName  types.String `tfsdk:"last_name"`
	FirstName types.String `tfsdk:"first_name"`
	Email     types.String `tfsdk:"email"`
	Mobile    types.String `tfsdk:"mobile"`
	IDStateYn types.String `tfsdk:"id_state_yn"`
	DeptID    types.String `tfsdk:"dept_id"`
}

// NewUserResource returns a new resource.Resource for the user resource.
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
				Description: "사용자 고유 ID (API에서 자동 생성)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dept_name": schema.StringAttribute{
				Description: "부서명",
				Required:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "성",
				Required:    true,
			},
			"first_name": schema.StringAttribute{
				Description: "이름",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "이메일",
				Required:    true,
			},
			"mobile": schema.StringAttribute{
				Description: "전화번호",
				Required:    true,
			},
			"id_state_yn": schema.StringAttribute{
				Description: "ID 사용여부 (Y/N)",
				Required:    true,
			},
			"dept_id": schema.StringAttribute{
				Description: "부서 아이디",
				Optional:    true,
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
		resp.Diagnostics.AddError(
			"잘못된 Provider 데이터",
			"Provider에서 전달된 데이터가 *AlphaKeyClient 타입이 아닙니다.",
		)
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

	// Build request body
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

	// Call API
	apiResp, err := r.client.Post(ctx, "/iam/v1/user/create", body)
	if err != nil {
		apiErr, ok := err.(*APIError)
		if ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 생성 실패", err.Error())
		}
		return
	}

	// Parse response to get userId
	var respData struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "사용자 생성 응답에서 userId를 파싱할 수 없습니다: "+err.Error())
		return
	}

	// Set state
	plan.ID = types.StringValue(respData.UserID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	body := map[string]interface{}{
		"userId": state.ID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/user/info/basic/detail", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		apiErr, ok := err.(*APIError)
		if ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 조회 실패", err.Error())
		}
		return
	}

	// Parse response
	var respData struct {
		UserID    string `json:"userId"`
		DeptName  string `json:"deptName"`
		LastName  string `json:"lastName"`
		FirstName string `json:"firstName"`
		Email     string `json:"email"`
		Mobile    string `json:"mobile"`
		IDStateYn string `json:"idStateYn"`
		DeptID    string `json:"deptId"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "사용자 조회 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	// Map response to state
	state.ID = types.StringValue(respData.UserID)
	state.DeptName = types.StringValue(respData.DeptName)
	state.LastName = types.StringValue(respData.LastName)
	state.FirstName = types.StringValue(respData.FirstName)
	state.Email = types.StringValue(respData.Email)
	state.Mobile = types.StringValue(respData.Mobile)
	state.IDStateYn = types.StringValue(respData.IDStateYn)

	if respData.DeptID != "" {
		state.DeptID = types.StringValue(respData.DeptID)
	} else {
		state.DeptID = types.StringNull()
	}

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

	// Build update request body
	body := map[string]interface{}{
		"userId":    state.ID.ValueString(),
		"deptName":  plan.DeptName.ValueString(),
		"lastName":  plan.LastName.ValueString(),
		"firstName": plan.FirstName.ValueString(),
		"email":     plan.Email.ValueString(),
		"mobile":    plan.Mobile.ValueString(),
	}

	if !plan.DeptID.IsNull() && !plan.DeptID.IsUnknown() {
		body["deptId"] = plan.DeptID.ValueString()
	}

	// Call update API
	_, err := r.client.Post(ctx, "/iam/v1/user/info/update", body)
	if err != nil {
		apiErr, ok := err.(*APIError)
		if ok {
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
		idBody := map[string]interface{}{
			"userId": state.ID.ValueString(),
		}

		if oldIDState == "N" && newIDState == "Y" {
			// Grant ID
			_, err := r.client.Post(ctx, "/iam/v1/user/id/grant", idBody)
			if err != nil {
				apiErr, ok := err.(*APIError)
				if ok {
					resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
				} else {
					resp.Diagnostics.AddError("사용자 ID 부여 실패", err.Error())
				}
				return
			}
		} else if oldIDState == "Y" && newIDState == "N" {
			// Revoke ID
			_, err := r.client.Post(ctx, "/iam/v1/user/id/revoke", idBody)
			if err != nil {
				apiErr, ok := err.(*APIError)
				if ok {
					resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
				} else {
					resp.Diagnostics.AddError("사용자 ID 회수 실패", err.Error())
				}
				return
			}
		}
	}

	// Set state from plan
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"userId": state.ID.ValueString(),
	}

	// First, revoke ID (ignore error if already revoked)
	_, _ = r.client.Post(ctx, "/iam/v1/user/id/revoke", body)

	// Then, delete user
	_, err := r.client.Post(ctx, "/iam/v1/user/delete", body)
	if err != nil {
		apiErr, ok := err.(*APIError)
		if ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("사용자 삭제 실패", err.Error())
		}
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

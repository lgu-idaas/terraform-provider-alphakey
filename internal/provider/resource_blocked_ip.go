package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &BlockedIPResource{}
	_ resource.ResourceWithImportState = &BlockedIPResource{}
)

type BlockedIPResource struct {
	client *AlphaKeyClient
}

type BlockedIPResourceModel struct {
	ID         types.String `tfsdk:"id"`
	IPAddress  types.String `tfsdk:"ip_address"`
	SubnetMask types.String `tfsdk:"subnet_mask"`
	Reason     types.String `tfsdk:"reason"`
}

func NewBlockedIPResource() resource.Resource {
	return &BlockedIPResource{}
}

func (r *BlockedIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blocked_ip"
}

func (r *BlockedIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "알파키 접속 차단 IP 리소스를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "차단 정책 고유 ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_address": schema.StringAttribute{
				Description: "차단할 IPv4 주소",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					ValidateIPv4(),
				},
			},
			"subnet_mask": schema.StringAttribute{
				Description: "서브넷 마스크",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					ValidateSubnetMask(),
				},
			},
			"reason": schema.StringAttribute{
				Description: "차단 사유",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *BlockedIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BlockedIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlockedIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"ipAddress":  plan.IPAddress.ValueString(),
		"subnetMask": plan.SubnetMask.ValueString(),
		"reason":     plan.Reason.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/security/policy/network/ip/block/create", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("차단 IP 생성 실패", err.Error())
		}
		return
	}

	var respData struct {
		NetworkPolicyID string `json:"networkPolicyId"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "차단 IP 생성 응답에서 networkPolicyId를 파싱할 수 없습니다: "+err.Error())
		return
	}

	plan.ID = types.StringValue(respData.NetworkPolicyID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BlockedIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlockedIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"networkPolicyId": state.ID.ValueString(),
	}

	apiResp, err := r.client.Post(ctx, "/iam/v1/security/policy/network/ip/block/detail", body)
	if err != nil {
		if HandleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("차단 IP 조회 실패", err.Error())
		}
		return
	}

	var respData struct {
		NetworkPolicyID string `json:"networkPolicyId"`
		IPAddress       string `json:"ipAddress"`
		SubnetMask      string `json:"subnetMask"`
		Reason          string `json:"reason"`
	}
	if err := json.Unmarshal(apiResp.Data, &respData); err != nil {
		resp.Diagnostics.AddError("응답 파싱 실패", "차단 IP 조회 응답을 파싱할 수 없습니다: "+err.Error())
		return
	}

	state.ID = types.StringValue(respData.NetworkPolicyID)
	state.IPAddress = types.StringValue(respData.IPAddress)
	state.SubnetMask = types.StringValue(respData.SubnetMask)
	state.Reason = types.StringValue(respData.Reason)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BlockedIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *BlockedIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlockedIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"networkPolicyId": state.ID.ValueString(),
	}

	_, err := r.client.Post(ctx, "/iam/v1/security/policy/network/ip/block/delete", body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			resp.Diagnostics.Append(MapAPIErrorToDiagnostics(apiErr)...)
		} else {
			resp.Diagnostics.AddError("차단 IP 삭제 실패", err.Error())
		}
		return
	}
}

func (r *BlockedIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

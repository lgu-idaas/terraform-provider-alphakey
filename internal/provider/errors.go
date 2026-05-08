package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ErrorMessages maps AlphaKey error codes to Korean diagnostic messages.
var ErrorMessages = map[string]string{
	"E1040601": "이메일 주소가 중복되었습니다",
	"E1040608": "사용 가능한 계정이 없습니다",
	"E1040304": "인사동기화로 추가한 사용자는 수정할 수 없습니다",
	"E1040301": "인사동기화로 추가한 사용자는 삭제할 수 없습니다",
	"E1040606": "사용 중인 앱 권한을 먼저 회수해야 합니다",
	"E1030605": "사용자와 서비스 그룹에 등록되어 있어 삭제할 수 없습니다",
	"E1040605": "그룹에 사용자가 등록되어 있어 삭제할 수 없습니다",
	"E1040603": "해당 그룹명은 이미 사용 중입니다",
	"E1030607": "동일한 이름의 서비스 그룹이 이미 존재합니다",
	"E1030603": "구매한 아이디 수를 초과하여 사용자를 추가할 수 없습니다",
	"E1030604": "이미 앱에 할당된 사용자가 포함되어 있습니다",
	"E1050601": "이미 등록된 IP 주소입니다",
	"E1060103": "유효하지 않은 사용자가 있습니다",
	"E1060104": "유효기간 시작일은 오늘 날짜 이후로 지정할 수 없습니다",
	"E3060901": "인증 정책 정보 업데이트 시 오류가 발생하였습니다",
}

// MapAPIErrorToDiagnostics converts an APIError into Terraform diag.Diagnostics.
// It checks if the error code exists in the ErrorMessages map and uses the mapped
// Korean message. Otherwise, it falls back to the error's own Message field.
func MapAPIErrorToDiagnostics(err *APIError) diag.Diagnostics {
	var diags diag.Diagnostics

	if err == nil {
		return diags
	}

	// Determine the detail message
	var detail string
	if mappedMsg, ok := ErrorMessages[err.Code]; ok {
		detail = fmt.Sprintf("[%s] %s", err.Code, mappedMsg)
	} else if err.Code != "" {
		detail = fmt.Sprintf("[%s] %s", err.Code, err.Message)
	} else {
		detail = err.Message
	}

	diags.AddError("알파키 API 오류", detail)

	return diags
}

// HandleNotFound checks if the given error is an *APIError with HTTPStatus == 404.
// Returns true if the error represents a "not found" response, indicating the resource
// should be removed from Terraform State.
func HandleNotFound(err error) bool {
	if err == nil {
		return false
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}

	return apiErr.HTTPStatus == 404
}

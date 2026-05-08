package provider

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// =============================================================================
// IPv4Validator - validates ip_address attribute
// =============================================================================

// ipv4Validator validates that a string is a valid IPv4 address.
type ipv4Validator struct{}

// ValidateIPv4 returns a validator that checks for valid IPv4 format.
func ValidateIPv4() validator.String {
	return &ipv4Validator{}
}

func (v *ipv4Validator) Description(_ context.Context) string {
	return "유효한 IPv4 주소 형식이어야 합니다 (예: 192.168.1.1)"
}

func (v *ipv4Validator) MarkdownDescription(_ context.Context) string {
	return "유효한 IPv4 주소 형식이어야 합니다 (예: `192.168.1.1`)"
}

func (v *ipv4Validator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation for null or unknown values
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Parse IPv4: must be exactly 4 octets, dot-separated, each 0-255
	parts := strings.Split(value, ".")
	if len(parts) != 4 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 IPv4 주소",
			fmt.Sprintf("IPv4 주소는 점(.)으로 구분된 4개의 옥텟이어야 합니다. 입력값: %q", value),
		)
		return
	}

	for i, part := range parts {
		if part == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"유효하지 않은 IPv4 주소",
				fmt.Sprintf("IPv4 주소의 %d번째 옥텟이 비어있습니다. 입력값: %q", i+1, value),
			)
			return
		}

		// Check for leading zeros (e.g., "01", "001")
		if len(part) > 1 && part[0] == '0' {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"유효하지 않은 IPv4 주소",
				fmt.Sprintf("IPv4 주소의 옥텟에 선행 0이 포함되어 있습니다. 입력값: %q", value),
			)
			return
		}

		num, err := strconv.Atoi(part)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"유효하지 않은 IPv4 주소",
				fmt.Sprintf("IPv4 주소의 옥텟은 숫자여야 합니다. 입력값: %q", value),
			)
			return
		}

		if num < 0 || num > 255 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"유효하지 않은 IPv4 주소",
				fmt.Sprintf("IPv4 주소의 각 옥텟은 0~255 범위여야 합니다. 입력값: %q", value),
			)
			return
		}
	}
}

// =============================================================================
// SubnetMaskValidator - validates subnet_mask attribute
// =============================================================================

// subnetMaskValidator validates that a string is a valid subnet mask.
type subnetMaskValidator struct{}

// ValidateSubnetMask returns a validator that checks for valid subnet mask format.
func ValidateSubnetMask() validator.String {
	return &subnetMaskValidator{}
}

func (v *subnetMaskValidator) Description(_ context.Context) string {
	return "유효한 서브넷 마스크 형식이어야 합니다 (예: 255.255.255.0)"
}

func (v *subnetMaskValidator) MarkdownDescription(_ context.Context) string {
	return "유효한 서브넷 마스크 형식이어야 합니다 (예: `255.255.255.0`)"
}

func (v *subnetMaskValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation for null or unknown values
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Parse as IPv4 first
	ip := net.ParseIP(value)
	if ip == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 서브넷 마스크",
			fmt.Sprintf("서브넷 마스크는 유효한 IPv4 형식이어야 합니다. 입력값: %q", value),
		)
		return
	}

	ip = ip.To4()
	if ip == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 서브넷 마스크",
			fmt.Sprintf("서브넷 마스크는 IPv4 형식이어야 합니다. 입력값: %q", value),
		)
		return
	}

	// Convert to 32-bit integer
	mask := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])

	// A valid subnet mask must be contiguous 1-bits followed by contiguous 0-bits
	// This means: if we invert the mask and add 1, the result must be a power of 2
	// Or equivalently: mask must not be 0, and (^mask + 1) & ^mask == 0
	if mask == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 서브넷 마스크",
			fmt.Sprintf("서브넷 마스크는 0.0.0.0이 될 수 없습니다. 입력값: %q", value),
		)
		return
	}

	// Check contiguous bits: invert and check if it's (2^n - 1) form
	inverted := ^mask
	if inverted != 0 && (inverted&(inverted+1)) != 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 서브넷 마스크",
			fmt.Sprintf("서브넷 마스크는 연속된 1비트 뒤에 연속된 0비트로 구성되어야 합니다. 입력값: %q", value),
		)
		return
	}
}

// =============================================================================
// DateFormatValidator - validates yyyy.MM.dd format
// =============================================================================

// dateFormatValidator validates that a string matches yyyy.MM.dd format.
type dateFormatValidator struct{}

// ValidateDateFormat returns a validator that checks for yyyy.MM.dd date format.
func ValidateDateFormat() validator.String {
	return &dateFormatValidator{}
}

func (v *dateFormatValidator) Description(_ context.Context) string {
	return "날짜는 yyyy.MM.dd 형식이어야 합니다 (예: 2024.01.15)"
}

func (v *dateFormatValidator) MarkdownDescription(_ context.Context) string {
	return "날짜는 `yyyy.MM.dd` 형식이어야 합니다 (예: `2024.01.15`)"
}

func (v *dateFormatValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation for null or unknown values
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Use time.Parse with Go reference layout "2006.01.02" for yyyy.MM.dd
	_, err := time.Parse("2006.01.02", value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 날짜 형식",
			fmt.Sprintf("날짜는 yyyy.MM.dd 형식이어야 합니다 (예: 2024.01.15). 입력값: %q", value),
		)
		return
	}
}

// =============================================================================
// AuthTypeValidator - validates auth_type enum
// =============================================================================

// authTypeValidator validates that a string is one of the allowed auth types.
type authTypeValidator struct{}

// ValidateAuthType returns a validator that checks for valid auth_type values.
func ValidateAuthType() validator.String {
	return &authTypeValidator{}
}

// validAuthTypes contains the allowed auth_type values.
var validAuthTypes = map[string]bool{
	"sms":      true,
	"email":    true,
	"otp":      true,
	"push":     true,
	"fido":     true,
	"passkey":  true,
	"easyauth": true,
}

func (v *authTypeValidator) Description(_ context.Context) string {
	return "인증수단 타입은 sms, email, otp, push, fido, passkey, easyauth 중 하나여야 합니다"
}

func (v *authTypeValidator) MarkdownDescription(_ context.Context) string {
	return "인증수단 타입은 `sms`, `email`, `otp`, `push`, `fido`, `passkey`, `easyauth` 중 하나여야 합니다"
}

func (v *authTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation for null or unknown values
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	if !validAuthTypes[value] {
		allowed := []string{"sms", "email", "otp", "push", "fido", "passkey", "easyauth"}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 인증수단 타입",
			fmt.Sprintf(
				"auth_type은 다음 값 중 하나여야 합니다: %s. 입력값: %q",
				strings.Join(allowed, ", "),
				value,
			),
		)
		return
	}
}

// =============================================================================
// PositiveIntValidator - validates positive integers
// =============================================================================

// positiveIntValidator validates that an int64 value is greater than 0.
type positiveIntValidator struct{}

// ValidatePositiveInt returns a validator that checks for positive integer values.
func ValidatePositiveInt() validator.Int64 {
	return &positiveIntValidator{}
}

func (v *positiveIntValidator) Description(_ context.Context) string {
	return "값은 0보다 큰 양의 정수여야 합니다"
}

func (v *positiveIntValidator) MarkdownDescription(_ context.Context) string {
	return "값은 0보다 큰 양의 정수여야 합니다"
}

func (v *positiveIntValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	// Skip validation for null or unknown values
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueInt64()

	if value <= 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"유효하지 않은 값",
			fmt.Sprintf("값은 0보다 큰 양의 정수여야 합니다. 입력값: %d", value),
		)
		return
	}
}

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// =============================================================================
// IPv4Validator Tests
// =============================================================================

func TestIPv4Validator_Valid(t *testing.T) {
	validIPs := []string{
		"0.0.0.0",
		"192.168.1.1",
		"10.0.0.1",
		"255.255.255.255",
		"172.16.0.1",
		"1.2.3.4",
	}

	for _, ip := range validIPs {
		t.Run(ip, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("ip_address"),
				ConfigValue: types.StringValue(ip),
			}
			resp := &validator.StringResponse{}

			ValidateIPv4().ValidateString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", ip, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestIPv4Validator_Invalid(t *testing.T) {
	invalidIPs := []string{
		"",
		"256.1.1.1",
		"1.2.3",
		"1.2.3.4.5",
		"abc.def.ghi.jkl",
		"192.168.1",
		"192.168.1.1.1",
		"-1.0.0.0",
		"1.2.3.256",
		"01.02.03.04",
		"1..2.3",
	}

	for _, ip := range invalidIPs {
		t.Run(ip, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("ip_address"),
				ConfigValue: types.StringValue(ip),
			}
			resp := &validator.StringResponse{}

			ValidateIPv4().ValidateString(context.Background(), req, resp)

			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", ip)
			}
		})
	}
}

func TestIPv4Validator_NullAndUnknown(t *testing.T) {
	// Null value should skip validation
	req := validator.StringRequest{
		Path:        path.Root("ip_address"),
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}
	ValidateIPv4().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	// Unknown value should skip validation
	req = validator.StringRequest{
		Path:        path.Root("ip_address"),
		ConfigValue: types.StringUnknown(),
	}
	resp = &validator.StringResponse{}
	ValidateIPv4().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

// =============================================================================
// SubnetMaskValidator Tests
// =============================================================================

func TestSubnetMaskValidator_Valid(t *testing.T) {
	validMasks := []string{
		"255.255.255.0",
		"255.255.0.0",
		"255.0.0.0",
		"255.255.255.128",
		"255.255.255.192",
		"255.255.255.224",
		"255.255.255.240",
		"255.255.255.248",
		"255.255.255.252",
		"255.255.255.254",
		"255.255.255.255",
		"128.0.0.0",
		"192.0.0.0",
		"224.0.0.0",
		"240.0.0.0",
		"248.0.0.0",
		"252.0.0.0",
		"254.0.0.0",
	}

	for _, mask := range validMasks {
		t.Run(mask, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("subnet_mask"),
				ConfigValue: types.StringValue(mask),
			}
			resp := &validator.StringResponse{}

			ValidateSubnetMask().ValidateString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", mask, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestSubnetMaskValidator_Invalid(t *testing.T) {
	invalidMasks := []string{
		"0.0.0.0",
		"255.0.255.0",
		"255.255.0.255",
		"192.168.1.1",
		"0.255.255.255",
		"abc",
		"",
		"255.255.255.1",
		"1.0.0.0",
	}

	for _, mask := range invalidMasks {
		t.Run(mask, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("subnet_mask"),
				ConfigValue: types.StringValue(mask),
			}
			resp := &validator.StringResponse{}

			ValidateSubnetMask().ValidateString(context.Background(), req, resp)

			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", mask)
			}
		})
	}
}

func TestSubnetMaskValidator_NullAndUnknown(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("subnet_mask"),
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}
	ValidateSubnetMask().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	req = validator.StringRequest{
		Path:        path.Root("subnet_mask"),
		ConfigValue: types.StringUnknown(),
	}
	resp = &validator.StringResponse{}
	ValidateSubnetMask().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

// =============================================================================
// DateFormatValidator Tests
// =============================================================================

func TestDateFormatValidator_Valid(t *testing.T) {
	validDates := []string{
		"2024.01.15",
		"2023.12.31",
		"2000.01.01",
		"2025.06.30",
		"1999.02.28",
	}

	for _, date := range validDates {
		t.Run(date, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("validate_from"),
				ConfigValue: types.StringValue(date),
			}
			resp := &validator.StringResponse{}

			ValidateDateFormat().ValidateString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", date, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestDateFormatValidator_Invalid(t *testing.T) {
	invalidDates := []string{
		"",
		"2024-01-15",
		"2024/01/15",
		"01.15.2024",
		"2024.13.01",
		"2024.00.01",
		"2024.01.32",
		"2024.01.00",
		"24.01.15",
		"abcd.ef.gh",
		"2024.1.1",
		"2023.02.29",
	}

	for _, date := range invalidDates {
		t.Run(date, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("validate_from"),
				ConfigValue: types.StringValue(date),
			}
			resp := &validator.StringResponse{}

			ValidateDateFormat().ValidateString(context.Background(), req, resp)

			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", date)
			}
		})
	}
}

func TestDateFormatValidator_NullAndUnknown(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("validate_from"),
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}
	ValidateDateFormat().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	req = validator.StringRequest{
		Path:        path.Root("validate_from"),
		ConfigValue: types.StringUnknown(),
	}
	resp = &validator.StringResponse{}
	ValidateDateFormat().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

// =============================================================================
// AuthTypeValidator Tests
// =============================================================================

func TestAuthTypeValidator_Valid(t *testing.T) {
	validTypes := []string{
		"sms",
		"email",
		"otp",
		"push",
		"fido",
		"passkey",
		"easyauth",
	}

	for _, authType := range validTypes {
		t.Run(authType, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("auth_type"),
				ConfigValue: types.StringValue(authType),
			}
			resp := &validator.StringResponse{}

			ValidateAuthType().ValidateString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", authType, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestAuthTypeValidator_Invalid(t *testing.T) {
	invalidTypes := []string{
		"",
		"SMS",
		"EMAIL",
		"totp",
		"biometric",
		"password",
		"unknown",
		"sms ",
		" email",
	}

	for _, authType := range invalidTypes {
		t.Run(authType, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("auth_type"),
				ConfigValue: types.StringValue(authType),
			}
			resp := &validator.StringResponse{}

			ValidateAuthType().ValidateString(context.Background(), req, resp)

			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", authType)
			}
		})
	}
}

func TestAuthTypeValidator_NullAndUnknown(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("auth_type"),
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}
	ValidateAuthType().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	req = validator.StringRequest{
		Path:        path.Root("auth_type"),
		ConfigValue: types.StringUnknown(),
	}
	resp = &validator.StringResponse{}
	ValidateAuthType().ValidateString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

// =============================================================================
// PositiveIntValidator Tests
// =============================================================================

func TestPositiveIntValidator_Valid(t *testing.T) {
	validValues := []int64{1, 2, 10, 100, 365, 9999}

	for _, val := range validValues {
		t.Run("value_positive", func(t *testing.T) {
			req := validator.Int64Request{
				Path:        path.Root("pwd_reset_time"),
				ConfigValue: types.Int64Value(val),
			}
			resp := &validator.Int64Response{}

			ValidatePositiveInt().ValidateInt64(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %d, got: %s", val, resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

func TestPositiveIntValidator_Invalid(t *testing.T) {
	invalidValues := []int64{0, -1, -100}

	for _, val := range invalidValues {
		t.Run("value_non_positive", func(t *testing.T) {
			req := validator.Int64Request{
				Path:        path.Root("pwd_reset_time"),
				ConfigValue: types.Int64Value(val),
			}
			resp := &validator.Int64Response{}

			ValidatePositiveInt().ValidateInt64(context.Background(), req, resp)

			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %d, got none", val)
			}
		})
	}
}

func TestPositiveIntValidator_NullAndUnknown(t *testing.T) {
	req := validator.Int64Request{
		Path:        path.Root("pwd_reset_time"),
		ConfigValue: types.Int64Null(),
	}
	resp := &validator.Int64Response{}
	ValidatePositiveInt().ValidateInt64(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	req = validator.Int64Request{
		Path:        path.Root("pwd_reset_time"),
		ConfigValue: types.Int64Unknown(),
	}
	resp = &validator.Int64Response{}
	ValidatePositiveInt().ValidateInt64(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

// =============================================================================
// Description/MarkdownDescription Tests
// =============================================================================

func TestValidatorDescriptions(t *testing.T) {
	ctx := context.Background()

	// IPv4
	ipv4 := ValidateIPv4()
	if ipv4.Description(ctx) == "" {
		t.Error("IPv4 validator Description should not be empty")
	}
	if ipv4.MarkdownDescription(ctx) == "" {
		t.Error("IPv4 validator MarkdownDescription should not be empty")
	}

	// SubnetMask
	subnet := ValidateSubnetMask()
	if subnet.Description(ctx) == "" {
		t.Error("SubnetMask validator Description should not be empty")
	}
	if subnet.MarkdownDescription(ctx) == "" {
		t.Error("SubnetMask validator MarkdownDescription should not be empty")
	}

	// DateFormat
	date := ValidateDateFormat()
	if date.Description(ctx) == "" {
		t.Error("DateFormat validator Description should not be empty")
	}
	if date.MarkdownDescription(ctx) == "" {
		t.Error("DateFormat validator MarkdownDescription should not be empty")
	}

	// AuthType
	auth := ValidateAuthType()
	if auth.Description(ctx) == "" {
		t.Error("AuthType validator Description should not be empty")
	}
	if auth.MarkdownDescription(ctx) == "" {
		t.Error("AuthType validator MarkdownDescription should not be empty")
	}

	// PositiveInt
	posInt := ValidatePositiveInt()
	if posInt.Description(ctx) == "" {
		t.Error("PositiveInt validator Description should not be empty")
	}
	if posInt.MarkdownDescription(ctx) == "" {
		t.Error("PositiveInt validator MarkdownDescription should not be empty")
	}
}

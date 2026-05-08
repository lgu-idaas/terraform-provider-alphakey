package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AlphaKeyClient is the HTTP client for communicating with the AlphaKey API.
type AlphaKeyClient struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
}

// APIResponse represents the common response structure from the AlphaKey API.
type APIResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// APIError represents a structured error from the AlphaKey API.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
	Retryable  bool
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("AlphaKey API error (HTTP %d, code=%s): %s", e.HTTPStatus, e.Code, e.Message)
	}
	return fmt.Sprintf("AlphaKey API error (HTTP %d): %s", e.HTTPStatus, e.Message)
}

// buildURL constructs the full API URL from the base URL and path.
// If the path does not start with /openapi, it prepends /openapi.
// Example: BaseURL="https://company.alphakey.kr", path="/iam/v1/user/list"
// Result: "https://company.alphakey.kr/openapi/iam/v1/user/list"
func (c *AlphaKeyClient) buildURL(path string) string {
	baseURL := strings.TrimRight(c.BaseURL, "/")

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if strings.HasPrefix(path, "/openapi") {
		return baseURL + path
	}

	return baseURL + "/openapi" + path
}

// maskToken masks the API token for safe logging.
// Shows first 4 and last 4 characters, masks the rest.
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// DoRequest performs a common HTTP request to the AlphaKey API.
// It handles JSON serialization, header injection, logging, and response parsing.
func (c *AlphaKeyClient) DoRequest(ctx context.Context, method, path string, body interface{}) (*APIResponse, error) {
	fullURL := c.buildURL(path)

	var reqBody io.Reader
	var bodyBytes []byte

	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("요청 본문 JSON 직렬화 실패: %w", err)
		}
		reqBody = bytes.NewReader(bodyBytes)
	}

	// Log the request (mask token)
	tflog.Debug(ctx, "AlphaKey API 요청", map[string]interface{}{
		"method": method,
		"url":    fullURL,
		"body":   string(bodyBytes),
		"token":  maskToken(c.APIToken),
	})

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 생성 실패: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIToken)

	// Execute the request
	startTime := time.Now()
	resp, err := c.HTTPClient.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		// Check for timeout errors
		if ctx.Err() != nil || isTimeoutError(err) {
			tflog.Error(ctx, "AlphaKey API 타임아웃", map[string]interface{}{
				"method":  method,
				"url":     fullURL,
				"elapsed": elapsed.String(),
				"error":   err.Error(),
			})
			return nil, &APIError{
				HTTPStatus: 0,
				Code:       "TIMEOUT",
				Message:    fmt.Sprintf("네트워크 연결 시간이 초과되었습니다 (%s): %s", elapsed.Round(time.Millisecond), err.Error()),
				Retryable:  true,
			}
		}
		tflog.Error(ctx, "AlphaKey API 네트워크 오류", map[string]interface{}{
			"method": method,
			"url":    fullURL,
			"error":  err.Error(),
		})
		return nil, &APIError{
			HTTPStatus: 0,
			Code:       "NETWORK_ERROR",
			Message:    fmt.Sprintf("네트워크 연결 실패: %s", err.Error()),
			Retryable:  true,
		}
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 본문 읽기 실패: %w", err)
	}

	// Log the response
	tflog.Debug(ctx, "AlphaKey API 응답", map[string]interface{}{
		"method":      method,
		"url":         fullURL,
		"status_code": resp.StatusCode,
		"elapsed":     elapsed.String(),
		"body":        string(respBody),
	})

	// Handle HTTP-level errors
	if resp.StatusCode != http.StatusOK {
		return nil, handleHTTPError(resp.StatusCode, respBody)
	}

	// Parse the JSON response
	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("응답 JSON 파싱 실패: %w (body: %s)", err, string(respBody))
	}

	// Check API-level error (code != "200")
	if apiResp.Code != "200" {
		tflog.Error(ctx, "AlphaKey API 비즈니스 오류", map[string]interface{}{
			"method": method,
			"url":    fullURL,
			"code":   apiResp.Code,
			"msg":    apiResp.Msg,
		})
		return nil, &APIError{
			HTTPStatus: resp.StatusCode,
			Code:       apiResp.Code,
			Message:    apiResp.Msg,
			Retryable:  false,
		}
	}

	return &apiResp, nil
}

// Post performs a POST request to the AlphaKey API.
// Most AlphaKey API endpoints use POST method.
func (c *AlphaKeyClient) Post(ctx context.Context, path string, body interface{}) (*APIResponse, error) {
	return c.DoRequest(ctx, http.MethodPost, path, body)
}

// Get performs a GET request to the AlphaKey API.
func (c *AlphaKeyClient) Get(ctx context.Context, path string) (*APIResponse, error) {
	return c.DoRequest(ctx, http.MethodGet, path, nil)
}

// Put performs a PUT request to the AlphaKey API.
func (c *AlphaKeyClient) Put(ctx context.Context, path string, body interface{}) (*APIResponse, error) {
	return c.DoRequest(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request to the AlphaKey API.
func (c *AlphaKeyClient) Delete(ctx context.Context, path string, body interface{}) (*APIResponse, error) {
	return c.DoRequest(ctx, http.MethodDelete, path, body)
}

// handleHTTPError creates an APIError based on the HTTP status code and response body.
func handleHTTPError(statusCode int, body []byte) *APIError {
	// Try to parse the response body for additional error info
	var apiResp APIResponse
	_ = json.Unmarshal(body, &apiResp)

	switch statusCode {
	case http.StatusUnauthorized: // 401
		return &APIError{
			HTTPStatus: statusCode,
			Code:       apiResp.Code,
			Message:    "인증 정보가 유효하지 않습니다. api_token을 확인해주세요.",
			Retryable:  false,
		}
	case http.StatusForbidden: // 403
		return &APIError{
			HTTPStatus: statusCode,
			Code:       apiResp.Code,
			Message:    "이 리소스에 접근할 권한이 없습니다.",
			Retryable:  false,
		}
	case http.StatusNotFound: // 404
		return &APIError{
			HTTPStatus: statusCode,
			Code:       apiResp.Code,
			Message:    "요청한 리소스를 찾을 수 없습니다.",
			Retryable:  false,
		}
	case http.StatusBadRequest: // 400
		msg := "클라이언트 요청 오류가 발생했습니다."
		if apiResp.Msg != "" {
			msg = apiResp.Msg
		}
		code := apiResp.Code
		// Try to extract errorCode from data field
		if apiResp.Data != nil {
			var errorData struct {
				ErrorCode    string `json:"errorCode"`
				ErrorMessage string `json:"errorMessage"`
			}
			if json.Unmarshal(apiResp.Data, &errorData) == nil && errorData.ErrorCode != "" {
				code = errorData.ErrorCode
				if errorData.ErrorMessage != "" {
					msg = errorData.ErrorMessage
				}
			}
		}
		return &APIError{
			HTTPStatus: statusCode,
			Code:       code,
			Message:    msg,
			Retryable:  false,
		}
	case http.StatusInternalServerError: // 500
		return &APIError{
			HTTPStatus: statusCode,
			Code:       apiResp.Code,
			Message:    "알파키 서버 내부 오류가 발생했습니다. 잠시 후 다시 시도해주세요.",
			Retryable:  true,
		}
	default:
		msg := fmt.Sprintf("예상하지 못한 HTTP 오류가 발생했습니다 (상태 코드: %d)", statusCode)
		if apiResp.Msg != "" {
			msg = apiResp.Msg
		}
		return &APIError{
			HTTPStatus: statusCode,
			Code:       apiResp.Code,
			Message:    msg,
			Retryable:  statusCode >= 500,
		}
	}
}

// isTimeoutError checks if an error is a timeout error.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	// Check for net.Error interface with Timeout() method
	type timeoutErr interface {
		Timeout() bool
	}
	if te, ok := err.(timeoutErr); ok {
		return te.Timeout()
	}
	// Check error message for common timeout indicators
	errMsg := err.Error()
	return strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "context deadline exceeded")
}

package wrapresp

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	v1 "github.com/go-kratos/gateway/api/gateway/middleware/wrapresp/v1"
	"github.com/go-kratos/gateway/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	middleware.Register("wrapresp", Middleware)
}

// Middleware is a response wrapper middleware that wraps response body into {code, msg, data} format.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.WrapResp{}
	if c != nil && c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	// Set defaults
	successCode := int32(0)
	successMsg := "success"
	wrapError := true
	wrapSuccess := true

	if options.SuccessCode != nil {
		successCode = *options.SuccessCode
	}
	if options.SuccessMsg != nil {
		successMsg = *options.SuccessMsg
	}
	if options.WrapError != nil {
		wrapError = *options.WrapError
	}
	if options.WrapSuccess != nil {
		wrapSuccess = *options.WrapSuccess
	}

	// Pre-build success response prefix and suffix for optimization
	// {"code":0,"msg":"success","data":
	successPrefix := `{"code":` + strconv.Itoa(int(successCode)) + `,"msg":"` + successMsg + `","data":`
	successSuffix := `}`

	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			// Check content type - only wrap JSON responses
			contentType := resp.Header.Get("Content-Type")
			if contentType != "" && !isJSONContentType(contentType) {
				return resp, nil
			}

			// Read response body
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body.Close()

			// Determine if this is a success or error response
			isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 400

			// Check if we should wrap this response
			if isSuccess && !wrapSuccess {
				resp.Body = io.NopCloser(bytes.NewReader(data))
				return resp, nil
			}
			if !isSuccess && !wrapError {
				resp.Body = io.NopCloser(bytes.NewReader(data))
				return resp, nil
			}

			var wrappedData []byte

			if isSuccess {
				// Optimized: directly build JSON string without json.Marshal
				// For success: {"code":0,"msg":"success","data":<original_data>}
				wrappedData = buildSuccessResponse(successPrefix, data, successSuffix)
			} else {
				// Error response
				code := int32(resp.StatusCode)
				if options.ErrorCode != nil {
					code = *options.ErrorCode
				}

				msg := http.StatusText(resp.StatusCode)
				// Try to extract error message from response body if configured
				if options.ErrorMsgField != nil && *options.ErrorMsgField != "" {
					msg = extractErrorMsg(data, *options.ErrorMsgField, msg)
				}

				// For error: {"code":500,"msg":"Internal Server Error","data":null}
				wrappedData = buildErrorResponse(int(code), msg)
			}

			// Create new response with wrapped body
			resp.Body = io.NopCloser(bytes.NewReader(wrappedData))
			resp.ContentLength = int64(len(wrappedData))
			resp.Header.Set("Content-Type", "application/json")
			resp.Header.Del("Content-Length")

			return resp, nil
		})
	}, nil
}

// buildSuccessResponse builds a success response JSON directly
func buildSuccessResponse(prefix string, data []byte, suffix string) []byte {
	// Estimate capacity to reduce allocations
	estimated := len(prefix) + len(data) + len(suffix)
	result := make([]byte, 0, estimated)
	result = append(result, prefix...)
	result = append(result, data...)
	result = append(result, suffix...)
	return result
}

// buildErrorResponse builds an error response JSON directly
func buildErrorResponse(code int, msg string) []byte {
	// {"code":500,"msg":"Internal Server Error","data":null}
	return []byte(`{"code":` + strconv.Itoa(code) + `,"msg":"` + escapeJSONString(msg) + `","data":null}`)
}

// extractErrorMsg extracts error message from response body
func extractErrorMsg(data []byte, field string, defaultMsg string) string {
	// Simple field extraction without full JSON parsing
	// Look for "field":"value" pattern
	searchKey := `"` + field + `":"`
	startIdx := bytes.Index(data, []byte(searchKey))
	if startIdx == -1 {
		return defaultMsg
	}

	// Move to the start of value
	valueStart := startIdx + len(searchKey)
	if valueStart >= len(data) {
		return defaultMsg
	}

	// Find the end quote
	endIdx := bytes.Index(data[valueStart:], []byte(`"`))
	if endIdx == -1 {
		return defaultMsg
	}

	return string(data[valueStart : valueStart+endIdx])
}

// escapeJSONString escapes special characters in JSON strings
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// isJSONContentType checks if the content type is JSON (optimized with strings.Contains)
func isJSONContentType(contentType string) bool {
	return strings.Contains(contentType, "application/json")
}
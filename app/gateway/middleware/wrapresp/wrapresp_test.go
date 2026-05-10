package wrapresp

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/go-kratos/gateway/middleware"
)

func TestMiddleware_SuccessResponse(t *testing.T) {
	m, err := Middleware(nil)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(`{"name":"test"}`))),
		}, nil
	})

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	wrapped := m(mockNext)
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		t.Fatalf("failed to execute middleware: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	expected := `{"code":0,"msg":"success","data":{"name":"test"}}`
	if string(body) != expected {
		t.Errorf("expected body '%s', got '%s'", expected, string(body))
	}
}

func TestMiddleware_ErrorResponse(t *testing.T) {
	m, err := Middleware(nil)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(`{"error":"internal error"}`))),
		}, nil
	})

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	wrapped := m(mockNext)
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		t.Fatalf("failed to execute middleware: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	expected := `{"code":500,"msg":"Internal Server Error","data":null}`
	if string(body) != expected {
		t.Errorf("expected body '%s', got '%s'", expected, string(body))
	}
}

func TestMiddleware_NonJSONResponse(t *testing.T) {
	m, err := Middleware(nil)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/html"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(`<html><body>test</body></html>`))),
		}, nil
	})

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	wrapped := m(mockNext)
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		t.Fatalf("failed to execute middleware: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	expected := `<html><body>test</body></html>`
	if string(body) != expected {
		t.Errorf("expected body '%s', got '%s'", expected, string(body))
	}
}

func TestMiddleware_EmptyContentType(t *testing.T) {
	m, err := Middleware(nil)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// When content type is empty, should still wrap
	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"name":"test"}`))),
		}, nil
	})

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	wrapped := m(mockNext)
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		t.Fatalf("failed to execute middleware: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	expected := `{"code":0,"msg":"success","data":{"name":"test"}}`
	if string(body) != expected {
		t.Errorf("expected body '%s', got '%s'", expected, string(body))
	}
}

func TestBuildSuccessResponse(t *testing.T) {
	result := buildSuccessResponse(`{"code":0,"msg":"success","data":`, []byte(`{"name":"test"}`), `}`)
	expected := `{"code":0,"msg":"success","data":{"name":"test"}}`
	if string(result) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(result))
	}
}

func TestBuildErrorResponse(t *testing.T) {
	result := buildErrorResponse(500, "Internal Server Error")
	expected := `{"code":500,"msg":"Internal Server Error","data":null}`
	if string(result) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(result))
	}
}

func TestExtractErrorMsg(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		field       string
		defaultMsg  string
		want        string
	}{
		{
			name:       "field exists",
			data:       `{"error":"custom error message"}`,
			field:      "error",
			defaultMsg: "default",
			want:       "custom error message",
		},
		{
			name:       "field not found",
			data:       `{"message":"some msg"}`,
			field:      "error",
			defaultMsg: "default",
			want:       "default",
		},
		{
			name:       "empty data",
			data:       ``,
			field:      "error",
			defaultMsg: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractErrorMsg([]byte(tt.data), tt.field, tt.defaultMsg)
			if got != tt.want {
				t.Errorf("extractErrorMsg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeJSONString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{`hello "world"`, `hello \"world\"`},
		{`hello\world`, `hello\\world`},
		{"hello\nworld", `hello\nworld`},
		{"hello\tworld", `hello\tworld`},
	}

	for _, tt := range tests {
		got := escapeJSONString(tt.input)
		if got != tt.want {
			t.Errorf("escapeJSONString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"text/html", false},
		{"text/plain", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			if got := isJSONContentType(tt.contentType); got != tt.want {
				t.Errorf("isJSONContentType(%s) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}
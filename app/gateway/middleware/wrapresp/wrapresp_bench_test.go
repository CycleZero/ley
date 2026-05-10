package wrapresp

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/go-kratos/gateway/middleware"
)

// BenchmarkMiddleware benchmarks the middleware performance
func BenchmarkMiddleware(b *testing.B) {
	// Create middleware with default options
	m, _ := Middleware(nil)

	// Sample response body
	responseBody := `{"id":12345,"name":"test user","email":"test@example.com","created_at":"2024-01-01T00:00:00Z"}`

	// Create mock roundtripper
	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(responseBody))),
		}, nil
	})

	wrapped := m(mockNext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "http://example.com/test", nil)
		resp, _ := wrapped.RoundTrip(req)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkMiddlewareNoWrap benchmarks without wrapping (passthrough)
func BenchmarkMiddlewareNoWrap(b *testing.B) {
	// Direct response without middleware
	responseBody := `{"id":12345,"name":"test user","email":"test@example.com","created_at":"2024-01-01T00:00:00Z"}`

	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(responseBody))),
		}, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "http://example.com/test", nil)
		resp, _ := mockNext.RoundTrip(req)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkMiddlewareLargeResponse benchmarks with large response body
func BenchmarkMiddlewareLargeResponse(b *testing.B) {
	m, _ := Middleware(nil)

	// Large response body (simulate ~10KB)
	var buf bytes.Buffer
	buf.WriteString(`{"data":[`)
	for i := 0; i < 100; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(`{"id":` + string(rune('0'+i%10)) + `,"name":"item` + string(rune('0'+i%10)) + `"}`)
	}
	buf.WriteString(`]}`)
	responseBody := buf.String()

	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(responseBody))),
		}, nil
	})

	wrapped := m(mockNext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "http://example.com/test", nil)
		resp, _ := wrapped.RoundTrip(req)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkMiddlewareSmallResponse benchmarks with small response body
func BenchmarkMiddlewareSmallResponse(b *testing.B) {
	m, _ := Middleware(nil)

	responseBody := `{"ok":true}`

	mockNext := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader([]byte(responseBody))),
		}, nil
	})

	wrapped := m(mockNext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "http://example.com/test", nil)
		resp, _ := wrapped.RoundTrip(req)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
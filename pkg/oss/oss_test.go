package oss

import (
	"errors"
	"testing"
)

// TestErrorMapping 验证各实现错误码到统一哨兵错误的映射
func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		target  error
		matched bool
	}{
		// 阿里云 OSS 错误码映射
		{"aliyun NoSuchKey -> ErrObjectNotFound", &aliyunError{code: "NoSuchKey", err: errors.New("x")}, ErrObjectNotFound, true},
		{"aliyun NoSuchBucket -> ErrBucketNotFound", &aliyunError{code: "NoSuchBucket", err: errors.New("x")}, ErrBucketNotFound, true},
		{"aliyun AccessDenied -> ErrAccessDenied", &aliyunError{code: "AccessDenied", err: errors.New("x")}, ErrAccessDenied, true},
		{"aliyun 未知码不匹配", &aliyunError{code: "InternalError", err: errors.New("x")}, ErrObjectNotFound, false},
		// MinIO / S3 错误码映射
		{"minio NoSuchKey -> ErrObjectNotFound", &minioError{code: "NoSuchKey", err: errors.New("x")}, ErrObjectNotFound, true},
		{"minio NoSuchBucket -> ErrBucketNotFound", &minioError{code: "NoSuchBucket", err: errors.New("x")}, ErrBucketNotFound, true},
		{"minio SignatureDoesNotMatch -> ErrAccessDenied", &minioError{code: "SignatureDoesNotMatch", err: errors.New("x")}, ErrAccessDenied, true},
		{"minio 未知码不匹配", &minioError{code: "SlowDown", err: errors.New("x")}, ErrObjectNotFound, false},
	}
	for _, c := range cases {
		if got := errors.Is(c.err, c.target); got != c.matched {
			t.Errorf("%s: errors.Is = %v, want %v", c.name, got, c.matched)
		}
	}
}

// TestValidateExpiry 验证预签名过期时间边界（1s ~ 7 天）
func TestValidateExpiry(t *testing.T) {
	cases := []struct {
		seconds int64
		valid   bool
	}{
		{0, false},
		{-1, false},
		{1, true},
		{3600, true},
		{7 * 24 * 3600, true},
		{7*24*3600 + 1, false},
	}
	for _, c := range cases {
		err := validateExpiry(c.seconds)
		if c.valid && err != nil {
			t.Errorf("validateExpiry(%d) 不应报错: %v", c.seconds, err)
		}
		if !c.valid && !errors.Is(err, ErrInvalidParam) {
			t.Errorf("validateExpiry(%d) 应返回 ErrInvalidParam, got %v", c.seconds, err)
		}
	}
}

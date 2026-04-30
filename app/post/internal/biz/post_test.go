package biz

import (
	"errors"
	"strings"
	"testing"

	"ley/app/post/internal/model"
)

// =============================================================================
// GenerateSlug Tests — 标题 → URL Slug
// =============================================================================

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{name: "纯英文小写", title: "hello world", want: "hello-world"},
		{name: "英文含大写", title: "Hello World", want: "hello-world"},
		{name: "英文含标点", title: "Hello, World!", want: "hello-world"},
		{name: "连续空格变连字符", title: "a   b", want: "a-b"},
		{name: "首尾连字符去除", title: "-hello-world-", want: "hello-world"},
		{name: "含数字", title: "Post 42 Tips", want: "post-42-tips"},
		{name: "中文标题（未收录的字回退）", title: "你好世界", want: "post"}, // "你"不在预设字表中，全特殊字符回退
		{name: "中英混合（仅英文保留）", title: "Kratos微服务框架", want: "kratos"}, // "微服务框架"不在字表中，仅保留 kratos
		{name: "全特殊字符", title: "!@#$%", want: "post"},
		{name: "空标题", title: "", want: "post"},
		{name: "仅连字符", title: "---", want: "post"},
		{name: "超长截断", title: strings.Repeat("a", 250), want: strings.Repeat("a", 200)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.title)
			if got != tt.want {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

// =============================================================================
// GenerateExcerpt Tests — Markdown → 纯文本摘要
// =============================================================================

func TestGenerateExcerpt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		maxLen  int
		// wantPrefix: 因为开头的空格/换行去除可能变化，只验证前缀
		wantPrefix string
		wantLen    int // 预期 rune 长度（不含 "..."）
	}{
		{
			name:       "短于 maxLen 全量返回",
			content:    "Hello world",
			maxLen:     100,
			wantPrefix: "Hello world",
			wantLen:    11,
		},
		{
			name:       "中文明文截断",
			content:    "你好世界你好世界你好世界",
			maxLen:     5,
			wantPrefix: "你好世界你", // runes[0:5] = 你好世界你
			wantLen:    5,
		},
		{
			name:       "超长英文截断",
			content:    strings.Repeat("abc", 50), // 150 chars
			maxLen:     10,
			wantPrefix: strings.Repeat("abc", 4)[:10], // "abcabcabca" = 10 runes
			wantLen:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateExcerpt(tt.content, tt.maxLen)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("generateExcerpt(..., %d) prefix = %q, want prefix %q",
					tt.maxLen, truncateForDisplay(got, 50), tt.wantPrefix)
			}
		})
	}
}

// =============================================================================
// Tag / Slug Helpers
// =============================================================================

func TestTagNameToSlug(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Go Language", want: "go-language"},
		{name: "  Kubernetes  ", want: "kubernetes"},
		{name: "AI/ML", want: "ai/ml"},
		{name: "全中文标签", want: "全中文标签"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagNameToSlug(tt.name)
			if got != tt.want {
				t.Errorf("tagNameToSlug(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestTagIDs(t *testing.T) {
	tags := []*model.Tag{
		{Name: "Go"},
		{Name: "Rust"},
	}
	ids := tagIDs(tags)
	if len(ids) != 2 {
		t.Fatalf("tagIDs count = %d, want 2", len(ids))
	}
}

// =============================================================================
// validatePostInput Tests
// =============================================================================

func TestValidatePostInput(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		wantErr error
	}{
		{name: "合法输入", title: "Hello World", content: "Some content here", wantErr: nil},
		{name: "标题过短 1 字符", title: "A", content: "Some content", wantErr: ErrPostTitleEmpty},
		{name: "标题为空", title: "", content: "Content", wantErr: ErrPostTitleEmpty},
		{name: "标题超长 201", title: strings.Repeat("A", 201), content: "X", wantErr: ErrPostTitleEmpty},
		{name: "内容为空", title: "Title", content: "", wantErr: ErrPostContentEmpty},
		{name: "内容超长 100001", title: "Title", content: strings.Repeat("X", 100001), wantErr: ErrPostContentTooBig},
		{name: "边界值 200 字符标题", title: strings.Repeat("A", 200), content: "X", wantErr: nil},
		{name: "边界值 100000 字符内容", title: "OK", content: strings.Repeat("X", 100000), wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePostInput(tt.title, tt.content)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validatePostInput() = %v, want nil", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("validatePostInput() = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

// =============================================================================
// Error Sentinel Tests
// =============================================================================

func TestPostErrorSentinels(t *testing.T) {
	tests := []struct {
		name        string
		a           error
		b           error
		shouldMatch bool
	}{
		{name: "ErrPostNotFound 自反", a: ErrPostNotFound, b: ErrPostNotFound, shouldMatch: true},
		{name: "ErrNotPostOwner ≠ ErrPostNotFound", a: ErrNotPostOwner, b: ErrPostNotFound, shouldMatch: false},
		{name: "ErrPostAlreadyPublished ≠ ErrPostContentEmpty", a: ErrPostAlreadyPublished, b: ErrPostContentEmpty, shouldMatch: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errors.Is(tt.a, tt.b) != tt.shouldMatch {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.a, tt.b, !tt.shouldMatch, tt.shouldMatch)
			}
		})
	}
}

// =============================================================================
// Helpers
// =============================================================================

func truncateForDisplay(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

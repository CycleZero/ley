package biz

import (
	"strings"
	"testing"

	"ley/app/comment/internal/model"
)

// =============================================================================
// BuildCommentTree Tests
// =============================================================================

func TestBuildCommentTree(t *testing.T) {
	t.Run("空列表返回 nil", func(t *testing.T) {
		tree := BuildCommentTree(nil)
		if tree != nil {
			t.Errorf("BuildCommentTree(nil) = %v, want nil", tree)
		}
		tree = BuildCommentTree([]*model.Comment{})
		if tree != nil {
			t.Errorf("BuildCommentTree([]) = %v, want nil", tree)
		}
	})

	t.Run("全部顶级评论无嵌套", func(t *testing.T) {
		comments := []*model.Comment{
			{Content: "Comment 1"},
			{Content: "Comment 2"},
			{Content: "Comment 3"},
		}
		for i := range comments {
			comments[i].ID = uint(i + 1)
		}

		tree := BuildCommentTree(comments)
		if len(tree) != 3 {
			t.Fatalf("root count = %d, want 3", len(tree))
		}
		for i, node := range tree {
			if node.Comment.Content != comments[i].Content {
				t.Errorf("node[%d].Comment.Content = %q, want %q",
					i, node.Comment.Content, comments[i].Content)
			}
			if len(node.Children) != 0 {
				t.Errorf("node[%d] has %d children, want 0", i, len(node.Children))
			}
		}
	})

	t.Run("两级嵌套评论", func(t *testing.T) {
		pID := uint(1)
		comments := []*model.Comment{
			{Content: "Parent"},                // ID=1
			{Content: "Reply 1", ParentID: &pID}, // ID=2
			{Content: "Reply 2", ParentID: &pID}, // ID=3
		}
		for i := range comments {
			comments[i].ID = uint(i + 1)
		}

		tree := BuildCommentTree(comments)
		if len(tree) != 1 {
			t.Fatalf("root count = %d, want 1", len(tree))
		}
		parent := tree[0]
		if parent.Comment.Content != "Parent" {
			t.Errorf("root content = %q, want %q", parent.Comment.Content, "Parent")
		}
		if len(parent.Children) != 2 {
			t.Fatalf("children count = %d, want 2", len(parent.Children))
		}
		if parent.Children[0].Comment.Content != "Reply 1" {
			t.Errorf("child[0] = %q, want %q", parent.Children[0].Comment.Content, "Reply 1")
		}
	})

	t.Run("三级深层嵌套", func(t *testing.T) {
		p1 := uint(1)
		p2 := uint(2)
		comments := []*model.Comment{
			{Content: "L1"},                        // ID=1
			{Content: "L2", ParentID: &p1},           // ID=2
			{Content: "L3", ParentID: &p2},           // ID=3
		}
		for i := range comments {
			comments[i].ID = uint(i + 1)
		}

		tree := BuildCommentTree(comments)
		if len(tree) != 1 {
			t.Fatalf("root count = %d, want 1", len(tree))
		}
		mid := tree[0].Children
		if len(mid) != 1 {
			t.Fatalf("mid count = %d, want 1", len(mid))
		}
		deep := mid[0].Children
		if len(deep) != 1 {
			t.Fatalf("deep count = %d, want 1", len(deep))
		}
		if deep[0].Comment.Content != "L3" {
			t.Errorf("deep content = %q, want %q", deep[0].Comment.Content, "L3")
		}
	})

	t.Run("孤儿评论（父节点不存在）提升为顶级", func(t *testing.T) {
		orphanPID := uint(999) // 不存在的父节点
		comments := []*model.Comment{
			{Content: "Orphan", ParentID: &orphanPID}, // ID=1
		}
		comments[0].ID = 1

		tree := BuildCommentTree(comments)
		if len(tree) != 1 {
			t.Fatalf("orphan should become root: got %d, want 1", len(tree))
		}
		if tree[0].Comment.Content != "Orphan" {
			t.Errorf("orphan root content = %q, want %q", tree[0].Comment.Content, "Orphan")
		}
	})

	t.Run("乱序评论仍能正确挂载", func(t *testing.T) {
		// 子评论在前、父评论在后
		pID := uint(2)
		comments := []*model.Comment{
			{Content: "Child", ParentID: &pID},  // ID=1, parent is ID=2
			{Content: "Parent"},                 // ID=2
		}
		comments[0].ID = 1
		comments[1].ID = 2

		tree := BuildCommentTree(comments)
		if len(tree) != 1 {
			t.Fatalf("root count = %d, want 1", len(tree))
		}
		if len(tree[0].Children) != 1 {
			t.Fatalf("children count = %d, want 1", len(tree[0].Children))
		}
	})

	t.Run("ParentID=0 视为顶级评论", func(t *testing.T) {
		zeroID := uint(0)
		comments := []*model.Comment{
			{Content: "Top", ParentID: &zeroID},
		}
		comments[0].ID = 1

		tree := BuildCommentTree(comments)
		if len(tree) != 1 {
			t.Fatalf("ParentID=0 should be root: got %d, want 1", len(tree))
		}
	})
}

// =============================================================================
// validateCommentContent Tests
// =============================================================================

func TestValidateCommentContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{name: "正常评论", content: "Great post!", wantErr: nil},
		{name: "边界最小长度", content: "A", wantErr: nil},
		{name: "空白字符转为空", content: "   ", wantErr: ErrCommentTooShort},
		{name: "空评论", content: "", wantErr: ErrCommentTooShort},
		{name: "边界最大长度", content: strings.Repeat("好", 2000), wantErr: nil},
		{name: "超长评论 2001", content: strings.Repeat("A", 2001), wantErr: ErrCommentTooLong},
		{name: "中文字符计数", content: strings.Repeat("你好", 1001), wantErr: ErrCommentTooLong}, // 2002 runes > 2000
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommentContent(tt.content)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validateCommentContent(%q) = %v, want nil", tt.content, err)
				}
			} else {
				if err == nil {
					t.Errorf("validateCommentContent(%q) = nil, want %v", tt.content, tt.wantErr)
				}
			}
		})
	}
}

// =============================================================================
// Comment Error Sentinel Isolation
// =============================================================================

func TestCommentErrorSentinels(t *testing.T) {
	// Verify each error is distinct via errors.Is
	errPairs := []struct{ a, b error }{
		{ErrCommentNotFound, ErrNotCommentOwner},
		{ErrCommentTooShort, ErrCommentTooLong},
		{ErrMaxDepthExceeded, ErrParentNotFound},
	}
	for _, p := range errPairs {
		if p.a == p.b {
			t.Errorf("errors should be distinct: %v == %v", p.a, p.b)
		}
	}
}

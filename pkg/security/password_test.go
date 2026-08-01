package security

import "testing"

func TestHashPasswordAndVerify(t *testing.T) {
	password := "Password123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if hash == "" {
		t.Fatal("hash 不应为空")
	}
	if hash == password {
		t.Fatal("hash 不应等于明文")
	}

	if !VerifyPassword(password, hash) {
		t.Error("正确密码应验证通过")
	}
}

func TestVerifyPasswordWrongPassword(t *testing.T) {
	hash, err := HashPassword("Password123")
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if VerifyPassword("WrongPass123", hash) {
		t.Error("错误密码不应验证通过")
	}
	if VerifyPassword("", hash) {
		t.Error("空密码不应验证通过")
	}
}

func TestHashIsRandomized(t *testing.T) {
	// bcrypt 每次生成不同的盐：同一密码两次哈希应不同，但都能验证
	password := "Password123"
	h1, _ := HashPassword(password)
	h2, _ := HashPassword(password)
	if h1 == h2 {
		t.Error("两次哈希不应相同（盐随机化）")
	}
	if !VerifyPassword(password, h1) || !VerifyPassword(password, h2) {
		t.Error("两个哈希都应能验证同一密码")
	}
}

func TestVerifyPasswordInvalidHash(t *testing.T) {
	// 非法 hash 不应 panic，应返回 false
	if VerifyPassword("x", "not-a-bcrypt-hash") {
		t.Error("非法 hash 应返回 false")
	}
	if VerifyPassword("x", "") {
		t.Error("空 hash 应返回 false")
	}
}

package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
)

// PoWConfig PoW 配置
type PoWConfig struct {
	Enabled      bool   // 是否启用 PoW
	Mode         string // enforce | log | off
	Difficulty   int    // 难度（前 N 位为 0
	ChallengeTTL int64  // challenge 有效期（秒）
}

// PoWChallenge PoW 组件
type PoW struct {
	config     *PoWConfig
	challenges map[string]*challenge // challenge 存储
	mu         sync.RWMutex
}

type challenge struct {
	ID         string
	Challenge  string
	Difficulty int
	CreatedAt  int64
	Used       bool
}

// NewPoW 创建 PoW 实例
func NewPoW(config *PoWConfig) *PoW {
	p := &PoW{
		config:     config,
		challenges: make(map[string]*challenge),
	}
	// 启动清理协程定期清理过期的 challenge
	go p.cleanupExpiredChallenges()
	return p
}

// ChallengeResponse 获取 challenge 的响应
type ChallengeResponse struct {
	ChallengeID string `json:"challenge_id"`
	Challenge   string `json:"challenge"`
	Difficulty  int    `json:"difficulty"`
}

// GenerateChallenge 生成 challenge
func (p *PoW) GenerateChallenge() *ChallengeResponse {
	challengeID := uuid.NewString()
	challengeStr := uuid.NewString()

	chal := &challenge{
		ID:         challengeID,
		Challenge:  challengeStr,
		Difficulty: p.config.Difficulty,
		CreatedAt:  time.Now().Unix(),
		Used:       false,
	}

	p.mu.Lock()
	p.challenges[challengeID] = chal
	p.mu.Unlock()

	return &ChallengeResponse{
		ChallengeID: challengeID,
		Challenge:   challengeStr,
		Difficulty:  p.config.Difficulty,
	}
}

// VerifyPoW 验证 PoW
func (p *PoW) VerifyPoW(challengeID, nonce string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	chal, exists := p.challenges[challengeID]
	if !exists {
		return errors.New("invalid challenge id")
	}

	if chal.Used {
		return errors.New("challenge already used")
	}

	// 检查是否过期
	now := time.Now().Unix()
	if now-chal.CreatedAt > p.config.ChallengeTTL {
		delete(p.challenges, challengeID)
		return errors.New("challenge expired")
	}

	// 验证 PoW
	if !p.verifyHash(chal.Challenge, nonce, chal.Difficulty) {
		return errors.New("invalid proof of work")
	}

	// 标记为已使用
	chal.Used = true
	return nil
}

// verifyHash 验证 hash
func (p *PoW) verifyHash(challenge, nonce string, difficulty int) bool {
	data := challenge + nonce
	hash := sha256.Sum256([]byte(data))
	hashStr := hex.EncodeToString(hash[:])

	// 检查前 N 位是否为 0
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hashStr, prefix)
}

// cleanupExpiredChallenges 清理过期的 challenge
func (p *PoW) cleanupExpiredChallenges() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		now := time.Now().Unix()
		for id, chal := range p.challenges {
			if now-chal.CreatedAt > p.config.ChallengeTTL || chal.Used {
				delete(p.challenges, id)
			}
		}
		p.mu.Unlock()
	}
}

// Server PoW 中间件
func (p *PoW) Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 如果 PoW 未启用，直接通过
			if !p.config.Enabled || p.config.Mode == "off" {
				return handler(ctx, req)
			}

			if header, ok := transport.FromServerContext(ctx); ok {
				path := header.Operation()

				// 检查是否是豁免路径
				if p.isExemptPath(path) {
					return handler(ctx, req)
				}

				// 只对 /webapi/* 路径进行 PoW 验证
				if !strings.HasPrefix(path, "/webapi/") {
					return handler(ctx, req)
				}

				// 验证 PoW
				challengeID := header.RequestHeader().Get("X-PoW-Challenge")
				nonce := header.RequestHeader().Get("X-PoW-Nonce")

				if challengeID == "" || nonce == "" {
					if p.config.Mode == "enforce" {
						return nil, errors.New("missing pow headers")
					}
					// log 模式，只记录不阻止
				} else {
					if err := p.VerifyPoW(challengeID, nonce); err != nil {
						if p.config.Mode == "enforce" {
							return nil, fmt.Errorf("pow verification failed: %w", err)
						}
						// log 模式，只记录不阻止
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// isExemptPath 检查是否是豁免路径
func (p *PoW) isExemptPath(path string) bool {
	exemptPaths := []string{
		"/webapi/pow/challenge",
		"/webapi/sms/callback",
		"/webapi/wechat-pay/notify",
	}
	for _, exemptPath := range exemptPaths {
		if strings.HasPrefix(path, exemptPath) {
			return true
		}
	}
	return false
}

// CalculateNonce 计算 nonce（用于测试）
func (p *PoW) CalculateNonce(challenge string, difficulty int) string {
	for i := 0; ; i++ {
		nonce := strconv.Itoa(i)
		if p.verifyHash(challenge, nonce, difficulty) {
			return nonce
		}
	}
}

#!/usr/bin/env bash
# Ley 业务配置 etcd 种子脚本
#
# 后端业务配置（DB/Redis/JWT/NATS/MinIO）运行时仅从 etcd 读取
# （路径 ley/configs/{service}/config.yaml，缺失即启动失败——不存在"本地 fallback"）。
# 用法：
#   1. 编辑下方 {auth,blog,entry} 配置中的密码/密钥（生产必改占位值）
#   2. ETCDCTL_API=3 etcdctl --endpoints=127.0.0.1:2379 可用
#   3. bash scripts/seed-etcd.sh
#
# 说明：值与 configs/{auth,blog}.yaml 的业务段保持一致（MySQL 基线）。
set -euo pipefail

ETCD_ENDPOINTS="${ETCD_ENDPOINTS:-127.0.0.1:2379}"

auth_cfg=$(cat <<'YAML'
data:
  database:
    driver: mysql
    host: 127.0.0.1
    port: 3306
    database: ley
    username: root
    password: change-me-in-production
  redis:
    host: 127.0.0.1
    port: 6379
    password: ""
    db: 0
jwt:
  secret: "change-me-in-production-use-256bit-random-key"
  access_ttl: 900s
  refresh_ttl: 604800s
  issuer: "ley-auth"
YAML
)

blog_cfg=$(cat <<'YAML'
data:
  database:
    driver: mysql
    host: 127.0.0.1
    port: 3306
    database: ley
    username: root
    password: change-me-in-production
  redis:
    host: 127.0.0.1
    port: 6379
    password: ""
    db: 1
nats:
  addr: nats://127.0.0.1:4222
  enable_jetstream: true
minio:
  endpoint: 127.0.0.1:9000
  access_key_id: minioadmin
  access_key_secret: minioadmin
  bucket_name: ley-images
YAML
)

# entry 入口服务远程业务配置（结构与 app/entry/internal/conf/remote.go 的
# RemoteConfig 一一对应：yaml 键名不得随意改动）。
# ⚠️ jwt.secret 必须与上方 auth 的 secret 完全一致：entry 只校验不签发
#    access/refresh token，密钥不一致将导致所有已登录请求鉴权失败；
#    redis 也用 db0 与 auth 对齐（黑名单读写同一实例同一库）。
entry_cfg=$(cat <<'YAML'
jwt:
  secret: "change-me-in-production-use-256bit-random-key"
  issuer: "ley-auth"
  access_ttl: 900s
  refresh_ttl: 604800s
redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0
cors:
  allow_origins:
    - "*"
  allow_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  allow_headers:
    - Origin
    - Content-Type
    - Authorization
  max_age: 86400
ratelimit:
  rps: 100
  burst: 200
YAML
)

put() {
  local key="$1" val="$2"
  echo "etcdctl put ${key}"
  ETCDCTL_API=3 etcdctl --endpoints="${ETCD_ENDPOINTS}" put "${key}" "${val}"
}

put "ley/configs/auth/config.yaml" "${auth_cfg}"
put "ley/configs/blog/config.yaml" "${blog_cfg}"
put "ley/configs/entry/config.yaml" "${entry_cfg}"

echo "完成。注意：网关 JWT 密钥若走 etcd 动态源，需另配 ley/configs/gateway/jwt（参考 configs/gateway.yaml 头注）。"

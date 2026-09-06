# Ley 博客平台部署指南

本文档提供 Ley 博客平台（Go/Kratos 后端 + Nuxt 4 前端）的**生产环境部署**详细流程。

## 一、架构概览

```
用户 ──HTTPS──> 服务器
                  │
    ┌─────────────┼─────────────┐
    │             │             │
 Nginx/Caddy   前端(Nuxt)    后端(Gateway)
 (:443)       (:3000)       (:8000)
                              │
                    ┌────────┴────────┐
                   Auth(:9001)      Blog(:9002)
                              │
                    ┌────────┴────────┐
              PostgreSQL    Redis    etcd
              MinIO        NATS    (可选: Jaeger)
```

## 二、服务器环境准备

### 2.1 系统要求

- **OS**: Ubuntu 22.04 LTS / Debian 12 / CentOS 9（推荐 Ubuntu）
- **CPU**: 2 核+
- **内存**: 4GB+
- **磁盘**: 40GB+ SSD
- **带宽**: 5Mbps+

### 2.2 安装 Docker 和 Docker Compose

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装 Docker
sudo apt install -y docker.io docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker $USER

# 验证
docker --version
docker compose version
```

> 注销并重新登录，使 docker 权限生效。

### 2.3 克隆项目

```bash
cd ~
git clone https://github.com/yourname/ley.git
cd ley
```

## 三、基础设施部署

创建 `docker-compose.infra.yml`：

```yaml
services:
  postgres:
    image: postgres:15-alpine
    container_name: ley-postgres
    environment:
      POSTGRES_DB: ley
      POSTGRES_USER: ley
      POSTGRES_PASSWORD: "你的强密码"
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "127.0.0.1:5432:5432"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 512M

  redis:
    image: redis:7-alpine
    container_name: ley-redis
    command: redis-server --appendonly yes --requirepass "你的强密码"
    volumes:
      - ./data/redis:/data
    ports:
      - "127.0.0.1:6379:6379"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 256M

  etcd:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: ley-etcd
    environment:
      ETCD_NAME: ley-etcd
      ETCD_DATA_DIR: /etcd-data
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_ADVERTISE_CLIENT_URLS: http://127.0.0.1:2379
      ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
      ETCD_INITIAL_ADVERTISE_PEER_URLS: http://127.0.0.1:2380
      ETCD_INITIAL_CLUSTER: ley-etcd=http://127.0.0.1:2380
      ETCD_INITIAL_CLUSTER_TOKEN: ley-etcd-cluster
      ETCD_INITIAL_CLUSTER_STATE: new
    volumes:
      - ./data/etcd:/etcd-data
    ports:
      - "127.0.0.1:2379:2379"
      - "127.0.0.1:2380:2380"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 512M

  minio:
    image: minio/minio:RELEASE.2023-09-23T03-47-50Z
    container_name: ley-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: "你的强密码"
    volumes:
      - ./data/minio:/data
    ports:
      - "127.0.0.1:9000:9000"
      - "127.0.0.1:9001:9001"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 512M

  nats:
    image: nats:2.10-alpine
    container_name: ley-nats
    command: -js -m 8222
    volumes:
      - ./data/nats:/data/jetstream
    ports:
      - "127.0.0.1:4222:4222"
      - "127.0.0.1:8222:8222"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 256M

  # 可选：Jaeger 链路追踪
  jaeger:
    image: jaegertracing/all-in-one:1.47
    container_name: ley-jaeger
    environment:
      COLLECTOR_OTLP_ENABLED: "true"
    ports:
      - "127.0.0.1:16686:16686"
      - "127.0.0.1:4317:4317"
      - "127.0.0.1:4318:4318"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 512M
```

### 3.1 启动基础设施

```bash
cd ~/ley

# 创建数据目录
mkdir -p data/postgres data/redis data/etcd data/minio data/nats data/logs

# 启动基础设施（-d 表示后台运行）
docker compose -f docker-compose.infra.yml up -d

# 查看状态
docker compose -f docker-compose.infra.yml ps

# 查看日志
docker compose -f docker-compose.infra.yml logs -f postgres
```

### 3.2 初始化 MinIO Bucket

```bash
# 进入 MinIO 容器创建 bucket
docker exec -it ley-minio mc alias set local http://127.0.0.1:9000 minioadmin "你的强密码"
docker exec -it ley-minio mc mb local/ley-images --ignore-existing
```

### 3.3 数据库初始化（可选）

如果你已有数据库迁移脚本：

```bash
# 项目内执行（需 Go 环境）
# 或者直接用 psql 导入初始 schema
docker exec -i ley-postgres psql -U ley -d ley < schema.sql
```

## 四、后端部署

### 4.1 修改生产配置

**生产环境必须修改以下配置文件：**

```bash
# data/auth/configs/config.yaml
# data/blog/configs/config.yaml
# data/gateway/configs/config.yaml
# data/entry/configs/config.yaml        # entry 引导配置（复制自 configs/entry.yaml）
```

**关键修改项：**

1. **数据库密码**：`data.auth.configs.config.yaml` 和 `data.blog.configs.config.yaml` 中的 `password: ley123` → 你的强密码
2. **Redis 密码**：加上 `password: "你的强密码"`
3. **JWT Secret**：`jwt.secret` 和 `gateway.middlewares.jwt.signingKey` 必须改为** 256 位随机字符串**
4. **日志级别**：`log.level: Info`（生产环境不要开 Debug）
5. **MinIO 密码**：`data/blog/configs/config.yaml` 中的 `access_key_secret`
6. **CORS**：把 `allowOrigins: ["*"]` 改成你的域名：`allowOrigins: ["https://yourdomain.com"]`
7. **Entry 业务配置**（etcd 键 `ley/configs/entry/config.yaml`，用 `bash scripts/seed-etcd.sh` 下发）：
   `jwt.secret` 必须与 auth 完全一致（entry 只校验不签发 token），`redis.db` 用 0 与 auth 黑名单对齐

### 4.2 Docker 部署（推荐）

项目已提供 Dockerfile，直接构建：

```bash
cd ~/ley

# 构建所有后端镜像
make docker-build

# 或者单独构建
docker build -f app/gateway/Dockerfile -t ley-gateway:latest .
docker build -f app/auth/Dockerfile --build-arg GOPROXY=https://goproxy.cn,direct --build-arg SERVICE_NAME=auth -t ley-auth:latest .
docker build -f app/blog/Dockerfile --build-arg GOPROXY=https://goproxy.cn,direct --build-arg SERVICE_NAME=blog -t ley-blog:latest .
docker build -f app/entry/Dockerfile --build-arg GOPROXY=https://goproxy.cn,direct -t ley-entry:latest .
```

> **⚠️ Entry 与 Gateway 端口共存（T13 阶段）**：两者均默认监听 `:8000`。
> 共存部署时先把 `data/entry/configs/config.yaml` 的 `server.http.addr` 改为 `0.0.0.0:8003`
> （`:8001`/`:8002` 已被 auth/blog HTTP 占用）。T14 切换后移除 gateway、entry 恢复 `:8000`。

**修改现有 `docker-compose.yml` 适配生产：**

原 `docker-compose.yml` 使用 `network_mode: host`，这在某些云服务器环境（如 Docker Swarm、部分 VPC）可能有问题。建议改为桥接网络：

```yaml
services:
  gateway:
    build:
      context: .
      dockerfile: app/gateway/Dockerfile
    image: ley-gateway:latest
    container_name: ley-gateway
    # network_mode: host  # 删除这行
    ports:
      - "127.0.0.1:8000:8000"   # 仅暴露给本机，由 Nginx 反向代理
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
    depends_on:
      - auth
      - blog

  auth:
    build:
      context: .
      dockerfile: app/auth/Dockerfile
      args:
        GOPROXY: https://goproxy.cn,direct
        SERVICE_NAME: auth
    image: ley-auth:latest
    container_name: ley-auth
    # network_mode: host
    # Auth 不需要暴露端口，Gateway 通过 Docker 内部网络访问
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped

  blog:
    build:
      context: .
      dockerfile: app/blog/Dockerfile
      args:
        GOPROXY: https://goproxy.cn,direct
        SERVICE_NAME: blog
    image: ley-blog:latest
    container_name: ley-blog
    # network_mode: host
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
```

**⚠️ 重要**：如果使用桥接网络，需要修改配置文件中的 `127.0.0.1` 为 Docker 服务名：

```yaml
# data/auth/configs/config.yaml 和 data/blog/configs/config.yaml
data:
  database:
    host: ley-postgres  # 不是 127.0.0.1
  redis:
    host: ley-redis
etcd:
  endpoints:
    - ley-etcd:2379
nats:
  addr: nats://ley-nats:4222
minio:
  endpoint: ley-minio:9000
```

把所有服务放在同一个 Docker Compose 文件中，让它们共享同一个 Docker 网络。

### 4.3 启动后端

```bash
# 如果使用桥接网络，需要把基础设施和后端放在同一个 docker-compose.yml
docker compose up -d

# 查看日志
docker compose logs -f gateway
docker compose logs -f auth
docker compose logs -f blog
docker compose logs -f entry   # entry 需先在 data/entry/configs/config.yaml 改 8003 端口并加入 compose 服务列表
```

### 4.4 验证后端健康

```bash
# 健康检查
curl http://127.0.0.1:8000/healthz
# 应返回 HTTP 200

# 测试 API
curl http://127.0.0.1:8000/api/v1/site/config
# 应返回站点配置 JSON
```

## 五、前端部署

> **当前状态（2026-09）**：前端为 **React 19 纯 SPA**（目录 `web/`，Vite 8 构建），无 SSR——产物为 `web/dist/` 静态文件，用 Nginx 等静态托管即可。旧 Nuxt 4 已删除。

### 5.1 构建生产版本

```bash
cd web

# 安装依赖（pnpm 10+）
pnpm install

# 构建（tsc -b && vite build，输出 dist/）
pnpm build
```

### 5.2 部署方式：静态托管（唯一方式，纯 SPA）

**推荐 Nginx 直接托管 `web/dist/`（复用 CD 工作流产物 `frontend-dist`）：**

```nginx
# /etc/nginx/sites-available/ley-web.conf
server {
    listen 443 ssl http2;
    server_name blog.yourdomain.com;

    root /var/www/ley;                # web/dist 解压至此
    index index.html;

    # SPA fallback：所有前端路由交给 index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 静态资源长期缓存（Vite 产物带 hash）
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # API 反向代理（客户端请求同源 /api/v1）
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

```bash
# 服务器上部署
scp -r frontend-dist/* user@server:/var/www/ley/
sudo systemctl reload nginx
```

> **API 地址说明**：SPA 通过同源 `/api/v1` 访问 Gateway（见 vite.config.ts 代理与 Nginx `/api/` 反代）；如需 Web-API 基址注入，请在构建时提供 `VITE_API_BASE` 环境变量（可选，默认同源）。

### 5.3 Docker 部署前端（可选）

纯静态文件只需 Nginx 容器：

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install
COPY . .
RUN pnpm build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
```

```bash
cd web
docker build -t ley-frontend:latest .
docker run -d -p 127.0.0.1:3000:3000 --name ley-frontend ley-frontend:latest
```

## 六、反向代理配置

### 6.1 Caddy（推荐，自动 HTTPS）

```bash
# 安装 Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy
```

**Caddyfile**：

```caddy
yourdomain.com {
    # 前端 Nuxt
    reverse_proxy 127.0.0.1:3000

    # API 直接透传给 Gateway
    handle_path /api/* {
        reverse_proxy 127.0.0.1:8000
    }

    # 静态文件缓存（可选）
    @static {
        path *.js *.css *.png *.jpg *.jpeg *.gif *.svg *.webp *.woff *.woff2
    }
    header @static {
        Cache-Control "public, max-age=31536000, immutable"
    }

    # 安全响应头
    header {
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    # 自动 HTTPS（Caddy 自动申请 Let's Encrypt 证书）
    tls your-email@example.com
}
```

```bash
sudo caddy reload
```

### 6.2 Nginx（传统方案）

```bash
sudo apt install -y nginx
sudo systemctl enable nginx
```

**`/etc/nginx/sites-available/ley`**：

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;  # HTTP 强制跳 HTTPS
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    # SSL 证书（需自行申请）
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;

    # 前端 Nuxt
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # API Gateway
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 30s;
        proxy_connect_timeout 30s;
    }

    # 静态文件缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|svg|webp|woff|woff2)$ {
        proxy_pass http://127.0.0.1:3000;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/ley /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

## 七、SSL 证书

### 7.1 Certbot（配合 Nginx）

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# 自动续期已默认配置，验证：
sudo certbot renew --dry-run
```

### 7.2 Caddy（自动 HTTPS，无需手动配置证书）

Caddy 内置 Let's Encrypt 支持，只需确保 `yourdomain.com` 的 DNS A 记录指向服务器 IP，Caddy 会自动申请和续期证书。

## 八、完整 Docker Compose（单文件部署）

如果你想用一个 `docker-compose.yml` 文件部署**所有服务**（基础设施 + 后端），可以创建：

```yaml
version: '3.8'

services:
  # ===== 基础设施 =====
  postgres:
    image: postgres:15-alpine
    container_name: ley-postgres
    environment:
      POSTGRES_DB: ley
      POSTGRES_USER: ley
      POSTGRES_PASSWORD: "你的强密码"
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    networks:
      - ley-network
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    container_name: ley-redis
    command: redis-server --appendonly yes --requirepass "你的强密码"
    volumes:
      - ./data/redis:/data
    networks:
      - ley-network
    restart: unless-stopped

  etcd:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: ley-etcd
    environment:
      ETCD_NAME: ley-etcd
      ETCD_DATA_DIR: /etcd-data
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_ADVERTISE_CLIENT_URLS: http://ley-etcd:2379
      ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
      ETCD_INITIAL_ADVERTISE_PEER_URLS: http://ley-etcd:2380
      ETCD_INITIAL_CLUSTER: ley-etcd=http://ley-etcd:2380
      ETCD_INITIAL_CLUSTER_TOKEN: ley-etcd-cluster
      ETCD_INITIAL_CLUSTER_STATE: new
    volumes:
      - ./data/etcd:/etcd-data
    networks:
      - ley-network
    restart: unless-stopped

  minio:
    image: minio/minio:latest
    container_name: ley-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: "你的强密码"
    volumes:
      - ./data/minio:/data
    networks:
      - ley-network
    restart: unless-stopped

  nats:
    image: nats:2.10-alpine
    container_name: ley-nats
    command: -js -m 8222
    volumes:
      - ./data/nats:/data/jetstream
    networks:
      - ley-network
    restart: unless-stopped

  # ===== 后端服务 =====
  auth:
    build:
      context: .
      dockerfile: app/auth/Dockerfile
      args:
        GOPROXY: https://goproxy.cn,direct
        SERVICE_NAME: auth
    image: ley-auth:latest
    container_name: ley-auth
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    networks:
      - ley-network
    restart: unless-stopped
    depends_on:
      - postgres
      - redis
      - etcd

  blog:
    build:
      context: .
      dockerfile: app/blog/Dockerfile
      args:
        GOPROXY: https://goproxy.cn,direct
        SERVICE_NAME: blog
    image: ley-blog:latest
    container_name: ley-blog
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    networks:
      - ley-network
    restart: unless-stopped
    depends_on:
      - postgres
      - redis
      - etcd
      - minio
      - nats

  gateway:
    build:
      context: .
      dockerfile: app/gateway/Dockerfile
    image: ley-gateway:latest
    container_name: ley-gateway
    ports:
      - "127.0.0.1:8000:8000"
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    networks:
      - ley-network
    restart: unless-stopped
    depends_on:
      - auth
      - blog

  # ===== 前端（纯 SPA 静态托管，通常用 Nginx 独立部署，见第五章；此处为可选容器方案） =====
  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    image: ley-frontend:latest
    container_name: ley-frontend
    ports:
      - "127.0.0.1:3000:3000"
    networks:
      - ley-network
    restart: unless-stopped

networks:
  ley-network:
    driver: bridge
```

**启动**：

```bash
cd ~/ley
mkdir -p data/postgres data/redis data/etcd data/minio data/nats data/logs

# 修改配置文件中的 host 为服务名（ley-postgres, ley-redis 等）
# 然后
sudo docker compose up -d
```

## 九、安全加固清单

部署完成后，逐项检查：

- [ ] **JWT Secret**：已修改为 256 位随机字符串
- [ ] **数据库密码**：非默认 `ley123`
- [ ] **Redis 密码**：已设置
- [ ] **MinIO 密码**：非默认 `minioadmin`
- [ ] **CORS**：`allowOrigins` 限定为你的域名，非 `*`
- [ ] **防火墙**：仅开放 443 和 80，其他端口（3000, 8000, 9000, 9001）仅允许本机访问
- [ ] **日志**：生产环境日志级别为 Info，不输出 Debug
- [ ] **自动重启**：Docker 服务配置了 `restart: unless-stopped`
- [ ] **备份**：配置了 `data/` 目录的定期备份（PostgreSQL + 文件）

## 十、常见问题

### Q1: Gateway 无法连接 Auth/Blog

检查 etcd 是否注册成功：
```bash
docker exec -it ley-etcd etcdctl get --prefix ley
```
如果没有服务注册，检查 auth/blog 是否能连上 etcd。

### Q2: 前端调用 API 报 CORS 错误

检查 `gateway/configs/config.yaml` 中的 `allowOrigins` 是否包含你的前端域名。

### Q3: 上传图片失败

检查 MinIO bucket 是否创建，以及 blog 配置中的 `minio.endpoint` 是否正确。

### Q4: 如何更新部署？

```bash
# 后端更新
cd ~/ley && git pull
make docker-build
sudo docker compose up -d

# 前端更新（重新构建静态产物 → 同步到 Nginx 根目录）
cd ~/ley/web && git pull
pnpm install && pnpm build
rsync -av --delete dist/ /var/www/ley/
sudo systemctl reload nginx
```

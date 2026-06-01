# GitHub Actions CI/CD 配置说明

本项目包含两套自动化工作流：

| 工作流 | 文件 | 触发条件 | 用途 |
|---|---|---|---|
| **CI** | `.github/workflows/ci.yml` | `push` / `pull_request` | 编译检查、单元测试、前端构建、Docker 镜像构建验证 |
| **CD** | `.github/workflows/deploy.yml` | `push` 到 `main` / 手动触发 | 构建并推送 Docker 镜像、SSH 部署到服务器 |

---

## 工作流详情

### CI (`ci.yml`)

包含三个并行 Job：

1. **backend** — Go 后端
   - 安装 protoc v25.3 + Go 1.26
   - `make init` → `make api config internal_proto` → `make wire`
   - `make test-unit`（运行单元测试 + 覆盖率）
   - `make build && make build-gateway`

2. **frontend** — Nuxt 前端
   - 安装 Node 22 + pnpm 10
   - `pnpm install --frozen-lockfile`
   - `pnpm build`

3. **docker** — Docker 镜像构建检查
   - 构建 `ley-builder`、`ley-auth`、`ley-blog`、`ley-gateway` 镜像
   - **不推送**，仅验证 Dockerfile 可用性
   - 启用 BuildKit 缓存加速

### CD (`deploy.yml`)

包含两个串行 Job：

1. **build-and-push** — 构建并推送镜像到 GitHub Container Registry (GHCR)
   - 镜像标签：`latest` + `${{ github.sha }}`
   - 仓库地址：`ghcr.io/<你的用户名>/ley-{service}`

2. **deploy** — SSH 到远程服务器执行部署
   - 服务器端执行：`git pull` → `docker-compose pull` → `docker-compose up -d`
   - 自动清理 7 天前的旧镜像
   - 部署完成后进行 HTTP 健康检查 (`/api/v1/site/config`)

---

## 必需配置：仓库 Secrets

在 GitHub 仓库页面 → **Settings** → **Secrets and variables** → **Actions** → **New repository secret** 中添加以下密钥：

| Secret 名称 | 说明 | 示例 |
|---|---|---|
| `SSH_PRIVATE_KEY` | 服务器 SSH 私钥（用于免密登录） | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `SERVER_HOST` | 部署目标服务器 IP 或域名 | `123.45.67.89` |
| `SERVER_USER` | SSH 登录用户名 | `root` 或 `deploy` |
| `SERVER_PORT` | SSH 端口（可选，默认 22） | `22` |
| `COMPOSE_PROJECT` | 服务器上项目路径（可选，默认 `~/ley`） | `/opt/ley` |

### 配置步骤

1. **生成部署专用 SSH 密钥对**（不要在服务器上复用你的个人密钥）：
   ```bash
   ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/ley_deploy
   ```

2. **将公钥添加到服务器的 `~/.ssh/authorized_keys`**：
   ```bash
   ssh-copy-id -i ~/.ssh/ley_deploy.pub root@你的服务器IP
   ```

3. **将私钥内容复制到 GitHub Secrets**：
   ```bash
   cat ~/.ssh/ley_deploy | pbcopy  # macOS
   # 或
   cat ~/.ssh/ley_deploy | clip     # Windows
   ```
   在 GitHub 上新建 Secret，名称为 `SSH_PRIVATE_KEY`，粘贴私钥内容。

4. **添加其他 Secrets**：`SERVER_HOST`、`SERVER_USER` 等。

---

## 手动触发部署

除了自动推送 `main` 分支触发外，你也可以在 GitHub 页面手动触发：

1. 进入仓库 → **Actions** → **CD**
2. 点击右侧 **Run workflow**
3. 选择分支（默认 `main`）和部署目标（`production` / `staging`）
4. 点击 **Run workflow**

---

## 常见问题

### Q: CI 中 Docker 构建失败，提示 "host network mode" 不支持？
A: `docker-compose.yml` 使用了 `network_mode: host`，这在 GitHub Actions 的 Ubuntu runner 上不可用。但 CI 工作流只执行 `docker build`（不运行容器），所以不影响。如果运行时测试需要网络，请改用端口映射模式。

### Q: 如何切换到 Docker Hub 而不是 GHCR？
A: 修改 `deploy.yml` 中的 `REGISTRY` 和登录步骤：
```yaml
env:
  REGISTRY: docker.io
  IMAGE_PREFIX: docker.io/你的用户名
```
并添加 `secrets.DOCKER_USERNAME` 和 `secrets.DOCKER_PASSWORD`。

### Q: 我只想部署到多台服务器怎么办？
A: 在 `deploy.yml` 的 `deploy` job 中使用矩阵策略：
```yaml
strategy:
  matrix:
    server: [server1, server2]
```
并对应配置 `secrets.SERVER_HOST_1`、`secrets.SERVER_HOST_2` 等。

### Q: 前端构建的环境变量在哪里设置？
A: CI 中的前端构建使用占位地址，不影响实际部署。生产环境的前端环境变量应在服务器上的 Nginx/PM2 配置中设置，或在构建 Docker 镜像时通过 `ARG` / `ENV` 传入。

---

## 扩展建议

- **添加缓存预热**：在 CI 中缓存 `~/go/pkg/mod` 和 `pnpm store`，已配置
- **添加代码扫描**：集成 `gosec`（Go 安全扫描）和 `eslint`
- **添加集成测试**：在 `make test-integration` 可用的前提下，添加集成测试 job
- **添加通知**：部署成功后发送飞书/钉钉/Slack 消息

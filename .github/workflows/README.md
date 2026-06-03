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

**前后端分离部署**，包含四个 Job，分为两条并行流水线：

**后端流水线：**
1. **build-backend** — 构建 Go 服务镜像并推送到 **Harbor 私有仓库**
   - 镜像标签：`latest` + `${{ github.sha }}`
   - 仓库地址通过 `secrets.HARBOR_REGISTRY` 动态注入，**不在仓库中暴露任何 Registry 地址**
   - 默认项目名：`ley`

2. **deploy-backend** — SSH 到**后端服务器**执行部署
   - 服务器端执行：`git pull` → `docker login Harbor` → `docker pull` → `docker tag` → `docker-compose up -d`
   - 镜像从 Harbor 拉取后重命名为本地标签（`ley-auth:latest` 等），不与 `docker-compose.yml` 中硬编码的 Registry 地址耦合
   - 自动清理 7 天前的旧镜像
   - 部署完成后进行 HTTP 健康检查 (`/api/v1/site/config`)

**前端流水线：**
3. **build-frontend** — 构建 Nuxt 静态站点
   - `pnpm install` → `pnpm generate`
   - 产物保存在 `web/.output/public`
   - 上传到 Artifact（供 deploy-frontend 下载）

4. **deploy-frontend** — 通过 SSH/rsync 部署到**前端服务器**
   - 从 Artifact 下载构建产物
   - `rsync -avz --delete` 增量同步到 Nginx 根目录
   - 远程重载 Nginx
   - 部署完成后进行 HTTP 健康检查 (`/`)

---

## 必需配置：仓库 Secrets

在 GitHub 仓库页面 → **Settings** → **Secrets and variables** → **Actions** → **New repository secret** 中添加以下密钥：

| Secret 名称 | 说明 | 示例 |
|---|---|---|
| `SSH_PRIVATE_KEY` | 后端服务器 SSH 私钥（用于免密登录） | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `SERVER_HOST` | **后端**服务器 IP 或域名 | `123.45.67.89` |
| `SERVER_USER` | **后端**SSH 登录用户名 | `root` 或 `deploy` |
| `SERVER_PORT` | **后端**SSH 端口（可选，默认 22） | `22` |
| `COMPOSE_PROJECT` | **后端**服务器上项目路径（可选，默认 `~/ley`） | `/opt/ley` |
| `FRONTEND_SSH_PRIVATE_KEY` | **前端**服务器 SSH 私钥（可选，默认与后端共用 `SSH_PRIVATE_KEY`） | 同上 |
| `FRONTEND_SERVER_HOST` | **前端**服务器 IP 或域名 | `223.45.67.89` |
| `FRONTEND_SERVER_USER` | **前端**SSH 登录用户名（可选，默认与后端共用 `SERVER_USER`） | `root` |
| `FRONTEND_SERVER_PORT` | **前端**SSH 端口（可选，默认 22） | `22` |
| `FRONTEND_DEPLOY_PATH` | **前端**Nginx 根目录（可选，默认 `/var/www/ley`） | `/usr/share/nginx/html/ley` |
| `NUXT_PUBLIC_API_BASE` | 前端构建时的 API 基地址（可选，默认 `https://api.blog.poyuan233.cn`） | `https://api.yoursite.com` |
| `HARBOR_REGISTRY` | Harbor 仓库地址（不含协议前缀） | `harbor.example.com:8088` |
| `HARBOR_USERNAME` | Harbor 登录用户名 | `poyuan` |
| `HARBOR_PASSWORD` | Harbor 登录密码 | （你的 Harbor 密码） |

> ⚠️ **安全提示**：所有 Harbor、服务器相关信息（地址、用户名、密码）均通过 Secrets 注入，**不会以任何形式暴露在仓库代码或工作流日志中**（GitHub Actions 会自动将 Secrets 值替换为 `***`）。

> 💡 **简化配置**：如果前后端服务器共用同一套 SSH 密钥和用户名，可以只配置 `SSH_PRIVATE_KEY`、`SERVER_HOST`、`SERVER_USER`，前端部署会自动回退使用这些值。

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

4. **添加 Harbor Secrets**：`HARBOR_REGISTRY`、`HARBOR_USERNAME`、`HARBOR_PASSWORD`。

5. **添加服务器连接 Secrets**：`SERVER_HOST`、`SERVER_USER` 等。

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

### Q: 如何切换到其他私有仓库（如阿里云 ACR、腾讯云 TCR）？
A: 修改仓库 Secrets：
- `HARBOR_REGISTRY` → 你的仓库地址（如 `registry.cn-hangzhou.aliyuncs.com`）
- `HARBOR_USERNAME` / `HARBOR_PASSWORD` → 对应仓库的凭据
- 修改 `deploy.yml` 中的镜像路径前缀（默认是 `/ley/ley-auth`，根据你的仓库命名空间调整）

### Q: 我只想部署到多台服务器怎么办？
A: 在 `deploy.yml` 的 `deploy` job 中使用矩阵策略：
```yaml
strategy:
  matrix:
    server: [server1, server2]
```
并对应配置 `secrets.SERVER_HOST_1`、`secrets.SERVER_HOST_2` 等。

### Q: 前端构建的环境变量在哪里设置？
A: 通过 `secrets.NUXT_PUBLIC_API_BASE` 设置。CD 工作流在 `build-frontend` 步骤中注入该变量，确保前端打包时指向正确的后端 API 地址。

### Q: 我的前端服务器没有 rsync 怎么办？
A: 可以在前端服务器上安装：`sudo apt-get install rsync`（Ubuntu/Debian）或 `sudo yum install rsync`（CentOS）。如果确实无法安装，可改用 `scp -r` 替代，但无法做增量删除。

### Q: 前端部署时 Nginx 重载失败？
A: 确保 GitHub Actions 使用的 SSH 用户有 `sudo` 权限执行 `nginx -t` 和 `systemctl reload nginx`。如果用户不在 sudoers 中，需要在前端服务器上配置免密 sudo：
```bash
# 在前端服务器上执行
sudo visudo
# 添加以下行（将 deployuser 替换为实际用户名）
deployuser ALL=(ALL) NOPASSWD: /usr/sbin/nginx, /bin/systemctl reload nginx
```

### Q: 前后端部署能否只部署其中一个？
A: 当前设计是 `push main` 时前后端同时触发。如果只想部署后端：
1. 临时修改工作流文件，注释掉 `build-frontend` 和 `deploy-frontend` job
2. 或使用 `workflow_dispatch` 手动触发，配合条件判断

更优雅的方案是：通过 `paths` 过滤——修改 `web/` 目录只触发前端，修改 `app/` 只触发后端。如需此优化请告知。

---

## 扩展建议

- **添加缓存预热**：在 CI 中缓存 `~/go/pkg/mod` 和 `pnpm store`，已配置
- **添加代码扫描**：集成 `gosec`（Go 安全扫描）和 `eslint`
- **添加集成测试**：在 `make test-integration` 可用的前提下，添加集成测试 job
- **添加通知**：部署成功后发送飞书/钉钉/Slack 消息

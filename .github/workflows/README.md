# GitHub Actions CI/CD 配置说明

本项目采用 **前后端分离触发** 的 CI/CD 设计：

| 工作流 | 文件 | 触发条件 | 用途 |
|---|---|---|---|
| **CI** | `.github/workflows/ci.yml` | `push` / `pull_request`（全量） | 编译检查、单元测试、前端构建、Docker 镜像构建验证 |
| **CD - Backend** | `.github/workflows/deploy-backend.yml` | `push main` 且修改了**后端代码** | 构建并推送后端 Docker 镜像到 Harbor，SSH 部署到后端服务器 |
| **CD - Frontend** | `.github/workflows/deploy-frontend.yml` | `push main` 且修改了**前端代码** | 构建 Nuxt 静态站点，rsync 部署到前端 Nginx 服务器 |

> 💡 **设计原则**：CI 保持全量检查（编译不省），CD 按需触发（只改前端时不浪费后端镜像构建时间）。

---

## 触发规则

### 修改后端文件时（如 `app/blog/internal/biz/article.go`）

触发的工作流：
- ✅ **CI** — 全量检查（backend + frontend + docker）
- ✅ **CD - Backend** — 构建后端镜像 → 推送 Harbor → 部署后端服务器
- ❌ **CD - Frontend** — **不触发**

### 修改前端文件时（如 `web/app/pages/about.vue`）

触发的工作流：
- ✅ **CI** — 全量检查
- ❌ **CD - Backend** — **不触发**
- ✅ **CD - Frontend** — 构建静态站点 → rsync 到前端服务器

### 修改通用文件时（如 `go.mod`、`Makefile`）

触发的工作流：
- ✅ **CI** — 全量检查
- ✅ **CD - Backend** — 触发（`go.mod` 属于后端路径）
- ❌ **CD - Frontend** — **不触发**

---

## 工作流详情

### CI (`ci.yml`)

包含三个并行 Job：

1. **backend** — Go 后端编译 + 单元测试
2. **frontend** — Nuxt 前端安装依赖 + 构建
3. **docker** — Docker 镜像构建验证（不推送）

### CD - Backend (`deploy-backend.yml`)

包含两个串行 Job：

1. **build-backend** — 构建 auth/blog/gateway 镜像并推送到 Harbor
2. **deploy-backend** — SSH 到后端服务器，从 Harbor 拉取镜像并重命名标签，docker-compose 重启

### CD - Frontend (`deploy-frontend.yml`)

包含两个串行 Job：

1. **build-frontend** — `pnpm generate` 生成静态站点，上传 Artifact
2. **deploy-frontend** — 从 Artifact 下载，rsync 增量同步到前端服务器 Nginx 目录，远程 reload nginx

---

## 必需配置：仓库 Secrets

在 GitHub 仓库页面 → **Settings** → **Secrets and variables** → **Actions** → **New repository secret** 中添加以下密钥：

| Secret 名称 | 说明 | 示例 |
|---|---|---|
| `SSH_PRIVATE_KEY` | **后端**服务器 SSH 私钥 | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `SERVER_HOST` | **后端**服务器 IP 或域名 | `123.45.67.89` |
| `SERVER_USER` | **后端**SSH 用户名 | `root` 或 `deploy` |
| `SERVER_PORT` | **后端**SSH 端口（可选，默认 22） | `22` |
| `COMPOSE_PROJECT` | **后端**服务器项目路径（可选，默认 `~/ley`） | `/opt/ley` |
| `FRONTEND_SSH_PRIVATE_KEY` | **前端**服务器 SSH 私钥（可选，默认与后端共用） | 同上 |
| `FRONTEND_SERVER_HOST` | **前端**服务器 IP 或域名 | `223.45.67.89` |
| `FRONTEND_SERVER_USER` | **前端**SSH 用户名（可选，默认与后端共用） | `root` |
| `FRONTEND_SERVER_PORT` | **前端**SSH 端口（可选，默认 22） | `22` |
| `FRONTEND_DEPLOY_PATH` | **前端**Nginx 根目录（可选，默认 `/var/www/ley`） | `/usr/share/nginx/html/ley` |
| `NUXT_PUBLIC_API_BASE` | 前端构建时的 API 基地址（可选，默认 `https://api.blog.poyuan233.cn`） | `https://api.yoursite.com` |
| `HARBOR_REGISTRY` | Harbor 仓库地址（不含协议前缀） | `harbor.example.com:8088` |
| `HARBOR_USERNAME` | Harbor 登录用户名 | `poyuan` |
| `HARBOR_PASSWORD` | Harbor 登录密码 | （你的 Harbor 密码） |

> ⚠️ **安全提示**：所有 Harbor、服务器相关信息（地址、用户名、密码）均通过 Secrets 注入，**不会以任何形式暴露在仓库代码或工作流日志中**（GitHub Actions 会自动将 Secrets 值替换为 `***`）。

> 💡 **简化配置**：如果前后端服务器共用同一套 SSH 密钥和用户名，可以只配置 `SSH_PRIVATE_KEY`、`SERVER_HOST`、`SERVER_USER`，前端部署会自动回退使用这些值。

### 配置步骤

1. **生成部署专用 SSH 密钥对**：
   ```bash
   ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/ley_deploy
   ```

2. **将公钥添加到服务器的 `~/.ssh/authorized_keys`**（前后端服务器都要）：
   ```bash
   ssh-copy-id -i ~/.ssh/ley_deploy.pub root@后端服务器IP
   ssh-copy-id -i ~/.ssh/ley_deploy.pub root@前端服务器IP
   ```

3. **将私钥内容复制到 GitHub Secrets**：
   ```bash
   cat ~/.ssh/ley_deploy | pbcopy  # macOS
   # 或 Windows: cat ~/.ssh/ley_deploy | clip
   ```
   在 GitHub 上新建 Secret，名称为 `SSH_PRIVATE_KEY`（前后端共用）或分别配置 `SSH_PRIVATE_KEY` + `FRONTEND_SSH_PRIVATE_KEY`。

4. **添加 Harbor Secrets**：`HARBOR_REGISTRY`、`HARBOR_USERNAME`、`HARBOR_PASSWORD`。

5. **添加服务器连接 Secrets**：`SERVER_HOST`、`FRONTEND_SERVER_HOST` 等。

---

## 手动触发部署

除了自动推送 `main` 分支触发外，你也可以在 GitHub 页面手动触发：

1. 进入仓库 → **Actions** → **CD - Backend** 或 **CD - Frontend**
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
- 修改 workflow 中的镜像路径前缀（默认是 `/ley/ley-auth`，根据你的仓库命名空间调整）

### Q: 前端部署时 Nginx 重载失败？
A: 确保 GitHub Actions 使用的 SSH 用户有 `sudo` 权限执行 `nginx -t` 和 `systemctl reload nginx`。如果用户不在 sudoers 中，需要在前端服务器上配置免密 sudo：
```bash
# 在前端服务器上执行
sudo visudo
# 添加以下行（将 deployuser 替换为实际用户名）
deployuser ALL=(ALL) NOPASSWD: /usr/sbin/nginx, /bin/systemctl reload nginx
```

### Q: 修改 `docker-compose.yml` 会触发前端部署吗？
A: **不会**。`docker-compose.yml` 属于后端路径（在 `deploy-backend.yml` 的 `paths` 中定义），只会触发后端 CD。前端 CD 的触发路径只有 `web/**`。

### Q: 我想同时部署前后端怎么办？
A: 使用 **Actions** 页面分别手动触发 **CD - Backend** 和 **CD - Frontend**。或者修改一个前后端都触及的文件（如根目录的 README），但这会同时触发两个 workflow。

---

## 扩展建议

- **添加代码扫描**：集成 `gosec`（Go 安全扫描）和 `eslint`
- **添加集成测试**：在 `make test-integration` 可用的前提下，添加集成测试 job
- **添加通知**：部署成功后发送飞书/钉钉/Slack 消息
- **前端缓存策略**：在 Nginx 中配置静态资源的长期缓存（`Cache-Control: max-age=31536000`）

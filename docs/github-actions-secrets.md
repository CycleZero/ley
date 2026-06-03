# GitHub Actions Secrets 配置清单

> 本文档列出了 Ley 项目 CI/CD 工作流所需的全部 GitHub Actions Secrets。
> 配置位置：GitHub 仓库 → **Settings** → **Secrets and variables** → **Actions** → **New repository secret**

---

## 清单总览

| # | Secret 名称 | 类别 | 必需 | 默认值 | 说明 |
|---|---|---|---|---|---|
| 1 | `HARBOR_REGISTRY` | Harbor | ✅ | 无 | 私有镜像仓库地址 |
| 2 | `HARBOR_USERNAME` | Harbor | ✅ | 无 | Harbor 登录用户名 |
| 3 | `HARBOR_PASSWORD` | Harbor | ✅ | 无 | Harbor 登录密码 |
| 4 | `SSH_PRIVATE_KEY` | 后端服务器 | ✅ | 无 | SSH 私钥（免密登录后端服务器） |
| 5 | `SERVER_HOST` | 后端服务器 | ✅ | 无 | 后端服务器 IP 或域名 |
| 6 | `SERVER_USER` | 后端服务器 | ✅ | 无 | 后端 SSH 用户名 |
| 7 | `SERVER_PORT` | 后端服务器 | ❌ | `22` | 后端 SSH 端口 |
| 8 | `COMPOSE_PROJECT` | 后端服务器 | ❌ | `~/ley` | 后端服务器上项目存放路径 |
| 9 | `FRONTEND_SERVER_HOST` | 前端服务器 | ✅ | 无 | 前端服务器 IP 或域名 |
| 10 | `FRONTEND_SSH_PRIVATE_KEY` | 前端服务器 | ❌ | `SSH_PRIVATE_KEY` | 前端服务器 SSH 私钥（前后端共用时可不配） |
| 11 | `FRONTEND_SERVER_USER` | 前端服务器 | ❌ | `SERVER_USER` | 前端 SSH 用户名 |
| 12 | `FRONTEND_SERVER_PORT` | 前端服务器 | ❌ | `SERVER_PORT` → `22` | 前端 SSH 端口 |
| 13 | `FRONTEND_DEPLOY_PATH` | 前端服务器 | ❌ | `/var/www/ley` | Nginx 静态文件根目录 |
| 14 | `NUXT_PUBLIC_API_BASE` | 前端构建 | ❌ | `https://api.blog.poyuan233.cn` | 前端打包时的 API 基地址 |

> 💡 **简化策略**：如果前后端共用同一台服务器或同一套 SSH 密钥，只需配置带 ✅ 的 5 个核心密钥 + Harbor 3 个，其余留空会自动回退。

---

## 配置步骤

### 第一步：生成 SSH 部署密钥

⚠️ **不要使用你的个人日常 SSH 密钥**，应为 GitHub Actions 单独生成一对。

在你的**本地开发机**（或任意安全机器）上执行：

```bash
# 1. 生成密钥对（Ed25519 算法，更安全）
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/ley_deploy -N ""

# 2. 查看生成的文件
ls ~/.ssh/ley_deploy*
# 预期输出：ley_deploy（私钥）  ley_deploy.pub（公钥）
```

---

### 第二步：配置后端服务器

SSH 登录你的**后端服务器**，执行：

```bash
# 1. 确保有 .ssh 目录
mkdir -p ~/.ssh
chmod 700 ~/.ssh

# 2. 将公钥添加到 authorized_keys
cat >> ~/.ssh/authorized_keys << 'EOF'
# 把你本地生成的 ley_deploy.pub 内容粘贴到这里
EOF
chmod 600 ~/.ssh/authorized_keys

# 3. 验证（从本地执行）
# ssh -i ~/.ssh/ley_deploy root@后端服务器IP
# 应该无需密码直接登录
```

---

### 第三步：配置前端服务器

SSH 登录你的**前端服务器**，执行与第二步**相同的操作**：

```bash
mkdir -p ~/.ssh && chmod 700 ~/.ssh
cat >> ~/.ssh/authorized_keys << 'EOF'
# 同样粘贴 ley_deploy.pub 的内容（可与后端共用同一公钥）
EOF
chmod 600 ~/.ssh/authorized_keys
```

---

### 第四步：前端服务器安装必要软件

确保前端服务器已安装 Nginx 和 rsync：

```bash
# Ubuntu / Debian
sudo apt-get update
sudo apt-get install -y nginx rsync

# 创建部署目录
sudo mkdir -p /var/www/ley
sudo chown -R $USER:$USER /var/www/ley

# 配置 Nginx 站点（示例）
sudo tee /etc/nginx/sites-available/ley << 'EOF'
server {
    listen 80;
    server_name blog.poyuan233.cn;
    root /var/www/ley;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # 静态资源长期缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/ley /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

---

### 第五步：配置免密 sudo（用于 Nginx 重载）

在前端服务器上，给部署用户添加免密 sudo 权限：

```bash
sudo visudo
# 添加以下行（将 deployuser 替换为实际的 SSH 用户名）
# deployuser ALL=(ALL) NOPASSWD: /usr/sbin/nginx, /bin/systemctl reload nginx, /usr/bin/systemctl reload nginx
```

> 如果不配置这一步，前端 CD 部署时会因为 `sudo nginx -s reload` 需要密码而失败。

---

### 第六步：在 GitHub 上添加 Secrets

1. 打开浏览器，进入你的 GitHub 仓库
2. 点击顶部菜单 **Settings**
3. 左侧边栏 → **Secrets and variables** → **Actions**
4. 点击绿色按钮 **New repository secret**
5. 逐个添加以下 Secrets：

#### 6.1 Harbor 密钥（3 个）

| Secret 名称 | Value 示例 | 获取方式 |
|---|---|---|
| `HARBOR_REGISTRY` | `harbor.server.poyuan233.cn:8088` | 你的 Harbor 域名 + 端口 |
| `HARBOR_USERNAME` | `poyuan` | Harbor 登录用户名 |
| `HARBOR_PASSWORD` | `你的Harbor密码` | Harbor 登录密码 |

#### 6.2 后端服务器密钥（4-5 个）

| Secret 名称 | Value 示例 | 获取方式 |
|---|---|---|
| `SSH_PRIVATE_KEY` | `-----BEGIN OPENSSH PRIVATE KEY-----...` | `cat ~/.ssh/ley_deploy` 的完整内容 |
| `SERVER_HOST` | `123.45.67.89` | 后端服务器公网 IP 或域名 |
| `SERVER_USER` | `root` | SSH 登录用户名 |
| `SERVER_PORT` | `22` | SSH 端口（非 22 时才需配置） |
| `COMPOSE_PROJECT` | `/opt/ley` | 后端服务器上 `git clone` 的项目路径（默认 `~/ley`） |

> **SSH_PRIVATE_KEY 粘贴注意事项**：
> - 必须包含完整的 `-----BEGIN...` 到 `-----END...` 部分
> - 包括所有换行符（直接 `cat` 复制粘贴即可）
> - 不要添加额外的空格或注释

#### 6.3 前端服务器密钥（按需配置）

| Secret 名称 | Value 示例 | 说明 |
|---|---|---|
| `FRONTEND_SERVER_HOST` | `223.45.67.89` | 前端服务器公网 IP 或域名 |
| `FRONTEND_SSH_PRIVATE_KEY` | （留空） | 如果与后端共用同一密钥，**不需要填** |
| `FRONTEND_SERVER_USER` | （留空） | 如果与后端用户名相同，**不需要填** |
| `FRONTEND_SERVER_PORT` | （留空） | 如果与后端端口相同，**不需要填** |
| `FRONTEND_DEPLOY_PATH` | `/var/www/ley` | Nginx 根目录（默认 `/var/www/ley`，与第五步一致即可） |

#### 6.4 前端构建密钥（可选）

| Secret 名称 | Value 示例 | 说明 |
|---|---|---|
| `NUXT_PUBLIC_API_BASE` | `https://api.blog.poyuan233.cn` | 前端打包时注入的 API 地址 |

---

## 验证配置

全部添加完成后，可以通过以下方式验证：

### 1. 检查 Secrets 列表

在 GitHub 仓库页面 → **Settings** → **Secrets and variables** → **Actions**，应看到如下列表：

```
HARBOR_REGISTRY            Updated 1 minute ago
HARBOR_USERNAME            Updated 1 minute ago
HARBOR_PASSWORD            Updated 1 minute ago
SSH_PRIVATE_KEY            Updated 2 minutes ago
SERVER_HOST                Updated 2 minutes ago
SERVER_USER                Updated 2 minutes ago
SERVER_PORT                Updated 2 minutes ago   (可选)
COMPOSE_PROJECT            Updated 2 minutes ago   (可选)
FRONTEND_SERVER_HOST       Updated 1 minute ago
FRONTEND_DEPLOY_PATH       Updated 1 minute ago    (可选)
NUXT_PUBLIC_API_BASE       Updated 1 minute ago    (可选)
```

### 2. 本地 SSH 连通性测试

```bash
# 测试后端服务器
ssh -i ~/.ssh/ley_deploy -p 22 root@后端服务器IP "echo '后端连通 OK'"

# 测试前端服务器
ssh -i ~/.ssh/ley_deploy -p 22 root@前端服务器IP "echo '前端连通 OK'"
```

### 3. 触发一次真实部署

修改一个后端文件并推送：

```bash
# 示例：在后端代码里加一行注释
echo "// CI/CD test" >> app/blog/internal/biz/article.go
git add . && git commit -m "ci: 验证 CD-Backend 触发" && git push origin main
```

然后打开 GitHub 仓库 → **Actions** 页面，观察：
- ✅ **CI** 工作流运行
- ✅ **CD - Backend** 工作流出现并运行
- ❌ **CD - Frontend** 工作流**不出现**（因为没改前端文件）

---

## 常见问题

### Q: `SSH_PRIVATE_KEY` 粘贴后部署还是失败？

A: 常见原因：
1. 私钥内容不完整（缺少 `-----BEGIN/END` 标记）
2. 服务器上没有对应的公钥（`authorized_keys`）
3. 服务器 SSH 端口不是 22，但 `SERVER_PORT` 没配置
4. 私钥格式不对（GitHub Actions 需要 OpenSSH 格式，不是 PEM 格式）

修复方法：
```bash
# 检查私钥格式（应以 -----BEGIN OPENSSH PRIVATE KEY----- 开头）
head -n1 ~/.ssh/ley_deploy

# 如果是旧格式，转换一下
ssh-keygen -p -m PEM -f ~/.ssh/ley_deploy -N ""
```

### Q: Harbor 登录失败（401 Unauthorized）？

A: 检查：
1. `HARBOR_REGISTRY` 是否包含协议前缀（**不应该**，只写 `host:port`）
2. `HARBOR_PASSWORD` 是否包含特殊字符（如有，确保复制时没截断）
3. Harbor 中 `ley` 项目是否存在且用户有推送权限

### Q: 前后端能否完全共用一台服务器？

A: 可以。此时只需配置：
- `SERVER_HOST` = `FRONTEND_SERVER_HOST`（同一 IP）
- `FRONTEND_DEPLOY_PATH` = 该服务器上的 Nginx 目录
- `FRONTEND_SSH_PRIVATE_KEY`、`FRONTEND_SERVER_USER`、`FRONTEND_SERVER_PORT` 留空（自动回退到后端值）

### Q: Secrets 配置错了怎么修改？

A: GitHub 不支持直接修改 Secret，只能：
1. 在 Secrets 列表页面点击 Secret 名称旁边的 **Update** 按钮
2. 或者先 **Delete** 再重新 **New repository secret**

---

## 完成检查表

- [ ] 生成了 `ley_deploy` / `ley_deploy.pub` 密钥对
- [ ] 后端服务器的 `~/.ssh/authorized_keys` 包含公钥
- [ ] 前端服务器的 `~/.ssh/authorized_keys` 包含公钥
- [ ] 前端服务器安装了 `nginx` 和 `rsync`
- [ ] 前端服务器配置了 Nginx 站点并指向 `/var/www/ley`
- [ ] 前端服务器配置了免密 sudo（`visudo`）
- [ ] GitHub Secrets 添加了 `HARBOR_REGISTRY`、`HARBOR_USERNAME`、`HARBOR_PASSWORD`
- [ ] GitHub Secrets 添加了 `SSH_PRIVATE_KEY`、`SERVER_HOST`、`SERVER_USER`
- [ ] GitHub Secrets 添加了 `FRONTEND_SERVER_HOST`
- [ ] 本地 `ssh -i ~/.ssh/ley_deploy user@host` 测试通过
- [ ] 执行了一次 `git push` 验证 Actions 触发正常

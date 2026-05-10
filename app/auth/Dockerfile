# ================================
# Stage 1: Builder
# ================================
FROM ley-builder:v1 AS builder

# 设置 Go 代理和构建参数
ARG GOPROXY=https://goproxy.cn,direct
ARG SERVICE_NAME=user



# 设置工作目录
WORKDIR /src

# 优先复制 go.mod 和 go.sum 以利用 Docker 缓存
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
# 大仓模式需要复制整个项目，因为服务间有依赖
COPY . .

# 生成代码（proto、wire等）
RUN make api && make config && make internal_proto && make wire

RUN go build -o /app/server ./app/${SERVICE_NAME}/cmd/${SERVICE_NAME}

# ================================
# Stage 2: Runtime
# ================================
FROM debian:stable-slim

# 安装运行时依赖
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata libc6 && rm -rf /var/lib/apt/lists/*
# 设置时区（可选，默认 UTC）
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN groupadd -g 1000 appgroup && \
    useradd -u 1000 -g appgroup -m appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/server .

# 创建配置目录
RUN mkdir -p /app/data && chown -R appuser:appgroup /app /app/data

# 切换到非 root 用户
USER appuser

# 配置文件挂载点
VOLUME /app/data

# 注意: 端口通过配置文件动态配置，不在此处硬编码
# EXPOSE 仅作为文档说明，实际端口映射在 docker run 或 docker-compose 中指定
# 运行示例: docker run -p <host_port>:<container_port> ...

# 启动服务
CMD ["./server"]
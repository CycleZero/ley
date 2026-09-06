# ================================
# Ley — Entry 入口服务 Dockerfile
# ================================
# 与 app/auth/Dockerfile 同一多阶段模式：
#   Stage 1 builder（ley-builder:v1）：生成 proto/wire 代码后编译
#   Stage 2 runtime（ley-runtime:v1）：仅拷贝二进制 + 数据目录
#
# 与 auth/blog 的唯一差异：entry 的 main 包位于仓库根 module 的
# ./app/entry（无 cmd/ 子目录），且不依赖 SERVICE_NAME build-arg，
# 因此编译路径直接写死 ./app/entry。
#
# 运行期监听端口不在此硬编码：由本地引导配置
# data/entry/configs/config.yaml 的 server.http.addr 决定
# （默认 :8000，即旧 gateway 的入口端口；T13 与 gateway 共存阶段
# 需在引导配置中改 0.0.0.0:8003，见 docker-compose.yml 注释）。

# ================================
# Stage 1: Builder
# ================================
FROM ley-builder:v1 AS builder

ARG GOPROXY=https://goproxy.cn,direct

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 生成 *.pb.go / conf 代码 / wire_gen.go（仓库已 gitignore，镜像内必须现场生成；
# make wire 经 find 自动覆盖 ./app/entry/cmd/entry/wire.go）
RUN make api && make config && make internal_proto && make wire

# entry main 包在 ./app/entry/cmd/entry（对齐微服务 cmd 布局）
RUN go build -o /app/server ./app/entry/cmd/entry

# ================================
# Stage 2: Runtime
# ================================
FROM ley-runtime:v1

COPY --from=builder /app/server /app/server

RUN groupadd -g 1000 appgroup && \
    useradd -u 1000 -g appgroup -m appuser

WORKDIR /app

RUN mkdir -p /app/data && chown -R appuser:appgroup /app /app/data

USER appuser

VOLUME /app/data

# 声明性端口：实际监听由引导配置 server.http.addr 决定（默认 8000，T14 切换后生效）
EXPOSE 8000

CMD ["./server"]

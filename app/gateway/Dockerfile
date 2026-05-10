# ================================
# Stage 1: Builder
# ================================
FROM golang:1.23-alpine AS builder

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

RUN apk add --no-cache git

WORKDIR /src

# 网关是独立 Go module，只复制自身代码
COPY app/gateway/go.mod app/gateway/go.sum ./
RUN go mod download

COPY app/gateway/ ./

RUN go build -o /app/gateway ./cmd/gateway

# ================================
# Stage 2: Runtime
# ================================
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/gateway .

RUN mkdir -p /app/data

VOLUME /app/data

CMD ["./gateway"]

FROM golang:1.26.1-bookworm AS builder

# 设置 Go 代理和构建参数
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}


# 安装必要的构建工具（包含 CGO 编译所需的 gcc 和 musl-dev）
RUN apt-get update && apt-get install -y --no-install-recommends \
    git make bash gcc libc6-dev curl unzip \
    && rm -rf /var/lib/apt/lists/*


RUN PROTOC_ZIP=protoc-25.3-linux-x86_64.zip && \
    curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v25.3/$PROTOC_ZIP && \
    unzip -o $PROTOC_ZIP -d /usr/local bin/protoc && \
    unzip -o $PROTOC_ZIP -d /usr/local include/* && \
    rm -f $PROTOC_ZIP


# 设置工作目录
WORKDIR /dep

# 优先复制 go.mod 和 go.sum 以利用 Docker 缓存
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
# 大仓模式需要复制整个项目，因为服务间有依赖
COPY ./Makefile .

# 生成代码（proto、wire等）
RUN make init

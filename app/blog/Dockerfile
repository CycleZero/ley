# ================================
# Stage 1: Builder
# ================================
FROM ley-builder:v1 AS builder

ARG GOPROXY=https://goproxy.cn,direct
ARG SERVICE_NAME=blog

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make api && make config && make internal_proto && make wire

RUN go build -o /app/server ./app/${SERVICE_NAME}/cmd/${SERVICE_NAME}

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

CMD ["./server"]
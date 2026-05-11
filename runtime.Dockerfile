FROM debian:stable-slim

RUN sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/debian.sources && \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata libc6 \
    && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai



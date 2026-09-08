#!/bin/bash
# V次元微服务更新脚本
# 用法:
#   bash update.sh              # 更新全部服务
#   bash update.sh comment      # 只更新指定服务
set -e

SERVICES=(auth blog entry)

if [ $# -gt 0 ]; then
    # 只更新指定服务
    TARGETS=("$@")
else
    TARGETS=("${SERVICES[@]}")
fi

echo "=== 拉取最新代码 ==="
git pull

echo "=== 构建服务镜像 ==="
for svc in "${TARGETS[@]}"; do
    echo "  构建 $svc ..."
    docker-compose build "$svc"
done

echo "=== 重启服务 ==="
for svc in "${TARGETS[@]}"; do
    echo "  重启 $svc ..."
    docker-compose up -d "$svc"
done

echo "=== 清理旧镜像 ==="
docker image prune -f

echo "=== 服务状态 ==="
docker-compose ps

echo "=== 更新完成 ==="

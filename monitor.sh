#!/bin/bash

REDIS_CMD="redis-cli -h 127.0.0.1 -p 4999 -a 123456"
PG_URI="postgresql://myuser:mypassword@127.0.0.1:5433/database"

prev_redis=0
prev_pg=0
is_first_run=true

echo "开始监控... (按 Ctrl+C 停止)"
echo "========================================================================================="
printf "%-10s | %-15s | %-15s | %-15s | %-15s\n" "时间" "Redis 总量" "Redis 增量/s" "PG 总量" "PG 增量/s"
printf "%-10s | %-15s | %-15s | %-15s | %-15s\n" "----------" "---------------" "---------------" "---------------" "---------------"

while true; do
    current_redis=$($REDIS_CMD XLEN message:inbound 2>/dev/null | tr -d '\r')
    current_pg=$(psql "$PG_URI" -t -c "select count(*) from messages;" 2>/dev/null | xargs)

    current_redis=${current_redis:-0}
    current_pg=${current_pg:-0}

    if [ "$is_first_run" = true ]; then
        diff_redis=0
        diff_pg=0
        is_first_run=false
    else
        diff_redis=$((current_redis - prev_redis))
        diff_pg=$((current_pg - prev_pg))
    fi

    current_time=$(date '+%H:%M:%S')
    
    printf "%-10s | %-15s | %-15s | %-15s | %-15s\n" "$current_time" "$current_redis" "+$diff_redis" "$current_pg" "+$diff_pg"

    prev_redis=$current_redis
    prev_pg=$current_pg
    
    sleep 1
done
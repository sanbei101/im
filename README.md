# Go IM

一个使用 Go、Pebble 和固定逻辑槽分片的即时通讯服务。

## 架构

- `gateway` 是统一入口:维护 WebSocket 长连接和用户会话,同时把 `/api/` 请求转发到 API 节点;使用协议层 ping/pong 控制帧保活,每 30s 主动 ping,5s 未收到 pong 即断开并注销在线状态。
- `api` 同时承载控制面和消息面。控制面(用户、设备、好友、房间、成员、成员设置、已读游标)全部读写 PostgreSQL;消息面(消息、幂等、room_seq、置顶、表情、归档)使用本节点 Pebble。
- 所有 API 节点共享同一个 PostgreSQL,控制面请求可以落到任意节点。
- API 节点按 `hash(room_id) % slots` 拥有消息分片;Gateway 把消息、历史、搜索、置顶、表情等房间请求转发到归属节点。
- Gateway 与 API 之间使用 Kitex 双向流式 gRPC,底层使用 Netpoll,协议由 Kitex 生成 fastpb 编解码代码。
- 消息按照 `room_seq` 排序、分页和断线补拉;消息写入 Pebble 后把会话头(last_seq/last_message)异步入有界队列投影到 PostgreSQL,不阻塞 ACK。
- 跨 API 节点通知通过 PostgreSQL `LISTEN/NOTIFY` 分发,客户端断线后仍以 HTTP/room_seq 补拉为准。
- Pebble checkpoint 由 API 进程定期生成和保留。

## 目录

```text
cmd/api       API 进程(控制面 + 消息面)
cmd/gateway   WebSocket Gateway + HTTP Edge 进程
internal/api  HTTP Handler、Kitex Server 和业务逻辑
internal/gateway WebSocket、Kitex 客户端和会话
internal/store Pebble 消息存储、PostgreSQL 控制面存储与迁移
pkg           配置、分片、JWT、日志
proto         Protobuf 源文件与 Kitex 生成代码(proto/pb)
```

## 本地运行

需要 Go 1.27、Kitex 和 protoc。Kitex 协议代码生成(默认走 prutal,不再产出 `*.pb.fast.go`):

```bash
brew install protobuf
go install github.com/cloudwego/kitex/tool/cmd/kitex@latest
make gen
```

准备 PostgreSQL 并启动 API 和 Gateway:

```bash
createdb im
cp config.example.yaml config.yaml
# 填写 postgres.url、jwt_secret 与 storage 凭据；对象存储是 API 启动的硬依赖
go run ./cmd/api
go run ./cmd/gateway
```

API 启动时会自动执行 `internal/store/migrations` 下的迁移。多个 API 节点必须指向同一个 PostgreSQL。

二进制部署:

```bash
make build
./bin/api
./bin/gateway
```

或后台启动并查看日志:

```bash
make dev
make stop
```

默认端口:API HTTP `8801`、WebSocket `8800`、API 内部流 `9000`、pprof `6061`/`6062`、PostgreSQL `5432`。HTTP 与 pprof 监听地址可通过 `api.http_addr` / `api.pprof_addr` 或对应环境变量覆盖。

多 API 节点时,`gateway.api_http_addrs` 必须按 `node_index` 顺序列出每个 API 节点的 HTTP 地址,并与 `gateway.api_addrs` 的顺序一致:

```yaml
api:
  node_index: 1
  node_count: 2
  http_addr: ":8802"

gateway:
  api_addrs:
    - "api-1:9000"
    - "api-2:9001"
  api_http_addrs:
    - "api-1:8801"
    - "api-2:8802"

postgres:
  url: "postgres://im:im@postgres:5432/im?sslmode=disable"
```

客户端只需要连接 Gateway:WebSocket 走 `/ws`,HTTP 走 `/api/v1/...`。带房间语义的 HTTP 请求由 Gateway 转发到归属 API;用户、好友、房间控制请求可以在任意 API 节点处理,因为控制面在 PostgreSQL。

房间相关 HTTP 请求(路径或 `room_id` 查询参数/JSON 字段)只由 Gateway 转发到该房间的归属节点;API 节点只处理本地槽位的数据。单聊房间 ID 由双方用户 ID 确定性生成;群聊创建必须携带客户端生成的 `room_id`,以便在创建前完成路由。

## 数据写入约定

Pebble Value 使用 `encoding/binary` 手写编码,不使用 JSON。消息写入会在同一个 Batch 中更新消息、房间序号和幂等记录;只有同步提交成功后才返回成功结果。PostgreSQL 表结构见 `internal/store/migrations`。

消息幂等键为 `room_id + sender_id + client_msg_id`。相同请求重试返回原消息结果,使用相同键提交不同内容会返回冲突。

## 消息链路

一条消息从客户端到落盘的完整路径:

```text
客户端 ──WS──▶ Gateway(8800) ──Kitex bidi stream(9000)──▶ API
                     │                                        │
                     │                                  Pebble Batch(Sync)
                     │                                  message/room/dedup 同批提交
                     │◀────SendResultBatch(mid,room_seq)──────┤
                     ▼
              ack 帧回客户端;Pebble 提交成功才发 ack

推送:API 写盘后按 PostgreSQL 成员列表推送 → RoomPushBatch → Gateway 分发到成员 WS 连接
补拉:GET /api/v1/messages/history?room_id=&before_seq=&page_size=
会话列表:PostgreSQL 房间/成员/已读/会话头;消息正文仍从归属 API 的 Pebble 读取
```

- `before_seq` 是排他上界:返回 `room_seq < before_seq` 的消息,用于按本地最新序号向前补拉。
- 历史消息字段与推送帧一致:`msg_id / client_msg_id / sender_id / room_id / room_seq / server_time / msg_type / payload / ext`。
- 消息幂等键 `room_id + sender_id + client_msg_id`;推送帧带 `client_msg_id`,接收端可自去重。

## 测试、Benchmark 与 Pprof

所有检查通过 `Makefile` 执行

```bash
make verify       # lint、全部测试(-race)、benchmark
make bench        # Pebble、PostgreSQL、异步会话头投影和编解码 benchmark
make pprof        # 真实批量写入并生成 CPU、heap、mutex、block profile
make analyze      # 输出 pprof top 热点
```

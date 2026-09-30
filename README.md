# Go IM

一个使用 Go、Pebble 和固定逻辑槽分片的即时通讯服务。

## 架构

- `gateway` 维护 WebSocket 长连接和用户会话。
- `api` 提供 HTTP 接口和房间分片存储。
- Gateway 与 API 之间使用 Kitex 双向流式 gRPC，底层使用 Netpoll，协议由 Kitex 生成 fastpb 编解码代码。
- 消息、房间、成员和幂等记录使用 Pebble 本地持久化。
- 房间通过 `hash(room_id) % slots` 路由到 API 节点。
- 消息按照 `room_seq` 排序、分页和断线补拉。
- API 定期生成 Pebble checkpoint，不依赖 PostgreSQL、Redis、外部 MQ 或 Raft。

## 目录

```text
cmd/api       API 进程
cmd/gateway   WebSocket Gateway 进程
internal/api  HTTP 和 Kitex 服务
internal/gateway WebSocket、Kitex 客户端和会话
internal/store Pebble 与固定二进制记录编码
kitex_gen     Kitex/Protobuf 生成代码
pkg           配置、分片、JWT、日志
proto         Protobuf 源文件
```

## 本地运行

需要 Go 1.27、Kitex 和 protoc。Kitex 协议代码生成(默认走 prutal，不再产出 `*.pb.fast.go`)：

```bash
brew install protobuf
go install github.com/cloudwego/kitex/tool/cmd/kitex@latest
kitex -type protobuf -streamx -module github.com/sanbei101/im -gen-path kitex_gen proto/im/v1/gateway.proto
```

启动 API 和 Gateway：

```bash
go run ./cmd/api
go run ./cmd/gateway
```

或使用 Docker Compose：

```bash
docker compose up --build
```

默认端口：HTTP `8801`、WebSocket `8800`、API 内部流 `9000`。

## 数据写入约定

Pebble Value 使用 `encoding/binary` 手写编码，不使用 JSON。消息写入会在同一个 Batch 中更新消息、房间序号和幂等记录；只有同步提交成功后才返回成功结果。

消息幂等键为 `room_id + sender_id + client_msg_id`。相同请求重试返回原消息结果，使用相同键提交不同内容会返回冲突。

## 消息链路

一条消息从客户端到落盘的完整路径：

```text
客户端 ──WS──▶ Gateway(8800) ──Kitex bidi stream(9000)──▶ API
                     │                                        │
                     │                                  Pebble Batch(Sync)
                     │                                  message/room/dedup 同批提交
                     │◀────SendResultBatch(mid,room_seq)──────┤
                     ▼
              ack 帧回客户端；Pebble 提交成功才发 ack

推送：API 写盘后按房间成员推送 → PushFrame{batch} → Gateway 分发到成员 WS 连接
补拉：GET /api/v1/messages/history?room_id=&before_seq=&page_size=
```

- `before_seq` 是排他上界：返回 `room_seq < before_seq` 的消息，用于按本地最新序号向前补拉。
- 历史消息字段与推送帧一致：`msg_id / client_msg_id / sender_id / room_id / room_seq / server_time / msg_type / payload / ext`。
- 消息幂等键 `room_id + sender_id + client_msg_id`；推送帧带 `client_msg_id`，接收端可自去重。

## 测试、Benchmark 与 Pprof

所有检查通过 `Makefile` 执行

```bash
make verify       # gofmt、go vet、全部测试、race、benchmark
make bench        # 真实 Pebble 写入和历史读取 benchmark
make pprof        # 真实批量写入并生成 CPU、heap、mutex、block profile
make analyze      # 输出 pprof top 热点
```

SDK 集成测试（需先启动 `./cmd/api` 与 `./cmd/gateway`）：

```bash
cd sdk && pnpm install
API_BASE_URL=http://127.0.0.1:8801 WS_GATEWAY_URL=ws://127.0.0.1:8800/ws pnpm test
```

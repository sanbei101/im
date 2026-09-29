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

需要 Go 1.27、Kitex 和 protoc。Kitex 协议代码生成：

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

## 测试、Benchmark 与 Pprof

所有检查通过 `Makefile` 执行

```bash
make verify       # gofmt、go vet、全部测试、race、benchmark
make bench        # 真实 Pebble 写入和历史读取 benchmark
make pprof        # 真实批量写入并生成 CPU、heap、mutex、block profile
make analyze      # 输出 pprof top 热点
```

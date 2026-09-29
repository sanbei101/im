# IM 项目开发规范

## 架构

本项目使用 Go、Pebble 和按 `room_id` 的固定逻辑槽分片，不依赖 PostgreSQL、Redis、外部消息队列、Raft 或服务注册中心。

- `gateway` 负责 WebSocket 长连接、鉴权、会话管理、按 `room_id` 路由和批量发送。
- `api` 负责 HTTP API、房间分片写入、用户/好友元数据和 Pebble 读写。
- Gateway 与 API 之间使用 Kitex 双向流式 gRPC，底层使用 Netpoll，消息使用 Kitex 生成的 fastpb 编解码。
- 每个 API 节点只拥有配置中分配的逻辑槽；`hash(room_id) % slot_count` 得到槽，再通过静态拓扑找到节点。
- 消息、房间、成员、消息幂等记录必须写入同一个 Pebble Batch；只有本地持久化成功后才能返回成功 ACK。
- 实时推送不是可靠存储，断线和推送失败统一通过房间消息序号补拉。
- Pebble checkpoint 由 API 进程定期生成和保留；当前不实现 `imctl`。

## 目录

保持目录扁平，优先在现有包中增加文件，不为单一职责新建目录。

```text
cmd/api       API 入口
cmd/gateway   Gateway 入口
internal/api  HTTP Handler、Kitex Server 和 API 业务逻辑
internal/gateway WebSocket、Kitex Client、会话和批量发送
internal/store Pebble、Key 编码、用户、房间、消息和备份
pkg           配置、分片、JWT、日志和通用工具
proto         Kitex/fastpb 协议源文件
```

禁止重新引入 `internal/model`、`internal/domain`、`internal/repository`、`internal/usecase` 等层级。协议结构、存储记录和业务输入在可以复用时直接复用；只有在边界确实不同且转换有明确价值时才定义新结构。

## 代码风格

- 代码优先简洁、直白、可读；小函数只做一件事。
- 错误必须带上下文，禁止静默忽略；重复错误使用包级 sentinel 或 `errors.Is`。
- 所有 goroutine、channel、Pebble iterator 和 stream 都必须有明确的生命周期和关闭路径。
- 所有队列同时限制数量和字节数，禁止无界缓存和无限创建 goroutine。
- 不在 Handler、Gateway 或 Store 中复制相同的业务规则。
- Key 编码集中维护，不在业务代码中手写字符串前缀。
- Pebble 读结果必须及时释放；Batch、Iterator、Snapshot 使用 `defer Close`。
- 任何写路径都要说明幂等键、顺序保证和 ACK 时机。
- 先写可验证的最小实现，再做性能优化；优化必须有 benchmark 或指标依据。

## 协议和存储

- fastpb/Kitex 编解码只使用 `proto3`，不要使用 protobuf edition、`Any` 或未验证的 known types。
- UUID 统一使用 Go 标准库 `uuid`，不要引入第三方 UUID 包。
- 消息使用 `room_seq` 做历史分页和断线补拉，不使用时间戳作为唯一游标。
- 消息幂等键为 `room_id + sender_id + client_msg_id`，相同键不同内容必须返回冲突。
- 客户端成功 ACK 表示消息已写入本地 Pebble；进入 channel、进入 gRPC 流或开始处理不代表成功。
- 任何新增字段必须考虑旧 checkpoint 的 schema 版本和恢复行为。

## 验证

提交前至少运行：

```bash
gofmt -w .
go vet ./...
go test ./...
```

涉及存储、重试、分片或协议的修改必须补充对应的故障、幂等或边界测试。

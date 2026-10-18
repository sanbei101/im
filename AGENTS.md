# IM 项目开发规范

## 架构

本项目使用 Go、PostgreSQL 和按 `room_id` 的固定逻辑槽分片;消息面使用 Pebble,控制面使用 PostgreSQL,不依赖 Redis、外部消息队列、Raft 或服务注册中心。

- `gateway` 负责 WebSocket 长连接、鉴权、会话管理、按 `room_id` 路由和批量发送。
- `api` 负责 HTTP API、控制面 PostgreSQL 读写、房间分片内的 Pebble 消息读写。
- Gateway 与 API 之间使用 Kitex 双向流式 gRPC,底层使用 Netpoll,消息使用 Kitex 生成的 fastpb 编解码。
- 每个 API 节点只拥有配置中分配的逻辑槽;`hash(room_id) % slot_count` 得到槽,再通过静态拓扑找到节点。
- 消息、room_seq 和消息幂等记录必须写入同一个 Pebble Batch;只有本地持久化成功后才能返回成功 ACK。
- 用户、设备、好友、申请、黑名单、房间、成员、成员设置和已读游标统一存 PostgreSQL;所有 API 节点共享同一数据库。
- 会话列表只从 PostgreSQL 返回房间、成员设置和已读游标;最新消息与预览由客户端本地维护,新设备按需懒加载。跨节点通知使用 `LISTEN/NOTIFY`。
- 实时推送不是可靠存储,断线和推送失败统一通过房间消息序号补拉。
- Pebble checkpoint 由 API 进程定期生成和保留;当前不实现 `imctl`。
- 多 API 节点时,`gateway.api_addrs` 与 `gateway.api_http_addrs` 都按 `node_index` 排列;客户端只连接 Gateway,`/ws` 走长连接,带 `room_id` 的 `/api` 请求由 Gateway 转发到归属节点,API 不再自行转发。
- 不再有 Meta 进程或 Meta RPC;控制面代码和 SQL 都在 API 进程内。

## 目录

保持目录扁平,优先在现有包中增加文件,不为单一职责新建目录。

```text
cmd/api          API 入口(控制面 + 消息面)
cmd/gateway      Gateway 入口
internal/api     HTTP Handler、Kitex Server 和 API 业务逻辑
internal/gateway WebSocket、Kitex Client、会话和批量发送
internal/store   Pebble 消息、PostgreSQL 控制面、迁移和备份
pkg              配置、分片、JWT、日志和通用工具
proto            Kitex/fastpb 协议源文件与生成代码(proto/pb)
```

禁止重新引入 `internal/model`、`internal/domain`、`internal/repository`、`internal/usecase` 等层级。协议结构、存储记录和业务输入在可以复用时直接复用;只有在边界确实不同且转换有明确价值时才定义新结构。

## 代码风格

- 代码优先简洁、直白、可读;小函数只做一件事。
- 错误必须带上下文,禁止静默忽略;重复错误使用包级 sentinel 或 `errors.Is`。
- 所有 goroutine、channel、Pebble iterator 和 stream 都必须有明确的生命周期和关闭路径。
- 所有队列同时限制数量和字节数,禁止无界缓存和无限创建 goroutine。
- 不在 Handler、Gateway 或 Store 中复制相同的业务规则。
- Key 编码集中维护,不在业务代码中手写字符串前缀。
- Pebble 读结果必须及时释放;Batch、Iterator、Snapshot 使用 `defer Close`。
- 任何写路径都要说明幂等键、顺序保证和 ACK 时机。
- 先写可验证的最小实现,再做性能优化;关键路径必须同时有 test 和 benchmark,`make verify` 通过才算可交付。

## 协议和存储

- fastpb/Kitex 编解码只使用 `proto3`,不要使用 protobuf edition、`Any` 或未验证的 known types。
- UUID 统一使用 Go 标准库 `uuid`,不要引入第三方 UUID 包。
- 消息使用 `room_seq` 做历史分页和断线补拉,不使用时间戳作为唯一游标。
- 消息幂等键为 `room_id + sender_id + client_msg_id`,相同键不同内容必须返回冲突。
- 不需要考虑任何的兼容性行为,一切以代码最优雅简洁,性能为目标

## 存储架构 (PostgreSQL -> 内存 -> Pebble -> Parquet)

### 1. 控制面 PostgreSQL
- 用户、设备 token、好友关系、好友申请、黑名单、房间、成员与成员设置、已读游标全部存 PostgreSQL。
- 所有 API 节点共享同一 PostgreSQL;控制面请求不按房间分片,可以落到任意 API 节点。
- 迁移文件位于 `internal/store/migrations`,API 启动时使用 advisory lock 串行执行。
- 会话列表用一条 `room_members LEFT JOIN room_reads` 查询完成,只返回房间、成员设置和 `read_seq`,禁止逐房间 N+1。
- 高频查询必须有对应索引:用户搜索使用 `pg_trgm` GIN,已读回执使用 `(room_id, read_seq, user_id)`,好友/申请/黑名单按时间排序列建索引。

### 2. 消息面内存层
- **高性能分片 LRU 缓存 (`phuslu/lru`)**:
  - `messageCache` (`roomTail`): 4096 房间的分片 LRU 环形缓冲区,每个活跃房间保留最新 128 条(至多 64KB)已提交消息。最新一页(`before=0`)历史读取及推拉结合增量拉取(`MessagesAfter`) 100% 命中内存
  - `roomShards`: 64 分片 LRU 缓存热点房间消息头(`LastSeq`/`LastMsg`)
  - `dedupCache`: 缓存近期 32K 条消息的幂等键(`room_id + sender_id + client_msg_id`),秒级拦截网络重发,消除写前读取 Pebble 的 LSM 读放大。
  - `StreamHandler.memberCache`: API 广播网关使用 `phuslu/lru.TTLCache` 缓存 4096 房间活跃成员,30s TTL,杜绝长连接无界 map 内存泄漏。
- **推拉结合与网关单包就地扇出 (RoomPush)**:
  - API 向 Gateway 发送 `RoomPushBatch` 单包广播,消息 Payload 仅在网络间传输 1 次,由 Gateway 在本地连接池就地扇出给所有目标 WebSocket 连接
  - 客户端支持推拉结合:支持推送完整消息与轻量同步信号(`SyncNotice`),断线或检测到序号空洞时通过 `MessagesAfter` 批量拉取,极速内存闭环。
- **写聚合缓冲管道**:客户端发送请求通过 `messageQueue`(4096 容量)由单写协程在 1ms 或 128 条批大小窗口内合并,变随机小写为顺序批写。

### 3. 消息面 Pebble
- 每个 API 节点只负责本地逻辑槽的消息数据,不依赖 PostgreSQL 存消息正文。
- 大端字典序 Key:消息 `m:<room_id>:<room_seq>`、消息索引 `i:<room_id>:<msg_id>`、幂等 `d:<room_id>:<sender_id>:<client_msg_id>`、置顶、表情、归档清单。
- 消息记录、room_seq、最新消息预览和幂等凭证在同一 Batch 提交;Pebble 同步提交成功后才返回 ACK。
- PostgreSQL 只保存控制面与已读游标;最新消息、预览和未读数由客户端维护,消息 ACK 和历史读取只依赖 Pebble,客户端通过 room_seq 补拉收敛。

### 4. 冷数据归档层
- **异步批归档**:后台 Archiver 协程周期巡检沉淀的冷历史消息(超过 `ArchiveThreshold`),按房间与序号区间分段。
- **Parquet 列式高压缩 (`parquet-go`)**:
  - 高基数 ID(`MsgID`、`SenderID` 等)采用字典编码(Dictionary Encoding);
  - 单调递增序号与时间戳(`RoomSeq`、`ServerTime`)采用 Delta 增量压缩;
  - 变长二进制内容(`Payload`、`Ext`)采用 ZSTD 块级压缩;
  - 体积较原始行存压缩 60%~80%,大幅降低长期存储成本。
- **透明分层读取 (`remoteReader`)**:当客户端请求早于 Pebble 本地保留范围的历史消息时,存储层自动向对象存储拉取对应 Parquet 数据块解码并拼接,上层 API 与前端无感知。

## 验证

提交前至少运行`lint`,确保无报错

```bash
golangci-lint fmt
golangci-lint run
```

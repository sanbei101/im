# IM 项目开发规范

## 架构

本项目使用 Go、Pebble 和按 `room_id` 的固定逻辑槽分片,不依赖 PostgreSQL、Redis、外部消息队列、Raft 或服务注册中心。

- `gateway` 负责 WebSocket 长连接、鉴权、会话管理、按 `room_id` 路由和批量发送。
- `api` 负责 HTTP API、房间分片写入、用户/好友元数据和 Pebble 读写。
- Gateway 与 API 之间使用 Kitex 双向流式 gRPC,底层使用 Netpoll,消息使用 Kitex 生成的 fastpb 编解码。
- 每个 API 节点只拥有配置中分配的逻辑槽;`hash(room_id) % slot_count` 得到槽,再通过静态拓扑找到节点。
- 消息、房间、成员、消息幂等记录必须写入同一个 Pebble Batch;只有本地持久化成功后才能返回成功 ACK。
- 实时推送不是可靠存储,断线和推送失败统一通过房间消息序号补拉。
- Pebble checkpoint 由 API 进程定期生成和保留;当前不实现 `imctl`。
- 多 API 节点时,`api.http_addrs`、`gateway.api_addrs`、`gateway.api_http_addrs` 都按 `node_index` 排列;客户端只连接 Gateway,`/ws` 走长连接,带 `room_id` 的 `/api` 请求由 Gateway 转发到归属节点。

## 目录

保持目录扁平,优先在现有包中增加文件,不为单一职责新建目录。

```text
cmd/api          API 入口
cmd/gateway      Gateway 入口
internal/api     HTTP Handler、Kitex Server 和 API 业务逻辑
internal/gateway WebSocket、Kitex Client、会话和批量发送
internal/store   Pebble、Key 编码、用户、房间、消息和备份
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
- 先写可验证的最小实现,再做性能优化;优化必须有 benchmark 或指标依据。

## 协议和存储

- fastpb/Kitex 编解码只使用 `proto3`,不要使用 protobuf edition、`Any` 或未验证的 known types。
- UUID 统一使用 Go 标准库 `uuid`,不要引入第三方 UUID 包。
- 消息使用 `room_seq` 做历史分页和断线补拉,不使用时间戳作为唯一游标。
- 消息幂等键为 `room_id + sender_id + client_msg_id`,相同键不同内容必须返回冲突。
- 不需要考虑任何的兼容性行为,一切以代码最优雅简洁,性能为目标

## 存储架构 (内存 -> 文件 -> Parquet 归档)

本项目采用严格的三层渐进式分层存储架构,兼顾极端读写性能、严格持久化一致性与海量冷数据低成本保留:

### 1. 内存层
- **高性能分片 LRU 缓存 (`phuslu/lru`)**:
  - `messageCache` (`roomTail`): 4096 房间的分片 LRU 环形缓冲区,每个活跃房间保留最新 128 条(至多 64KB)已提交消息。最新一页(`before=0`)历史读取及推拉结合增量拉取(`MessagesAfter`) 100% 命中内存
  - `roomShards`: 64 分片 LRU 缓存热点房间元数据与成员名单(`members`),反范式化嵌入 `LastMsg` 与 `LastSeq`
  - `userShards` & `usernameCache`: 64 分片 LRU 缓存热点用户资料与用户名反查,批量用户解析(`UsersByIDs`)优先走内存
  - `dedupCache`: 缓存近期 32K 条消息的幂等键(`room_id + sender_id + client_msg_id`),秒级拦截网络重发,消除写前读取 Pebble 的 LSM 读放大。
  - `StreamHandler.memberCache`: API 广播网关使用 `phuslu/lru.TTLCache` 缓存 4096 房间活跃成员,30s TTL,杜绝长连接无界 map 内存泄漏。
- **推拉结合与网关单包就地扇出 (RoomPush)**:
  - API 向 Gateway 发送 `RoomPushBatch` 单包广播,消息 Payload 仅在网络间传输 1 次,由 Gateway 在本地连接池就地扇出给所有目标 WebSocket 连接
  - 客户端支持推拉结合:推送完整消息,断线或检测到序号空洞时通过 `MessagesAfter` 批量拉取,极速内存闭环。
- **写聚合缓冲管道**:客户端发送请求通过 `messageQueue`(4096 容量)由单写协程在 1ms 或 128 条批大小窗口内合并,变随机小写为顺序批写。

### 2. 文件存储层
- **无中心化独立存储**:单 API 节点全权负责本地逻辑槽的 Pebble 实例,不依赖任何外部集中式数据库。
- **大端字典序 Key 编码**:
  - 消息主表 `m:<room_id>:<room_seq>`:单调自增前缀连续存储,消息游标分页与断线补拉天然有序高效。
  - 消息索引 `i:<room_id>:<msg_id>` -> `room_seq`:支持点查与快速定位。
  - 幂等记账 `d:<room_id>:<sender_id>:<client_msg_id>` -> `Dedup`(记录 SHA-256 `PayloadSum` 防止冲突)。
  - 房间元数据 `r:<room_id>`、成员关系 `M:<room_id>:<user_id>`、已读游标 `R:<user_id>:<room_id>`。
- **单 Batch 严格原子性**:消息记录、房间 `LastSeq`、最新消息预览与幂等凭证必须提交至同一个 `pebble.Batch`;持久化至本地 WAL/SST 后才向客户端返回 ACK。
- **Checkpoint 物理快照**:由 API 节点定期生成硬链接快照,零停机且无锁备份。

### 3. 冷数据归档层
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

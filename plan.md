# IM 成品实施方案

> 目标：不考虑旧客户端和旧接口兼容，直接将当前项目改造成个人项目可用的 IM MVP。
>
> 原则：先保证核心链路正确，再补业务能力；不为低概率、低收益问题引入高复杂度；所有明确错误都必须处理或记录，禁止静默忽略 `err`。

## 一、架构边界

继续保留三个模块：

- **API**：连接 PostgreSQL 和 Redis，负责认证、用户、好友、房间、历史消息、离线同步、上传等 REST 请求。
- **Gateway**：只连接 Redis 和 WebSocket，负责连接认证、协议编解码、基础校验、消息入队和实时下发，禁止连接 PostgreSQL。
- **Worker**：连接 PostgreSQL 和 Redis，负责消息权限校验、幂等落库、未读处理、投递任务和失败重试。

消息链路保持：

```text
WebSocket -> Gateway -> Redis inbound -> Worker -> PostgreSQL
Worker -> Redis delivery -> Gateway -> WebSocket
REST -> API -> PostgreSQL/Redis
```

不新增微服务，不引入复杂事件溯源、分布式事务或独立消息序列服务。

## 二、统一协议

### 1. 使用 client_msg_id
客户端每次请求生成 `client_msg_id`，用于请求关联、重试幂等、日志追踪和 WebSocket 去重。

统一字段含义：

```text
client_msg_id    客户端请求/消息 ID
msg_id           服务端生成的消息 ID
server_time      服务端消息时间和同步游标
```

### 2. 统一 WebSocket Frame

修改 Protobuf，不再直接发送裸 ACK 或裸 Message：

- `ClientFrame`：发送消息、已读、正在输入、Ping；
- `ServerFrame`：ready、ACK、消息、落库成功、失败、撤回、已读、输入状态、Pong、错误。

用 `oneof` 区分帧类型，响应中带对应 `client_msg_id`，消息事件带 `msg_id` 和 `server_time`。

### 3. 使用 server_time

不增加 `message_seq`。历史和离线同步使用 `server_time`：

- 房间历史：查询 `server_time < before_server_time`；
- 离线同步：查询 `server_time > after_server_time`；
- 结果按 server_time 排序；
- 如需处理同时间消息，用 `msg_id` 作为排序辅助，不设计独立序列系统。

## 三、第一阶段：修复主链路

### 1. 认证和错误处理

- JWT secret 从配置/环境变量读取，删除硬编码密钥。
- 实现 refresh token、logout 和基本会话吊销。
- 收紧 CORS、WebSocket Origin，并限制请求体、帧和 payload 大小。
- 生产环境不直接返回 `err.Error()`，用稳定错误码，详细错误写日志。
- 数据库、Redis、MQ、JSON、Protobuf、事务提交、关闭连接等错误必须处理；清理错误至少记录日志。

### 2. 权限校验

权限放在 API Service 和 Worker，不放在 Gateway：

- API 查询房间、成员、历史消息必须校验当前用户身份和成员关系；
- Worker 落库前校验 sender、room、成员、退群、禁言、拉黑等状态；
- 客户端传入的 sender_id、角色、owner_id 一律不可信；
- Gateway 只做身份解析、协议校验和 Redis 通信。

Gateway 不能连接数据库。第一版允许 Worker 直接查询 PostgreSQL，不为了缓存一致性增加复杂方案。

### 3. 消息幂等和 Worker 可靠性

数据库增加：

```sql
UNIQUE (sender_id, client_msg_id)
INDEX messages(room_id, server_time DESC, msg_id DESC)
```

Worker 处理流程：

1. 校验 UUID、房间、类型和 payload；
2. 校验发送权限；
3. 按 `(sender_id, client_msg_id)` 查询或插入；
4. 重试命中已有消息时返回原 `msg_id/server_time`；
5. 新消息落库后生成 delivery task；
6. delivery task 成功写入后 ACK inbound。

增加最小可靠机制：处理 Pending、有限重试、dead-letter stream。不要实现 exactly-once 或跨 Redis/PostgreSQL 两阶段提交，使用“数据库幂等 + MQ 至少一次投递”。

## 四、第二阶段：补核心业务

### 1. 用户和账号

补齐注册、登录、刷新、登出、当前用户资料、修改密码、用户搜索、公开用户信息和设备会话。

所有用户身份从 JWT 获取，不能接受请求体中的当前用户 ID。密码只保存 bcrypt hash，用户响应禁止返回密码。

### 2. 好友和黑名单

补齐好友申请、接受、拒绝、删除好友、好友列表、拉黑、取消拉黑和黑名单。

关系变化使用事务，处理自己加自己、重复申请、无效用户和黑名单。单聊默认只允许好友之间创建；如果产品决定允许陌生人，只调整策略规则。

### 3. 房间和群组

将现有房间接口整理为房间列表、单聊创建/获取、群创建、房间详情、成员列表、加人、踢人、退群、解散、转让群主、角色调整和个人房间设置。

要求：

- 单聊使用双方用户的唯一 hash；
- 群创建者强制成为 owner；
- 创建房间、添加成员、生成系统事件使用同一事务；
- 统一 owner/admin/member 权限；
- 成员变化后清理 Redis 成员缓存；
- 房间列表包含最后消息、未读数、置顶和免打扰状态。

## 五、第三阶段：消息查询和状态

### 1. 历史和离线同步

将历史接口改为按房间查询，并新增全局离线同步接口。客户端保存最后处理的 `server_time`，重连后继续拉取。

所有消息查询都必须验证成员权限。API 返回稳定 DTO，不直接暴露 sqlc 数据库行。第一版不设计统一事件日志，使用历史消息、房间详情和必要的重新查询恢复状态。

### 2. 已读、未读和撤回

在 `room_members` 增加 `last_read_server_time`：

- 已读位置只能向前移动；
- 房间列表先通过索引统计未读，不提前引入复杂计数器；
- 撤回只更新状态，不物理删除；
- 第一版只允许发送者在固定时间窗口内撤回自己的消息；
- 已读和撤回同时支持 REST 和 WebSocket。

## 六、第四阶段：实时、多设备和 Gateway 路由

Redis 保存短期 Presence：

```text
presence:{user_id}:{session_id} = gateway_id
```

用 TTL 和心跳刷新。Worker 根据目标用户在线的 gateway_id 聚合投递任务，写入：

```text
message:deliver:{gateway_id}
```

Gateway 只消费自己的投递流。一个用户允许多个设备和 session，消息推送到全部在线 session；离线用户通过 server_time 同步。

连接管理需要 ready、ping/pong、读写超时、连接数限制和发送队列保护。对于无法复现、发生概率极低且修复需要复杂锁层的竞态，不增加高成本实现，只保留日志和基础保护。

## 七、第五阶段：上传和文件消息

增加上传任务和对象存储流程：创建上传、完成确认、查询状态、取消上传。

文件直接上传对象存储，不经过 Gateway、Redis 或 Worker。消息只能引用当前用户已完成的 upload_id，并校验大小、MIME、扩展名、hash 和过期时间。

第一版先支持图片和普通文件，视频转码、缩略图、离线推送放到后续阶段。

## 八、数据库和代码规范

不要手动修改 `internal/db` 生成代码。只修改：

```text
sqlc/migrations/
sqlc/query/
proto/im/v1/*.proto
```

然后执行：

```text
sqlc generate
buf generate
gofmt -w .
go test ./...
```

核心表保留并补强：

```text
users
rooms
room_members
messages
```

新增：

```text
friend_requests
friendships
blocks
refresh_sessions
uploads
```

必须增加必要的外键、唯一约束和索引。迁移改为编号版本，不再只依赖 Docker 首次初始化脚本。每次数据库或 Protobuf 变更都必须重新生成代码并运行测试。

Go 代码要求：

- 不忽略数据库、Redis、MQ、JSON、Protobuf、事务提交和关闭连接错误；
- 错误使用 `%w` 包装，handler 使用 `errors.Is/As` 映射业务错误；
- 日志包含 `stream_id/client_msg_id/msg_id` 等定位信息；
- 不用 panic 处理普通请求错误；
- 不为了单一实现新增无用接口、工厂或通用 CRUD 框架。

## 九、实施顺序

```text
1. 修改 Protobuf：client_msg_id 和统一 Frame
2. 修复 JWT、错误响应、配置和请求限制
3. 修复 API/Worker 房间权限
4. 增加 client_msg_id 幂等和消息落库回执
5. 增加 Worker Pending、重试和 dead-letter
6. 完成用户、好友和黑名单
7. 完成房间、群成员和房间设置
8. 完成历史、server_time 同步、已读和撤回
9. 完成 Presence、多设备和 Gateway 专属投递流
10. 完成上传、健康检查和集成测试
```

每个阶段提交代码时同时提交 SQL 迁移、sqlc/Protobuf 生成文件、测试和接口示例。

## 十、MVP 完成标准

- 用户可以注册、登录、刷新、登出和修改资料；
- 用户可以建立好友关系和拉黑用户；
- 用户可以创建单聊、群聊并管理成员；
- 非成员不能读取或发送房间消息；
- WebSocket 能区分 ACK、消息、错误和心跳；
- 相同 client_msg_id 重试不会生成重复消息；
- 消息能落库、实时投递和离线同步；
- 支持已读、未读和消息撤回；
- Gateway 全程不连接数据库；
- 关键错误均有处理和日志；
- PostgreSQL、Redis、Worker、Gateway 有基础集成测试；
- Docker healthcheck 和部署配置正常。

第一版不做音视频、端到端加密、多租户、跨地域多活、复杂全文搜索、机器人平台、严格 exactly-once，以及没有实际复现且修复成本明显高于收益的低概率竞态问题。

# Go IM System 架构与开发指南 (AI 开发指令)

## 🤖 给 AI 的指令说明
你现在是一个资深的 Go 语言后端架构师。本项目的目标是使用 Go 语言从零开发一个高性能、分布式的即时通讯(IM)系统。

## 🏗️ 架构设计
本项目采用拆分架构,共包含三个核心独立模块,统一放在一个代码仓库中:(见cmd/目录)

1.  **消息网关模块 (Gateway - `cmd/gateway`)**
    * **技术选型**: 使用redis-streams作为消息队列,实现模块间的异步通信
    * **职责**:负责维护海量客户端的长连接。
    * **功能**:接收客户端发送的消息,进行协议解析和鉴权;将接收到的消息投递到消息队列(MQ);监听 MQ 的推送指令,将消息精准下发到对应的在线用户连接。
    * **核心组件**:Connection Manager (连接管理器)

2.  **消息处理模块 (Worker - `cmd/worker`)**
    * **职责**:异步处理繁重的消息逻辑,保护数据库。
    * **功能**:作为消费者从 MQ 中拉取消息;处理消息的持久化(落库PostgreSQL);生成消息 ID;更新未读数;将处理后的"投递指令"重新放回 MQ,交由 Gateway 进行推送。

3.  **常规 API 模块 (API - `cmd/api`)**
    * **职责**:处理所有非实时的短连接请求。
    * **功能**:基于 Gin 框架开发。负责用户注册登录、好友关系管理、群组管理、历史消息拉取、离线消息同步等 RESTful 接口。

4. **数据库层 (db/)**:使用 sqlc 生成类型安全的数据库访问代码,负责与 PostgreSQL 的交互,请勿修改sqlc 生成的代码(`internal/db`目录,一定是修改`db`目录下的sql文件,再使用`sqlc generate`命令生成代码)

## 🛠️ 技术栈选型

* **语言**:Go 1.26
* **Web 框架**:Gin (仅用于 API 模块)
* **长连接**:`coder/websocket`
* **数据库**:PostgreSQL + sqlc
* **缓存**:Redis
* **消息队列**:Redis Streams
* **配置管理**:直接读取config.yaml
* **日志记录**:`https://github.com/phuslu/log`(最快的日志库)(`pkg/logger/logger.go`)

## 📂 项目结构规范

请严格遵循标准的 Go Project Layout,所有公用工具放入 `pkg/`,业务代码按模块分装在 `internal/` 下,程序入口位于 `cmd/{module}/main.go`

## 代码逻辑
1.前端发送消息(JSON 格式)
```json
{
  "message": {
    "msg_id": "后端生成的uuidv7",
    "client_msg_id": "前端生成的uuidv7" // 保证幂等性,
    "sender_id": "user_1001",
    "room_id": "user_1002",
    "chat_type": "CHAT_TYPE_SINGLE",
    "server_time": 1743020000000000000,
    "text": {
      "content": "你好",
    },
  }
}
```

2. 网关接收消息
职责:接收 JSON -> 补全核心参数(msg_id,server_time) -> 丢进 Redis Stream

3. 工作线程处理消息
职责:从 Redis Stream 拉取消息 -> 持久化到 PostgreSQL -> 生成投递指令 -> 丢回 Redis Stream


4. 目标网关推送给接收方
职责:监听 Redis Stream 的投递指令 -> 定位接收方在线连接 -> 推送消息(先不考虑离线)
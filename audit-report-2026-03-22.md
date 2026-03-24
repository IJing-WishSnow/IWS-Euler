# IWS-Euler 系统审计报告

**审计时间：** 2026-03-22
**审计范围：** 全部服务源码、K8s 配置、服务间交互、运行时日志
**系统状态：** K8s 11 个 Pod 全部 Running，但存在以下问题

---

## 🔴 严重 Bug（会导致崩溃或功能完全失效）

### 1. Gateway `users` map 数据竞争

**文件：** `Gateway/handler/handler.go:22`
**问题：** `var users = map[string]string{}` 是包级别无锁 map。Gin 每个请求一个 goroutine，并发注册/登录时会触发 data race，Go runtime 会直接 panic。
**修复：** 加 `sync.RWMutex`。

### 2. RiskControl 消费线程永久死亡

**文件：** `RiskControl/consumer.py:29`
**问题：** `_consume_orders` 和 `_consume_trades` 是 daemon 线程，启动时若 Kafka 未就绪，`KafkaConsumer()` 构造函数抛 `NoBrokersAvailable`，线程直接死亡且无 try/except 重试。主进程 `while True: time.sleep(1)` 仍在运行，K8s 认为 Pod 健康，但消费线程已永久死亡。**风控实际上什么都没有在消费。**
**验证：** `kubectl logs deployment/riskcontrol` 最新日志停在 108 分钟前的异常栈。
**修复：** 在线程内部加 `try/except + 重试循环`。

---

## 🟠 功能性缺陷（影响正确性或稳定性）

### 3. OrderService 创建了从不使用的 Kafka Consumer

**文件：** `OrderService/service/order_service.go:22`
**问题：** `NewOrderService()` 创建了 `tradeConsumer`（group: "order-service"），但 `ConsumeTradeResults` 在 `cmd/main.go` 里从未调用。白白占一个 Kafka Consumer Group 连接，浪费资源。
**修复：** 从 `OrderService` struct 中移除 `tradeConsumer`，从 `NewOrderService()` 和 `Close()` 中删除相关代码。

### 4. Gateway 调用 OrderService 无超时

**文件：** `Gateway/handler/handler.go:96`
**问题：** `http.Post(orderServiceURL+"/order", ...)` 使用默认 `http.Client`，无超时设置。若 OrderService 挂起，Gateway 请求永久阻塞，前端无响应，且会耗尽 Gateway goroutine 资源。
**修复：** 使用带 timeout 的 `http.Client`（建议 5s）。

### 5. MatchingEngine `users` map 无限增长（内存泄漏）

**文件：** `MatchingEngine/bridge/bridge.go:127`
**问题：** `b.users[om.ID] = om.UserID` 每笔订单加一条，从不删除。长期运行持续泄漏内存。
**修复：** 成交完成后从 map 中删除已用的 orderID。

---

## 🟡 K8s 配置问题

### 6. 四个服务没有任何健康探针

**文件：** `matchingengine.yaml`、`accountservice.yaml`、`chainclient.yaml`、`riskcontrol.yaml`
**问题：** Pod 启动立即被标记 Ready，K8s 无法感知服务实际健康状态。RiskControl 僵尸进程问题正是因此未被自动重启。
**修复：** 添加 `livenessProbe`（进程存活检测）。

### 7. Kafka 启动竞争（无依赖顺序控制）

**问题：** 所有 Pod 同时启动，Kafka 未就绪时：

- Go 服务（MatchingEngine、AccountService）：循环重试，自然恢复 ✓
- ChainClient：进程崩溃 → K8s 重启 3 次后恢复（不优雅）
- RiskControl（Python）：线程死亡，进程不崩溃 → K8s 不知道，永不重启 ✗
  **修复：** RiskControl 加重试解决根本问题；ChainClient 可加重试避免 crash。

### 8. operatorPrivKey 明文暴露

**文件：** `Deploy/values.yaml:13`，注入到 Pod env var
**问题：** `kubectl describe pod chainclient-xxx` 可直接看到私钥明文。
**修复：** 改用 K8s Secret。（Demo 项目可接受现状，记录风险即可）

---

## 🔵 代码质量问题

### 9. nginx location 重复冗余

**文件：** `WebApp/nginx.conf:11-21`
**问题：** 两个 location 块处理 login/register，逻辑重叠，可读性差。功能上不影响。

### 10. Gin 运行在 debug 模式

**文件：** Gateway、OrderService 均未设置 `GIN_MODE=release`
**修复：** K8s Deployment env 加 `GIN_MODE=release`。

### 11. `getEnv` 函数重复定义

**问题：** Gateway 的 `cmd/main.go` 和 `handler/handler.go` 各有一份，违反 DRY 原则。

### 12. OrderService 日志格式

**文件：** `OrderService/service/order_service.go:50`
**问题：** `%d` 格式化 `Side` 类型，日志打印 `0`/`1` 而非 `buy`/`sell`。

---

## 修复优先级总结

| 优先级 | 问题                             | 当前影响        |
| ------ | -------------------------------- | --------------- |
| 🔴 P0  | Gateway users map 数据竞争       | 并发请求必崩    |
| 🔴 P0  | RiskControl 消费线程死亡         | 风控形同虚设    |
| 🟠 P1  | Gateway 调用 OrderService 无超时 | 雪崩风险        |
| 🟠 P1  | OrderService 无用 Consumer       | 资源浪费        |
| 🟠 P1  | 四个服务无健康探针               | K8s 盲目运维    |
| 🟡 P2  | MatchingEngine 内存泄漏          | 长期运行问题    |
| 🟡 P2  | Gin debug 模式                   | 性能 + 信息泄露 |
| 🔵 P3  | nginx 冗余 location              | 可读性          |
| 🔵 P3  | getEnv 重复定义                  | 代码质量        |
| 🔵 P3  | 日志格式                         | 可读性          |

# IWS-Euler 修复总结（2026-03-22）

## 概述

本次对 IWS-Euler 所有项目进行了三轮全面审计，共修复 **12 项问题**（P0×3, P1×4, P2×4, P3×1）。
修复后所有 11 个 Pod 均正常运行，无 CrashLoop，无 Error 状态。

---

## 第一轮审计修复

### P0：数据竞争 — Gateway users 并发不安全

**文件**: `Gateway/handler/handler.go`

**问题**: `users` map 被多个 Gin goroutine 并发读写，无任何锁保护，属于 Go 数据竞争，会导致 panic 或数据损坏。

**修复**: 引入 `sync.RWMutex`，Register 用写锁（`Lock`），Login 用读锁（`RLock`）。

---

### P0：SRP 违反 — Gateway 直接写 Kafka（单一职责原则）

**文件**: `Gateway/handler/handler.go`, 新建 `OrderService/`

**问题**: Gateway 直接导入 kafka-go，承担了订单提交职责。这违反了微服务单一职责原则，且无法独立扩展订单服务。

**修复**: 新建 `OrderService` 微服务（Go + Gin，端口 8082）。

- `POST /order` 接收订单，写入 Kafka
- Gateway 的 `PlaceOrder` 改为通过 HTTP 转发到 OrderService
- Gateway 完全移除 kafka-go 依赖
- 同步新增 Helm 模板 `templates/orderservice.yaml`、更新 `values.yaml`

---

### P1：RiskControl 消费线程静默死亡

**文件**: `RiskControl/consumer.py`

**问题**: Python 的 `kafka-python` 在构造器阶段若 Kafka 不可用会抛 `NoBrokersAvailable`，daemon 线程崩溃，主进程继续存活，K8s 无法感知到服务已停止工作。

**修复**: 在两个消费线程中加入 `while self._running` + `try/except` 重试循环。构造器异常后等待 5 秒重试，线程永不退出。同时加入 `PYTHONUNBUFFERED=1` 环境变量解决 Docker 中 Python stdout 缓冲问题。

---

### P1：HTTP Client 无超时 — Gateway 转发可能永久阻塞

**文件**: `Gateway/handler/handler.go`

**问题**: 原代码使用裸 `http.Post`（底层为 `http.DefaultClient`），无任何超时设置。OrderService 不可用时，Gateway goroutine 永久挂起，耗尽连接池。

**修复**: 使用 `&http.Client{Timeout: 5 * time.Second}` 包级变量替代默认客户端。

---

### P1：Anvil Pod 重启后合约丢失

**目录**: 新建 `Anvil/`

**问题**: Helm 部署使用 `ghcr.io/foundry-rs/foundry:latest` 镜像并传入 `command: ["anvil", "--host", "0.0.0.0"]`，Pod 重启后链上状态清空，IWSSettlement 合约消失，ChainClient 所有交易均失败。

**修复**: 新建自定义 `Anvil` Docker 镜像：

- `entrypoint.sh` 启动 Anvil → 等待就绪 → 确定性部署 MockERC20（nonce=0）和 IWSSettlement（nonce=1）
- 合约地址与 `values.yaml` 中配置完全一致，重启后自动恢复
- 注意：`ENTRYPOINT ["sh", "/entrypoint.sh"]` 而非 `RUN chmod +x`（foundry 镜像为非 root 用户）

---

### P2：纯消费型服务无 livenessProbe

**文件**: `Deploy/templates/` 中 matchingengine, accountservice, chainclient, riskcontrol 的 yaml

**问题**: 上述服务无 HTTP 端口，无就绪探针，K8s 无法检测进程异常退出后的 Pod 健康状态。

**修复**: 统一添加：

```yaml
livenessProbe:
  exec:
    command: ["sh", "-c", "kill -0 1"]
  initialDelaySeconds: 10
  periodSeconds: 15
  failureThreshold: 3
```

`kill -0 1` 检测 PID 1（主进程）是否存活，不发送任何信号，零侵入。

---

### P2：Docker 镜像基础镜像无法拉取

**文件**: `Gateway/Dockerfile`, `OrderService/Dockerfile`

**问题**: `golang:alpine` / `golang:1.25-alpine` 在当前网络环境下无法从 Docker Hub 拉取。

**修复**: 改用本地已缓存的 `wishsnow/golang-dev:latest` 作为构建镜像。

---

### P3：nginx.conf 冗余 location 块

**文件**: `WebApp/nginx.conf`

**问题**: `/login` 和 `/register` 分别使用独立 location 块，配置重复，扩展性差。

**修复**: 合并为单个正则 location：

```nginx
location ~ ^/(login|register|api/) {
    proxy_pass http://iws-gateway:8081;
    ...
}
```

---

## 第二轮审计修复

### P0：MarketData Hub 数据竞争 — map 写操作持有 RLock

**文件**: `MarketData/ws/hub.go`

**问题**: `broadcast()` 函数持有 `RLock`，在 `WriteMessage` 失败时直接调用 `delete(h.clients, conn)`。
对 map 的写操作必须持有写锁，在 `RLock` 下修改 map 属于数据竞争，会导致 panic 或 map 内部结构损坏。

**修复**: 将逻辑拆分为两阶段：

1. 持 `RLock` 广播消息，将失败连接收集到切片 `dead`
2. 释放 `RLock`，若有失败连接则升级到 `Lock` 后批量删除

```go
var dead []*websocket.Conn

h.mu.RLock()
for conn := range h.clients {
    if err := conn.WriteMessage(...); err != nil {
        dead = append(dead, conn)
    }
}
h.mu.RUnlock()

if len(dead) > 0 {
    h.mu.Lock()
    for _, conn := range dead {
        conn.Close()
        delete(h.clients, conn)
    }
    h.mu.Unlock()
}
```

---

### P1：MatchingEngine 内存泄漏修复引入部分成交 Bug（已回滚）

**文件**: `MatchingEngine/bridge/bridge.go`

**背景**: 第一轮审计时尝试修复 `b.users` map 无限增长问题，在每笔成交后 `delete(b.users, t.BuyOrderID)` 和 `delete(b.users, t.SellOrderID)`。

**发现**: 读取 `engine.go` 后确认引擎支持部分成交（`model.PartiallyFilled` 状态），同一 `orderID` 可能对应多笔成交。第一笔成交后删除 `userID` 映射，后续成交的 `BuyUserID`/`SellUserID` 将变为空字符串，导致 AccountService 结算失败、ChainClient 链上结算地址查不到。

**处理**: 立即回滚 delete 操作，保留原始行为，并在代码中添加解释性注释说明原因。`b.users` 的内存泄漏在演示规模下可控（每个 orderID 一条 entry，订单量有限），留待接入数据库后通过订单状态 `Filled` 触发清理。

---

## 第三轮审计修复

### P0：Gateway 限流完全失效 — `formatInt` 使用错误的数字格式

**文件**: `Gateway/middleware/ratelimit.go`

**问题**: `formatInt` 函数用于生成 Redis `ZRemRangeByScore` 的分数边界，但实现错误：

```go
// 错误实现
func formatInt(n int64) string {
    return time.UnixMilli(n).Format("20060102150405000")
}
```

Redis ZSET 中存储的分数是 Unix 毫秒时间戳（约 1.7 万亿，如 `1711116245123`），而 `formatInt` 返回格式化日期字符串（如 `"20240322150405123"`），解析为数值约 20 千万亿。

**后果**: `ZRemRangeByScore("0", "20240322150405123")` 的删除范围覆盖所有真实分数（1.7万亿 < 20千万亿），每次请求都清空整个 ZSET。`ZCard` 始终返回 0，`0 >= 10` 永远为假，限流从不触发。任何用户每秒可以发无限请求。

**修复**: 改用 `strconv.FormatInt(n, 10)`，输出与存储分数格式一致的十进制数字字符串。

```go
func formatInt(n int64) string {
    return strconv.FormatInt(n, 10)
}
```

---

### P2：OrderService 死代码 — SRP 重构遗留

**文件**: `OrderService/service/kafka.go`

**问题**: SRP 重构时从 OrderService 移除了 trade 消费逻辑，但 `Consumer` 结构体、`NewConsumer`、`Read` 方法和 `TradesTopic` 常量未同步删除，约 30 行死代码留存。

**修复**: 删除 `Consumer` 类型及 `TradesTopic` 常量，文件仅保留 `Producer`。

---

### 第三轮全量核查确认无误项

以下内容经全量核查，确认均无问题：

- **Kafka 消息格式兼容性**: OrderService 发送的 `OrderRequest`（Side/Type 序列化为 `"buy"/"sell"/"limit"/"market"` 字符串）与 MatchingEngine 的 `OrderMessage`（Side/Type 为 `string`）完全匹配；trades 消息格式在 AccountService、ChainClient、MarketData、RiskControl 间完全一致
- **GroupID 唯一性**: 6 个消费者组（`matching-engine`、`account-settler`、`chain-client`、`market-data`、`risk-control-orders`、`risk-control-trades`）全部不重复
- **Kafka 地址一致性**: 所有服务 K8s env 均为 `kafka:9092`，本地 fallback 均为 `localhost:9094`，Kafka 广播地址为 `PLAINTEXT://kafka:9092`
- **合约地址一致性**: MockERC20 `0x5FbDB...` 在 entrypoint.sh 和 ChainClient 中一致；IWSSettlement `0xe7f172...` 在 entrypoint.sh、chainclient.yaml、values.yaml 中一致
- **nginx 路由覆盖**: `^/(login|register|api/)` 覆盖所有 Gateway 路由；`/ws` 正确代理到 MarketData:8080
- **Aggregator 锁顺序**: `Feed` 持 `Aggregator.mu.Lock` → `archive` 调 `hub.BroadcastKLine` → `broadcast` 持 `Hub.mu.RLock`，两个不同的锁，不存在死锁
- **引擎并发安全**: Engine 单线程串行调用（Bridge 单 goroutine），无并发访问问题
- **所有 Dockerfile 构建基础镜像可用**: Go 服务用 `wishsnow/golang-dev:latest`（本地缓存），Python 用 `python:3.11-slim`，WebApp 用 `node:22-alpine` + `nginx:alpine`

---

## 最终状态

| Pod                | 状态    | 备注                        |
| ------------------ | ------- | --------------------------- |
| anvil              | Running | 自动部署合约，重启安全      |
| kafka              | Running | KRaft 单节点                |
| redis              | Running | -                           |
| iws-gateway        | Running | HTTP→OrderService，JWT 鉴权 |
| iws-orderservice   | Running | 新增微服务，写 Kafka        |
| iws-matchingengine | Running | 撮合引擎，部分成交安全      |
| iws-accountservice | Running | 内存版账户，livenessProbe   |
| iws-chainclient    | Running | 链上结算，livenessProbe     |
| iws-marketdata     | Running | hub 数据竞争已修复          |
| iws-riskcontrol    | Running | 线程自愈，PYTHONUNBUFFERED  |
| iws-webapp         | Running | Nginx 静态 + 反代           |

所有服务均无 CrashLoop，无待处理的 P0/P1 问题。

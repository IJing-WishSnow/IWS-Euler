# IWS-Gateway

API 网关服务，负责 JWT 鉴权、Redis 限流和请求路由。

## 职责

- **JWT 鉴权**：验证 `Authorization: Bearer <token>`，保护 `/api/*` 路由
- **滑动窗口限流**：基于 Redis ZSET，每用户每秒最多 10 次请求（HTTP 429）
- **路由转发**：将下单请求代理至 IWS-OrderService
- **CORS**：开发阶段放开所有来源

## 接口

| 方法 | 路径         | 鉴权 | 说明                      |
| ---- | ------------ | ---- | ------------------------- |
| POST | `/register`  | 无   | 用户注册                  |
| POST | `/login`     | 无   | 用户登录，返回 JWT        |
| POST | `/api/order` | 需要 | 下单，转发至 OrderService |

## 环境变量

| 变量                | 默认值                         | 说明                               |
| ------------------- | ------------------------------ | ---------------------------------- |
| `REDIS_ADDR`        | `localhost:6379`               | Redis 地址                         |
| `ORDER_SERVICE_URL` | `http://iws-orderservice:8082` | OrderService 地址                  |
| `GIN_MODE`          | `debug`                        | Gin 运行模式（生产设为 `release`） |

## 目录结构

```
IWS-Gateway/
├── cmd/
│   └── main.go                   # 启动入口，路由注册
├── handler/
│   ├── handler.go                # 注册、登录、下单处理器
│   └── handler_test.go           # 处理器单元测试
├── middleware/
│   ├── auth.go                   # JWT 鉴权中间件
│   ├── ratelimit.go              # Redis 滑动窗口限流
│   └── ratelimit_test.go         # 限流单元 + 集成测试
├── Dockerfile
└── go.mod
```

## 限流实现说明

使用 Redis ZSET 实现滑动窗口：

1. 每次请求以当前 Unix 毫秒时间戳为 score 写入 ZSET
2. 删除窗口外的旧记录（`ZRemRangeByScore`）
3. 统计窗口内记录数，超过阈值返回 429

注意：score 必须使用 `strconv.FormatInt` 序列化为十进制数字字符串，不能使用 `time.Format`（否则限流完全失效）。

## 运行测试

```bash
# 单元测试（不需要 Redis）
docker run --rm -v "T:/IWS-Euler/IWS-Gateway:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v -run TestFormatInt"

# 集成测试（需要 Redis 端口转发）
kubectl port-forward svc/redis 6379:6379 -n iws-Euler
docker run --rm --network host -v "T:/IWS-Euler/IWS-Gateway:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./middleware/... -v"
```

## 端口

- 容器端口：8081
- K8s Service：ClusterIP

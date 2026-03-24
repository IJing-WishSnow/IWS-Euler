# 开发指南

## 环境准备

### 必需工具

| 工具           | 版本               | 用途                                                                      |
| -------------- | ------------------ | ------------------------------------------------------------------------- |
| Docker Desktop | 最新版（启用 K8s） | 容器运行时 + 本地 K8s                                                     |
| kubectl        | 与 K8s 版本匹配    | K8s 集群管理                                                              |
| Helm           | 3.x                | K8s 包管理器                                                              |
| Go             | 1.22+              | Gateway/OrderService/MatchingEngine/AccountService/MarketData/ChainClient |
| Python         | 3.11+              | RiskControl                                                               |
| Node.js        | 18+                | WebApp / SettlementContract 测试                                          |
| Foundry        | 最新版             | 合约编译、部署、本地链                                                    |

### Go 开发环境

各 Go 服务均有独立的 `go.mod`，模块名为 `github.com/yourname/<ServiceName>`。

```bash
# 示例：进入 Gateway 开发
cd Gateway
go mod tidy
go build ./...
```

---

## 单元测试

### Go 服务（本地 Docker 运行）

使用 `wishsnow/golang-dev:latest` 镜像（包含 Go 工具链），避免本地安装依赖：

```bash
# MatchingEngine 单元测试
docker run --rm -v "T:/IWS-Euler/MatchingEngine:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v"

# AccountService 单元测试
docker run --rm -v "T:/IWS-Euler/AccountService:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v"

# Gateway 单元测试（含 formatInt 验证）
docker run --rm -v "T:/IWS-Euler/Gateway:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./... -v"
```

### Gateway 限流集成测试（需 Redis）

```bash
# 先启动 Redis 端口转发
kubectl port-forward svc/redis 6379:6379 -n iws-Euler &

# 运行集成测试（自动跳过如果 Redis 不可达）
docker run --rm --network host -v "T:/IWS-Euler/Gateway:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./middleware/... -v -run TestRateLimiter"
```

### RiskControl 单元测试（Python）

```bash
# 在 Pod 内运行（避免依赖安装问题）
POD=$(kubectl get pod -n iws-Euler -l app=iws-riskcontrol -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n iws-Euler $POD -- python -m unittest tests.test_rule_engine -v
```

或本地运行（需安装依赖）：

```bash
cd RiskControl
pip install -r requirements.txt
python -m unittest tests.test_rule_engine -v
```

### MatchingEngine Kafka 集成测试

```bash
# 先启动 Kafka 端口转发
kubectl port-forward svc/kafka 9094:9092 -n iws-Euler &

docker run --rm --network host \
  -v "T:/IWS-Euler/MatchingEngine:/workspace" \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./bridge/... -v -run TestBridge"
```

---

## 系统 E2E 测试

### 前提

启动必要的端口转发：

```bash
kubectl port-forward svc/iws-gateway 18083:8081 -n iws-Euler &
kubectl port-forward svc/iws-marketdata 18084:8080 -n iws-Euler &
```

### 运行测试

```bash
# Windows（PowerShell）
$env:PYTHONIOENCODING="utf-8"
python T:/IWS-Euler/tests/e2e_system_test.py -v

# Windows（Git Bash / CMD）
PYTHONIOENCODING=utf-8 python T:/IWS-Euler/tests/e2e_system_test.py -v
```

测试覆盖：

| TC    | 描述                        |
| ----- | --------------------------- |
| TC-01 | 新用户注册成功              |
| TC-02 | 重复注册返回 409            |
| TC-03 | 错误密码返回 401            |
| TC-04 | 正确登录返回 JWT token      |
| TC-05 | 限价单下单成功              |
| TC-06 | 市价单下单成功              |
| TC-07 | 未鉴权下单返回 401          |
| TC-08 | 数量为 0 返回 400           |
| TC-09 | 非法 side 返回 400          |
| TC-10 | 并发 15 请求触发限流（429） |
| TC-11 | 对手方撮合流程完成          |
| TC-12 | WebSocket 握手成功（101）   |

---

## 合约开发与测试

### 环境

```bash
cd SettlementContract
npm install
```

### 编译

```bash
npx hardhat compile
```

### 测试

```bash
npx hardhat test
```

测试覆盖：

- 部署验证
- deposit / withdraw 正常流程
- withdraw 余额不足 revert
- settle 单笔 / 批量结算
- settle 余额不足 revert
- 非 operator 调用 settle revert
- setOperator 权限验证

### 部署到本地 Anvil

```bash
# 启动 Anvil（另一终端）
kubectl port-forward svc/iws-anvil 8545:8545 -n iws-Euler

# 部署
npx hardhat run scripts/deploy.ts --network localhost
```

---

## 代码结构约定

### Go 服务结构

```
<ServiceName>/
├── cmd/
│   └── main.go          # 入口，依赖注入
├── <layer>/             # handler / service / consumer / engine 等
│   ├── <file>.go
│   └── <file>_test.go   # 与被测文件同目录
├── model/               # 数据结构（如有）
├── Dockerfile
└── go.mod
```

### 测试文件位置

- Go 单元测试：与被测文件同目录，`*_test.go`
- Python 单元测试：`RiskControl/tests/`
- 系统 E2E 测试：`tests/e2e_system_test.py`（项目根目录）

---

## 日志查看

```bash
# 实时日志
kubectl logs -n iws-Euler deployment/iws-gateway -f
kubectl logs -n iws-Euler deployment/iws-matchingengine -f
kubectl logs -n iws-Euler deployment/iws-accountservice -f
kubectl logs -n iws-Euler deployment/iws-riskcontrol -f

# 查看上一次崩溃的日志
kubectl logs -n iws-Euler deployment/iws-gateway --previous
```

---

## 常用 kubectl 命令

```bash
# 查看所有 Pod 状态
kubectl get pods -n iws-Euler

# 进入 Pod shell
kubectl exec -it -n iws-Euler deployment/iws-gateway -- sh

# 查看服务（端口信息）
kubectl get svc -n iws-Euler

# 重启单个服务
kubectl rollout restart deployment/iws-gateway -n iws-Euler

# 查看 Helm release 状态
helm status iws-Euler -n iws-Euler
```

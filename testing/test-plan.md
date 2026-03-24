# IWS-Euler 测试计划

**版本**: v1.0
**日期**: 2026-03-22
**状态**: 已执行完毕

---

## 1. 测试目标

验证 IWS-Euler 全栈系统在经过三轮代码审计修复后，所有核心功能正确运行、各服务之间集成无误、修复的 Bug 确已生效。

---

## 2. 测试范围

| 服务               | 测试类型   | 覆盖内容                     |
| ------------------ | ---------- | ---------------------------- |
| MatchingEngine | 单元、集成 | 撮合算法、Kafka 桥接         |
| AccountService | 单元       | 充提、冻结、结算、流水审计   |
| MarketData     | 单元       | K 线聚合、WebSocket 广播     |
| RiskControl    | 单元       | 高频检测、大额检测、幌骗检测 |
| Gateway        | 单元、集成 | 限流修复验证、API 接口       |
| 全系统             | 系统/E2E   | 注册→登录→下单→撮合→行情推送 |

**不在本次范围内**：

- Anvil 链上合约功能测试（需链上测试环境）
- ChainClient 链上结算（无预存用户余额）
- 性能压测（仅有基准测试数据）

---

## 3. 测试类型定义

| 类型              | 定义                                  | 工具                                     |
| ----------------- | ------------------------------------- | ---------------------------------------- |
| **单元测试**      | 单个函数/类的隔离测试，不依赖外部服务 | Go `testing` / Python `unittest`         |
| **集成测试**      | 多组件协作，依赖真实 Kafka            | Go `testing` + kubectl port-forward      |
| **API 接口测试**  | HTTP 接口请求/响应验证                | curl / Python `urllib`                   |
| **系统/E2E 测试** | 完整业务流程端到端验证                | Python `unittest` + kubectl port-forward |
| **性能基准测试**  | 撮合引擎吞吐量                        | Go `testing.B`                           |

---

## 4. 测试环境

| 组件                | 详情                                                     |
| ------------------- | -------------------------------------------------------- |
| K8s 集群            | Docker Desktop 本地集群                                  |
| 命名空间            | iws-Euler                                                |
| Pod 访问方式        | kubectl port-forward                                     |
| Go 测试执行环境     | wishsnow/golang-dev:latest (Docker)                      |
| Python 测试执行环境 | riskcontrol pod (Python 3.11) / 宿主机 Python 3.10.6 |

---

## 5. 测试数据

| 用户     | 密码                 | 预置资金                                |
| -------- | -------------------- | --------------------------------------- |
| alice    | alice123             | 100 BTC（含 10 BTC 冻结）               |
| bob      | bob123               | 10,000,000 USDT（含 500,000 USDT 冻结） |
| 动态用户 | testpass/password123 | 无预置（E2E 测试自动注册）              |

---

## 6. 测试执行命令

```bash
# 单元测试（Go）
docker run --rm -v "T:/IWS-Euler/MatchingEngine:/workspace" wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./engine/... ./bridge/... -v -run TestBridgeProcessOrder -count=1"

docker run --rm -v "T:/IWS-Euler/AccountService:/workspace" wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./service/... -v -count=1"

docker run --rm -v "T:/IWS-Euler/MarketData:/workspace" wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./kline/... ./ws/... -v -count=1"

docker run --rm -v "T:/IWS-Euler/Gateway:/workspace" wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./middleware/... -v -run TestFormatInt -count=1"

# 单元测试（Python）
kubectl exec -n iws-Euler <riskcontrol-pod> -- python -m unittest tests.test_rule_engine -v

# Kafka 集成测试（需 port-forward kafka:9092 -> localhost:9094）
kubectl port-forward svc/kafka 9094:9092 -n iws-Euler
docker run --rm -v "T:/IWS-Euler/MatchingEngine:/workspace" --network host \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./bridge/... -v -run TestBridgeKafkaIntegration -count=1 -timeout=60s"

# 限流 Redis 集成测试（需 port-forward redis:6379 -> localhost:6379）
kubectl port-forward svc/redis 6379:6379 -n iws-Euler
docker run --rm -v "T:/IWS-Euler/Gateway:/workspace" --network host \
  wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./middleware/... -v -run TestRateLimiterIntegration -count=1"

# 系统 E2E 测试（需 port-forward gateway + marketdata）
kubectl port-forward svc/gateway 18083:8081 -n iws-Euler
kubectl port-forward svc/marketdata 18084:8080 -n iws-Euler
PYTHONIOENCODING=utf-8 python T:/IWS-Euler/tests/e2e_system_test.py -v

# 性能基准测试
docker run --rm -v "T:/IWS-Euler/MatchingEngine:/workspace" wishsnow/golang-dev:latest \
  sh -c "cd /workspace && go test ./engine/... -bench=BenchmarkPlaceOrder -benchtime=3s -count=1"
```

---

## 7. 验收标准

- 所有单元测试通过率 100%
- Kafka 集成测试：30s 内完成完整撮合回路
- E2E 系统测试：12 个用例全部通过
- 性能基准：撮合引擎 > 500,000 orders/sec（实测 1.28M/sec）
- 限流：并发 15 请求中 10 通过、5 被拦

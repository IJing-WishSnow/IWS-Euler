# IWS-Euler 生成文档索引

生成日期：2026-03-22

---

## 审计报告

| 文件                                                     | 说明                                              |
| -------------------------------------------------------- | ------------------------------------------------- |
| [fix-summary-2026-03-22.md](./fix-summary-2026-03-22.md) | 三轮审计修复摘要（12 项修复，P0×3 / P1×5 / P2×4） |

---

## 测试文档

| 文件                                               | 说明                               |
| -------------------------------------------------- | ---------------------------------- |
| [testing/test-plan.md](./testing/test-plan.md)     | 测试计划（范围、策略、工具、环境） |
| [testing/test-cases.md](./testing/test-cases.md)   | 测试用例（52 个用例，8 个模块）    |
| [testing/test-report.md](./testing/test-report.md) | 测试执行报告（52/52 通过）         |

---

## 自动化测试代码（源码位置）

| 文件                                                                                                                               | 说明                                      |
| ---------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| [T:/IWS-Euler/IWS-Gateway/middleware/ratelimit_test.go](T:/IWS-Euler/IWS-Gateway/middleware/ratelimit_test.go)                     | Gateway 限流单元 + 集成测试（Go）         |
| [T:/IWS-Euler/IWS-MatchingEngine/engine/engine_test.go](T:/IWS-Euler/IWS-MatchingEngine/engine/engine_test.go)                     | 撮合引擎单元测试（Go）                    |
| [T:/IWS-Euler/IWS-MatchingEngine/bridge/bridge_test.go](T:/IWS-Euler/IWS-MatchingEngine/bridge/bridge_test.go)                     | MatchingEngine Kafka 集成测试（Go）       |
| [T:/IWS-Euler/IWS-AccountService/service/account_service_test.go](T:/IWS-Euler/IWS-AccountService/service/account_service_test.go) | AccountService 单元测试（Go）             |
| [T:/IWS-Euler/IWS-RiskControl/tests/test_rule_engine.py](T:/IWS-Euler/IWS-RiskControl/tests/test_rule_engine.py)                   | 风控规则引擎单元测试（Python，10 个用例） |
| [T:/IWS-Euler/tests/e2e_system_test.py](T:/IWS-Euler/tests/e2e_system_test.py)                                                     | 系统端到端测试（Python，12 个用例）       |
| [T:/IWS-Euler/IWS-SettlementContract/test/IWSSettlement.test.ts](T:/IWS-Euler/IWS-SettlementContract/test/IWSSettlement.test.ts)   | 合约测试（TypeScript，9 个用例）          |

---

## 系统文档（源码仓库中）

| 文件                                                                   | 说明                                           |
| ---------------------------------------------------------------------- | ---------------------------------------------- |
| [T:/IWS-Euler/README.md](T:/IWS-Euler/README.md)                       | 系统总览（架构图、服务列表、快速开始）         |
| [T:/IWS-Euler/docs/architecture.md](T:/IWS-Euler/docs/architecture.md) | 详细架构文档（数据流、Kafka topics、安全设计） |
| [T:/IWS-Euler/docs/deployment.md](T:/IWS-Euler/docs/deployment.md)     | 部署指南（构建镜像、Helm 部署、端口转发）      |
| [T:/IWS-Euler/docs/development.md](T:/IWS-Euler/docs/development.md)   | 开发指南（测试命令、本地调试、日志查看）       |

---

## 服务 README（各服务目录中）

| 服务                   | 路径                                                                                           |
| ---------------------- | ---------------------------------------------------------------------------------------------- |
| IWS-Gateway            | [T:/IWS-Euler/IWS-Gateway/README.md](T:/IWS-Euler/IWS-Gateway/README.md)                       |
| IWS-OrderService       | [T:/IWS-Euler/IWS-OrderService/README.md](T:/IWS-Euler/IWS-OrderService/README.md)             |
| IWS-MatchingEngine     | [T:/IWS-Euler/IWS-MatchingEngine/README.md](T:/IWS-Euler/IWS-MatchingEngine/README.md)         |
| IWS-AccountService     | [T:/IWS-Euler/IWS-AccountService/README.md](T:/IWS-Euler/IWS-AccountService/README.md)         |
| IWS-MarketData         | [T:/IWS-Euler/IWS-MarketData/README.md](T:/IWS-Euler/IWS-MarketData/README.md)                 |
| IWS-ChainClient        | [T:/IWS-Euler/IWS-ChainClient/README.md](T:/IWS-Euler/IWS-ChainClient/README.md)               |
| IWS-RiskControl        | [T:/IWS-Euler/IWS-RiskControl/README.md](T:/IWS-Euler/IWS-RiskControl/README.md)               |
| IWS-Anvil              | [T:/IWS-Euler/IWS-Anvil/README.md](T:/IWS-Euler/IWS-Anvil/README.md)                           |
| IWS-WebApp             | [T:/IWS-Euler/IWS-WebApp/README.md](T:/IWS-Euler/IWS-WebApp/README.md)                         |
| IWS-SettlementContract | [T:/IWS-Euler/IWS-SettlementContract/README.md](T:/IWS-Euler/IWS-SettlementContract/README.md) |
| IWS-Deploy             | [T:/IWS-Euler/IWS-Deploy/README.md](T:/IWS-Euler/IWS-Deploy/README.md)                         |

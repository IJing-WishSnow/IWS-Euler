# IWS-Euler 测试用例清单

**版本**: v1.0 **日期**: 2026-03-22

---

## 模块一：撮合引擎单元测试（MatchingEngine/engine）

| 编号     | 用例名称         | 前置条件                    | 输入                                     | 期望结果                            | 实际结果         |
| -------- | ---------------- | --------------------------- | ---------------------------------------- | ----------------------------------- | ---------------- |
| UT-ME-01 | 限价单完全成交   | 引擎空                      | 卖 price=100 qty=10，买 price=100 qty=10 | 1 笔成交 qty=10 price=100           | PASS             |
| UT-ME-02 | 限价单部分成交   | 引擎空                      | 卖 qty=5，买 qty=10                      | 1 笔成交 qty=5，买单剩余 qty=5 挂单 | PASS             |
| UT-ME-03 | 价格不匹配无成交 | 引擎空                      | 卖 price=200，买 price=100               | 0 笔成交，双方各挂 1 单             | PASS             |
| UT-ME-04 | 市价单多次吃单   | 先挂 sell@100×5，sell@101×5 | 市价买 qty=8                             | 2 笔成交：qty=5@100，qty=3@101      | PASS             |
| UT-ME-05 | 价格优先级       | 先挂 buy@100，buy@200       | 卖 price=100 qty=10                      | 与 buy@200 成交（价格优先）         | PASS             |
| UT-ME-06 | 资产守恒         | 混合订单序列                | 6 笔混合限价单                           | 买卖双方总成交量相等                | PASS             |
| BM-ME-01 | 撮合吞吐量基准   | 预置 1000 卖单              | b.N 买单                                 | > 500K orders/sec                   | PASS (1.28M/sec) |

---

## 模块二：撮合引擎桥接单元测试（MatchingEngine/bridge）

| 编号     | 用例名称         | 前置条件                    | 输入                                   | 期望结果                                 | 实际结果   |
| -------- | ---------------- | --------------------------- | -------------------------------------- | ---------------------------------------- | ---------- |
| UT-BR-01 | Bridge 撮合逻辑  | Bridge 初始化               | alice 买 qty=10，bob 卖 qty=10 @ 50000 | 1 笔成交，BuyUserID=alice SellUserID=bob | PASS       |
| IT-BR-01 | Kafka 全链路集成 | Kafka 可达 (localhost:9094) | 向 orders 写 buy-k1+sell-k1            | trades topic 内出现匹配成交回报          | PASS (18s) |

---

## 模块三：账户服务单元测试（AccountService/service）

| 编号     | 用例名称     | 前置条件                         | 输入                       | 期望结果                              | 实际结果 |
| -------- | ------------ | -------------------------------- | -------------------------- | ------------------------------------- | -------- |
| UT-AS-01 | 充值提现     | 新账户                           | 充值 1000，提现 300        | 余额 700                              | PASS     |
| UT-AS-02 | 超额提现被拒 | 余额 700                         | 提现 800                   | ErrInsufficientBalance                | PASS     |
| UT-AS-03 | 冻结与解冻   | 充值 1000                        | 冻结 400，解冻 400         | Available/Frozen 正确变化，Total 不变 | PASS     |
| UT-AS-04 | 买入成交结算 | 充值 1000 USDT，冻结 500         | SettleBuy(USDT,500,BTC,5)  | USDT.Available=500，BTC.Available=5   | PASS     |
| UT-AS-05 | 卖出成交结算 | 充值 10 BTC，冻结 3              | SettleSell(BTC,3,USDT,300) | BTC.Available=7，USDT.Available=300   | PASS     |
| UT-AS-06 | 流水审计     | 新账户                           | Deposit+Freeze+Unfreeze    | 3 条流水记录                          | PASS     |
| UT-AS-07 | 资产守恒     | buyer=10000 USDT，seller=100 BTC | 完整买卖成交结算           | 系统总 USDT=10000，总 BTC=100         | PASS     |

---

## 模块四：行情服务单元测试（MarketData）

| 编号     | 用例名称       | 前置条件      | 输入                              | 期望结果                                           | 实际结果 |
| -------- | -------------- | ------------- | --------------------------------- | -------------------------------------------------- | -------- |
| UT-MD-01 | K 线聚合       | 新 Aggregator | 同一分钟 4 笔成交                 | OHLCV 正确（O=50000 H=51000 L=49000 C=50500 V=26） | PASS     |
| UT-MD-02 | K 线归档回调   | UT-MD-01 后   | 喂入下一分钟成交                  | 上一根 K 线触发 onKLine 回调                       | PASS     |
| UT-MD-03 | WebSocket 广播 | Hub 初始化    | 1 个 WS 客户端连接，BroadcastTick | 客户端收到 type=tick 消息                          | PASS     |

---

## 模块五：风控规则引擎单元测试（RiskControl/rules）

| 编号     | 用例名称         | 前置条件   | 输入                          | 期望结果                                     | 实际结果 |
| -------- | ---------------- | ---------- | ----------------------------- | -------------------------------------------- | -------- |
| UT-RC-01 | 正常频率不报警   | 新引擎     | 5 笔下单                      | 无 HIGH_FREQUENCY 报警                       | PASS     |
| UT-RC-02 | 高频触发报警     | 新引擎     | 21 笔下单（> 阈值 20）        | 至少 1 条 HIGH_FREQUENCY 报警                | PASS     |
| UT-RC-03 | 滑动窗口过期     | window=1s  | 25 笔下单后等 1.1s，再下 1 单 | 无新的高频报警                               | PASS     |
| UT-RC-04 | 小额订单不报警   | 新引擎     | price=100 qty=10 (value=1000) | 无 LARGE_ORDER 报警                          | PASS     |
| UT-RC-05 | 大额订单报警     | 新引擎     | price=threshold qty=2         | 1 条 LARGE_ORDER 报警                        | PASS     |
| UT-RC-06 | 等于阈值不报警   | 新引擎     | value == threshold            | 无报警（严格 >）                             | PASS     |
| UT-RC-07 | 均衡买卖不报幌骗 | 新引擎     | 5 buy + 5 sell                | 无 SPOOFING_SUSPECT                          | PASS     |
| UT-RC-08 | 同向大量挂单幌骗 | 新引擎     | 10 笔 buy                     | 至少 1 条 SPOOFING_SUSPECT                   | PASS     |
| UT-RC-09 | 成交统计累积     | 新引擎     | 2 笔成交（alice-bob, qty=5）  | alice/bob count=2 volume=10                  | PASS     |
| UT-RC-10 | 摘要结构正确     | 报警已产生 | get_summary()                 | 含 total_alerts/alerts_by_rule/recent_alerts | PASS     |

---

## 模块六：限流器单元测试（Gateway/middleware）

| 编号     | 用例名称                 | 前置条件   | 输入                  | 期望结果                             | 实际结果 |
| -------- | ------------------------ | ---------- | --------------------- | ------------------------------------ | -------- |
| UT-RL-01 | formatInt 输出数字字符串 | —          | formatInt(now)        | 纯数字字符串 == now                  | PASS     |
| UT-RL-02 | formatInt 值合理范围     | —          | formatInt(now)        | < 3万亿（非日期格式）                | PASS     |
| UT-RL-03 | 滑动窗口 cutoff 正确     | —          | cutoff = now - 1000ms | cutoffStr 解析值 == cutoff，且 < now | PASS     |
| IT-RL-01 | Redis 滑动窗口集成       | Redis 可达 | 3 次请求后检查记录数  | 记录数 >= limit=3                    | PASS     |

---

## 模块七：Gateway API 接口测试

| 编号   | 用例名称          | 接口            | 输入               | 期望                   | 实际 |
| ------ | ----------------- | --------------- | ------------------ | ---------------------- | ---- |
| API-01 | 新用户注册        | POST /register  | 新用户名+密码      | 200 + message          | PASS |
| API-02 | 重复注册          | POST /register  | 同一用户名         | 409                    | PASS |
| API-03 | 错误密码登录      | POST /login     | 错误密码           | 401                    | PASS |
| API-04 | 正确凭证登录      | POST /login     | 正确凭证           | 200 + JWT token (3段)  | PASS |
| API-05 | 无 token 下单     | POST /api/order | 无 Authorization   | 401                    | PASS |
| API-06 | 携带 token 限价单 | POST /api/order | token + limit buy  | 200 + order_id         | PASS |
| API-07 | 携带 token 市价单 | POST /api/order | token + market buy | 200 + order_id         | PASS |
| API-08 | 数量为 0 被拒     | POST /api/order | quantity=0         | 400                    | PASS |
| API-09 | 非法 side 被拒    | POST /api/order | side="long"        | 400                    | PASS |
| API-10 | 并发限流          | POST /api/order | 15 并发请求        | 10 通过 + 5 被拦 (429) | PASS |

---

## 模块八：系统 E2E 测试

| 编号   | 用例名称       | 场景                      | 期望结果                | 实际结果 |
| ------ | -------------- | ------------------------- | ----------------------- | -------- |
| E2E-01 | 用户注册       | 新用户注册                | 200 + message           | PASS     |
| E2E-02 | 重复注册拒绝   | 已存在用户名              | 409                     | PASS     |
| E2E-03 | 错误密码拒绝   | 错误密码                  | 401                     | PASS     |
| E2E-04 | JWT 登录       | 正确凭证                  | JWT token 三段格式      | PASS     |
| E2E-05 | 限价单下单     | 已登录用户                | 200 + order_id          | PASS     |
| E2E-06 | 市价单下单     | 已登录用户                | 200 + order_id          | PASS     |
| E2E-07 | 未鉴权拒绝     | 无 token                  | 401                     | PASS     |
| E2E-08 | 参数校验       | quantity=0                | 400                     | PASS     |
| E2E-09 | 非法参数校验   | side=long                 | 400                     | PASS     |
| E2E-10 | 并发限流       | 15 并发                   | 10 通过 5 被拦          | PASS     |
| E2E-11 | 撮合流程       | alice 卖 + bob 买 @ 50000 | 订单进入撮合队列        | PASS     |
| E2E-12 | WebSocket 握手 | 连接 /ws                  | 101 Switching Protocols | PASS     |

---

## 汇总

| 测试类型           | 用例数 | 通过   | 失败  | 跳过  | 通过率   |
| ------------------ | ------ | ------ | ----- | ----- | -------- |
| 单元测试（Go）     | 18     | 18     | 0     | 0     | 100%     |
| 单元测试（Python） | 10     | 10     | 0     | 0     | 100%     |
| 集成测试（Kafka）  | 1      | 1      | 0     | 0     | 100%     |
| 集成测试（Redis）  | 1      | 1      | 0     | 0     | 100%     |
| API 接口测试       | 10     | 10     | 0     | 0     | 100%     |
| 系统 E2E 测试      | 12     | 12     | 0     | 0     | 100%     |
| **合计**           | **52** | **52** | **0** | **0** | **100%** |

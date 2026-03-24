# RiskControl

实时风控引擎，Python 实现，同时监听订单和成交消息，检测异常交易行为。

## 职责

- 消费 Kafka `orders` + `trades` 两个 topic
- 执行规则引擎，触发告警
- 提供 HTTP 接口查询风控统计摘要

## 风控规则

| 规则     | 触发条件                             | 告警类型           |
| -------- | ------------------------------------ | ------------------ |
| 高频下单 | 同一用户 60 秒内下单超过 20 次       | `HIGH_FREQUENCY`   |
| 大额订单 | 单笔金额（price × quantity）超过阈值 | `LARGE_ORDER`      |
| 幌骗检测 | 同方向挂单数量异常集中               | `SPOOFING_SUSPECT` |

## HTTP 接口

| 方法 | 路径       | 说明             |
| ---- | ---------- | ---------------- |
| GET  | `/summary` | 返回风控统计摘要 |

摘要格式：

```json
{
  "total_alerts": 3,
  "alerts_by_rule": {"HIGH_FREQUENCY": 1, "LARGE_ORDER": 2},
  "recent_alerts": [...],
  "trade_stats": {"alice": {"count": 5, "volume": 10}}
}
```

## Kafka

|      | Topic    | 消费组                     |
| ---- | -------- | -------------------------- |
| 输入 | `orders` | `riskcontrol-orders-group` |
| 输入 | `trades` | `riskcontrol-trades-group` |

## 目录结构

```
RiskControl/
├── rules/
│   └── engine.py           # 规则引擎核心（RuleEngine, Alert）
├── consumer/
│   └── kafka_consumer.py   # Kafka 消费逻辑
├── server.py               # HTTP 服务入口（Flask）
├── tests/
│   └── test_rule_engine.py # 规则引擎单元测试（10 个用例）
├── requirements.txt
└── Dockerfile
```

## 运行测试

```bash
# 在 Pod 内运行（推荐）
POD=$(kubectl get pod -n iws-Euler -l app=iws-riskcontrol -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n iws-Euler $POD -- python -m unittest tests.test_rule_engine -v

# 本地运行
cd RiskControl
pip install -r requirements.txt
python -m unittest tests.test_rule_engine -v
```

测试覆盖：

| 用例                                          | 描述                        |
| --------------------------------------------- | --------------------------- |
| test_normal_frequency_no_alert                | 5 次下单不触发高频报警      |
| test_high_frequency_triggers_alert            | 超过阈值触发 HIGH_FREQUENCY |
| test_sliding_window_expires_old_orders        | 窗口过期后不触发报警        |
| test_small_order_no_alert                     | 小额订单不触发报警          |
| test_large_order_triggers_alert               | 超阈值触发 LARGE_ORDER      |
| test_boundary_exactly_at_threshold_no_alert   | 等于阈值不触发（严格 >）    |
| test_balanced_sides_no_spoofing               | 买卖均衡不触发幌骗          |
| test_one_sided_large_volume_triggers_spoofing | 单向大量挂单触发幌骗        |
| test_trade_stats_accumulated                  | 成交统计正确累积            |
| test_summary_structure                        | get_summary 返回正确结构    |

## 无对外固定端口

RiskControl HTTP 服务端口在 K8s 内通过 ClusterIP 暴露，风控统计通过日志和 `/summary` 接口查询。

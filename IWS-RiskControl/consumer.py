"""Kafka 消费者：消费订单流和成交流，喂入风控规则引擎"""
import json
import os
import threading
import time
from kafka import KafkaConsumer
from kafka.errors import NoBrokersAvailable

KAFKA_BROKER = os.getenv("KAFKA_BROKER", "localhost:9094")
ORDERS_TOPIC = "orders"
TRADES_TOPIC = "trades"
RETRY_INTERVAL = 5  # 秒


class RiskConsumer:
    def __init__(self, rule_engine):
        self.rule_engine = rule_engine
        self._running = False

    def start(self):
        self._running = True
        t1 = threading.Thread(target=self._consume_orders, daemon=True)
        t2 = threading.Thread(target=self._consume_trades, daemon=True)
        t1.start()
        t2.start()
        print("[RiskControl] Kafka 消费者启动")

    def stop(self):
        self._running = False

    def _consume_orders(self):
        while self._running:
            try:
                consumer = KafkaConsumer(
                    ORDERS_TOPIC,
                    bootstrap_servers=KAFKA_BROKER,
                    group_id="risk-control-orders",
                    auto_offset_reset="earliest",
                    value_deserializer=lambda m: json.loads(m.decode("utf-8")),
                )
                print(f"[RiskControl] 已连接 Kafka，监听 {ORDERS_TOPIC}")
                for msg in consumer:
                    if not self._running:
                        break
                    self.rule_engine.on_order(msg.value)
            except NoBrokersAvailable:
                print(f"[RiskControl] Kafka 不可用，{RETRY_INTERVAL}s 后重试...")
                time.sleep(RETRY_INTERVAL)
            except Exception as e:
                print(f"[RiskControl] orders 消费异常: {e}，{RETRY_INTERVAL}s 后重试...")
                time.sleep(RETRY_INTERVAL)

    def _consume_trades(self):
        while self._running:
            try:
                consumer = KafkaConsumer(
                    TRADES_TOPIC,
                    bootstrap_servers=KAFKA_BROKER,
                    group_id="risk-control-trades",
                    auto_offset_reset="earliest",
                    value_deserializer=lambda m: json.loads(m.decode("utf-8")),
                )
                print(f"[RiskControl] 已连接 Kafka，监听 {TRADES_TOPIC}")
                for msg in consumer:
                    if not self._running:
                        break
                    self.rule_engine.on_trade(msg.value)
            except NoBrokersAvailable:
                print(f"[RiskControl] Kafka 不可用，{RETRY_INTERVAL}s 后重试...")
                time.sleep(RETRY_INTERVAL)
            except Exception as e:
                print(f"[RiskControl] trades 消费异常: {e}，{RETRY_INTERVAL}s 后重试...")
                time.sleep(RETRY_INTERVAL)

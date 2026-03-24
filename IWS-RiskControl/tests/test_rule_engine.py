"""IWS-RiskControl 规则引擎单元测试"""
import sys
import os
import time
import unittest

# 把项目根目录加入 PYTHONPATH
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from rules.engine import RuleEngine, Alert


class TestRuleEngineFrequency(unittest.TestCase):
    """高频下单检测"""

    def setUp(self):
        self.engine = RuleEngine()

    def _make_order(self, user_id="user1", side="buy", price=1000, qty=1):
        return {
            "user_id": user_id,
            "side": side,
            "type": "limit",
            "price": price,
            "quantity": qty,
            "symbol": "BTC/USDT",
        }

    def test_normal_frequency_no_alert(self):
        """正常频率下单不触发报警"""
        for i in range(5):
            self.engine.on_order(self._make_order())
        alerts = [a for a in self.engine.alerts if a.rule == "HIGH_FREQUENCY"]
        self.assertEqual(len(alerts), 0, f"5 次下单不应触发高频报警，实际 {len(alerts)} 条")

    def test_high_frequency_triggers_alert(self):
        """超过阈值（20次/分钟）触发高频报警"""
        threshold = self.engine.max_orders_per_min
        for i in range(threshold + 1):
            self.engine.on_order(self._make_order())
        alerts = [a for a in self.engine.alerts if a.rule == "HIGH_FREQUENCY"]
        self.assertGreater(len(alerts), 0, f"超过 {threshold} 次下单应触发高频报警")
        print(f"  高频报警触发 ✓ ({len(alerts)} 条，阈值={threshold})")

    def test_sliding_window_expires_old_orders(self):
        """滑动窗口：过期订单不计入当前窗口"""
        engine = RuleEngine()
        engine.window_sec = 1  # 缩短窗口到 1 秒便于测试
        for i in range(25):
            engine.on_order(self._make_order())
        initial_alerts = len([a for a in engine.alerts if a.rule == "HIGH_FREQUENCY"])

        # 等待窗口过期
        time.sleep(1.1)

        # 重置 alert 列表，下 1 单，不应触发高频
        engine.alerts.clear()
        engine.on_order(self._make_order())
        new_alerts = [a for a in engine.alerts if a.rule == "HIGH_FREQUENCY"]
        self.assertEqual(len(new_alerts), 0, "窗口过期后 1 次下单不应触发高频报警")
        print(f"  滑动窗口过期验证 ✓（初始报警={initial_alerts} 条，过期后新增=0）")


class TestRuleEngineLargeOrder(unittest.TestCase):
    """大额订单检测"""

    def setUp(self):
        self.engine = RuleEngine()

    def _make_order(self, price, qty, user_id="user1"):
        return {"user_id": user_id, "side": "buy", "type": "limit",
                "price": price, "quantity": qty, "symbol": "BTC/USDT"}

    def test_small_order_no_alert(self):
        """小额订单不触发报警"""
        self.engine.on_order(self._make_order(price=100, qty=10))  # value=1000
        alerts = [a for a in self.engine.alerts if a.rule == "LARGE_ORDER"]
        self.assertEqual(len(alerts), 0)

    def test_large_order_triggers_alert(self):
        """超过阈值的大额订单触发报警"""
        threshold = self.engine.large_order_threshold
        # value = price * qty > threshold
        self.engine.on_order(self._make_order(price=threshold, qty=2))
        alerts = [a for a in self.engine.alerts if a.rule == "LARGE_ORDER"]
        self.assertGreater(len(alerts), 0)
        print(f"  大额订单报警触发 ✓ (value={threshold * 2} > 阈值={threshold})")

    def test_boundary_exactly_at_threshold_no_alert(self):
        """恰好等于阈值不触发报警（严格大于才触发）"""
        threshold = self.engine.large_order_threshold
        self.engine.on_order(self._make_order(price=threshold, qty=1))  # value == threshold
        alerts = [a for a in self.engine.alerts if a.rule == "LARGE_ORDER"]
        self.assertEqual(len(alerts), 0, "等于阈值不应触发报警（严格 >）")


class TestRuleEngineSpoofing(unittest.TestCase):
    """幌骗检测"""

    def setUp(self):
        self.engine = RuleEngine()

    def _make_order(self, user_id="spoofer", side="buy"):
        return {"user_id": user_id, "side": side, "type": "limit",
                "price": 50000, "quantity": 1, "symbol": "BTC/USDT"}

    def test_balanced_sides_no_spoofing(self):
        """买卖均衡不触发幌骗"""
        for i in range(5):
            self.engine.on_order(self._make_order(side="buy"))
            self.engine.on_order(self._make_order(side="sell"))
        alerts = [a for a in self.engine.alerts if a.rule == "SPOOFING_SUSPECT"]
        self.assertEqual(len(alerts), 0)

    def test_one_sided_large_volume_triggers_spoofing(self):
        """大量同方向挂单触发幌骗"""
        for i in range(10):
            self.engine.on_order(self._make_order(side="buy"))
        alerts = [a for a in self.engine.alerts if a.rule == "SPOOFING_SUSPECT"]
        self.assertGreater(len(alerts), 0, "10 笔同向买单应触发幌骗报警")
        print(f"  幌骗报警触发 ✓ ({len(alerts)} 条)")


class TestRuleEngineTrade(unittest.TestCase):
    """成交统计"""

    def setUp(self):
        self.engine = RuleEngine()

    def test_trade_stats_accumulated(self):
        """成交统计正确累积"""
        trade = {"buy_user_id": "alice", "sell_user_id": "bob",
                 "quantity": 5, "price": 50000}
        self.engine.on_trade(trade)
        self.engine.on_trade(trade)

        self.assertEqual(self.engine.trade_stats["alice"]["count"], 2)
        self.assertEqual(self.engine.trade_stats["alice"]["volume"], 10)
        self.assertEqual(self.engine.trade_stats["bob"]["count"], 2)
        print(f"  成交统计 ✓ alice: count=2 volume=10")


class TestRuleEngineGetSummary(unittest.TestCase):
    """摘要接口"""

    def test_summary_structure(self):
        """get_summary 返回正确结构"""
        engine = RuleEngine()
        engine.on_order({"user_id": "u1", "side": "buy", "type": "limit",
                         "price": 999999, "quantity": 999, "symbol": "BTC/USDT"})
        summary = engine.get_summary()

        self.assertIn("total_alerts", summary)
        self.assertIn("alerts_by_rule", summary)
        self.assertIn("recent_alerts", summary)
        self.assertIn("trade_stats", summary)
        self.assertGreater(summary["total_alerts"], 0)
        print(f"  摘要结构 ✓ total_alerts={summary['total_alerts']}")


if __name__ == "__main__":
    unittest.main(verbosity=2)

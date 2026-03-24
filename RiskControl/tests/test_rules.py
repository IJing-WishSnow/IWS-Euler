"""风控规则引擎测试"""
import sys
import os
sys.stdout.reconfigure(encoding='utf-8')
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from rules.engine import RuleEngine


def test_high_frequency():
    """高频下单检测"""
    engine = RuleEngine()
    engine.max_orders_per_min = 5  # 降低阈值方便测试

    for i in range(6):
        engine.on_order({"id": f"order-{i}", "user_id": "alice", "side": "buy",
                         "type": "limit", "price": 50000, "quantity": 1, "symbol": "BTC/USDT"})

    alerts = [a for a in engine.alerts if a.rule == "HIGH_FREQUENCY"]
    assert len(alerts) > 0, "应检测到高频下单"
    print(f"✓ 高频检测通过: {alerts[0].detail}")


def test_large_order():
    """大额订单检测"""
    engine = RuleEngine()
    engine.large_order_threshold = 1000

    engine.on_order({"id": "big-1", "user_id": "bob", "side": "buy",
                     "type": "limit", "price": 100, "quantity": 20, "symbol": "BTC/USDT"})

    alerts = [a for a in engine.alerts if a.rule == "LARGE_ORDER"]
    assert len(alerts) > 0, "应检测到大额订单"
    print(f"✓ 大额检测通过: {alerts[0].detail}")


def test_spoofing():
    """幌骗检测"""
    engine = RuleEngine()

    # 连续 10 个同方向买单
    for i in range(10):
        engine.on_order({"id": f"spoof-{i}", "user_id": "charlie", "side": "buy",
                         "type": "limit", "price": 50000, "quantity": 1, "symbol": "BTC/USDT"})

    alerts = [a for a in engine.alerts if a.rule == "SPOOFING_SUSPECT"]
    assert len(alerts) > 0, "应检测到幌骗嫌疑"
    print(f"✓ 幌骗检测通过: {alerts[0].detail}")


def test_trade_stats():
    """成交统计"""
    engine = RuleEngine()

    engine.on_trade({"buy_user_id": "alice", "sell_user_id": "bob",
                     "price": 50000, "quantity": 10})
    engine.on_trade({"buy_user_id": "alice", "sell_user_id": "charlie",
                     "price": 51000, "quantity": 5})

    assert engine.trade_stats["alice"]["count"] == 2
    assert engine.trade_stats["alice"]["volume"] == 15
    assert engine.trade_stats["bob"]["count"] == 1
    print(f"✓ 成交统计通过: alice={engine.trade_stats['alice']}")


def test_offline_report():
    """离线报告生成"""
    from report.generator import ReportGenerator

    engine = RuleEngine()
    engine.max_orders_per_min = 3

    for i in range(5):
        engine.on_order({"id": f"o-{i}", "user_id": "alice", "side": "buy",
                         "type": "limit", "price": 50000, "quantity": 1, "symbol": "BTC/USDT"})

    gen = ReportGenerator()
    report = gen.generate_offline(engine.get_summary())
    assert "告警总数" in report
    assert "alice" in report
    print(f"✓ 离线报告生成通过:\n{report}")


def test_llm_report():
    """LLM 真实调用测试（需要 OPENAI_API_KEY 和 OPENAI_BASE_URL 环境变量）"""
    import os
    if not os.getenv("OPENAI_API_KEY"):
        print("⚠ 跳过 LLM 测试：未设置 OPENAI_API_KEY")
        return

    from report.generator import ReportGenerator

    engine = RuleEngine()
    engine.max_orders_per_min = 3
    for i in range(5):
        engine.on_order({"id": f"o-{i}", "user_id": "alice", "side": "buy",
                         "type": "limit", "price": 50000, "quantity": 1, "symbol": "BTC/USDT"})
    engine.on_trade({"buy_user_id": "alice", "sell_user_id": "bob",
                     "price": 50000, "quantity": 10})

    gen = ReportGenerator()
    report = gen.generate(engine.get_summary())
    assert "[报告生成失败]" not in report, f"LLM 调用失败: {report}"
    assert len(report) > 50, "报告内容过短"
    print(f"✓ LLM 报告生成通过:\n{report}")


if __name__ == "__main__":
    test_high_frequency()
    test_large_order()
    test_spoofing()
    test_trade_stats()
    test_offline_report()
    test_llm_report()
    print("\n全部测试通过")

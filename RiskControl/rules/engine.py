"""风控规则引擎：实时检测异常交易模式"""
import time
from collections import defaultdict
from dataclasses import dataclass, field


@dataclass
class Alert:
    rule: str
    user_id: str
    detail: str
    timestamp: float = field(default_factory=time.time)


class RuleEngine:
    def __init__(self):
        # 滑动窗口：用户最近 N 秒的订单记录
        self.order_window: dict[str, list[dict]] = defaultdict(list)
        # 用户成交统计
        self.trade_stats: dict[str, dict] = defaultdict(lambda: {"count": 0, "volume": 0})
        # 告警列表
        self.alerts: list[Alert] = []
        # 配置
        self.window_sec = 60  # 滑动窗口 60 秒
        self.max_orders_per_min = 20  # 每分钟最多下单数
        self.large_order_threshold = 100_000  # 大额订单阈值（price * quantity）

    def on_order(self, order: dict):
        """处理新订单，检测异常"""
        user_id = order.get("user_id", "unknown")
        now = time.time()

        # 记录到滑动窗口
        self.order_window[user_id].append({**order, "_ts": now})
        # 清理过期记录
        self._clean_window(user_id, now)

        # 规则1：高频下单检测
        self._check_frequency(user_id)

        # 规则2：大额订单检测
        self._check_large_order(user_id, order)

        # 规则3：幌骗检测（短时间内大量同方向挂单）
        self._check_spoofing(user_id)

    def on_trade(self, trade: dict):
        """处理成交，更新统计"""
        for uid in [trade.get("buy_user_id"), trade.get("sell_user_id")]:
            if uid:
                self.trade_stats[uid]["count"] += 1
                self.trade_stats[uid]["volume"] += trade.get("quantity", 0)

    def _check_frequency(self, user_id: str):
        """高频下单：1分钟内超过阈值"""
        orders = self.order_window[user_id]
        if len(orders) > self.max_orders_per_min:
            self._alert("HIGH_FREQUENCY", user_id,
                        f"1分钟内下单 {len(orders)} 次，超过阈值 {self.max_orders_per_min}")

    def _check_large_order(self, user_id: str, order: dict):
        """大额订单检测"""
        value = order.get("price", 0) * order.get("quantity", 0)
        if value > self.large_order_threshold:
            self._alert("LARGE_ORDER", user_id,
                        f"大额订单 value={value}，超过阈值 {self.large_order_threshold}")

    def _check_spoofing(self, user_id: str):
        """幌骗检测：短时间内同方向大量挂单（可能是虚假流动性）"""
        orders = self.order_window[user_id]
        if len(orders) < 5:
            return

        # 统计最近订单的方向分布
        sides = [o.get("side") for o in orders[-10:]]
        buy_count = sides.count("buy")
        sell_count = sides.count("sell")

        # 如果 90% 以上是同方向，且数量超过 8 单
        total = len(sides)
        if total >= 8 and (buy_count / total > 0.9 or sell_count / total > 0.9):
            dominant = "buy" if buy_count > sell_count else "sell"
            self._alert("SPOOFING_SUSPECT", user_id,
                        f"疑似幌骗：最近 {total} 单中 {max(buy_count, sell_count)} 单为 {dominant}")

    def _clean_window(self, user_id: str, now: float):
        """清理滑动窗口中过期的记录"""
        cutoff = now - self.window_sec
        self.order_window[user_id] = [
            o for o in self.order_window[user_id] if o.get("_ts", 0) > cutoff
        ]

    def _alert(self, rule: str, user_id: str, detail: str):
        alert = Alert(rule=rule, user_id=user_id, detail=detail)
        self.alerts.append(alert)
        print(f"[ALERT] {rule} | {user_id} | {detail}")

    def get_alerts(self, limit: int = 50) -> list[Alert]:
        return self.alerts[-limit:]

    def get_summary(self) -> dict:
        """获取风控摘要，供 LLM 生成报告"""
        return {
            "total_alerts": len(self.alerts),
            "alerts_by_rule": self._count_by_rule(),
            "alerts_by_user": self._count_by_user(),
            "recent_alerts": [
                {"rule": a.rule, "user": a.user_id, "detail": a.detail}
                for a in self.alerts[-20:]
            ],
            "trade_stats": dict(self.trade_stats),
        }

    def _count_by_rule(self) -> dict:
        counts: dict[str, int] = defaultdict(int)
        for a in self.alerts:
            counts[a.rule] += 1
        return dict(counts)

    def _count_by_user(self) -> dict:
        counts: dict[str, int] = defaultdict(int)
        for a in self.alerts:
            counts[a.user_id] += 1
        return dict(counts)

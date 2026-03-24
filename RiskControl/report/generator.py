"""LLM 风险报告生成器：将风控摘要转化为可读的分析报告"""
import json
import os
from openai import OpenAI


SYSTEM_PROMPT = """你是一个专业的金融风控分析师。
根据提供的交易风控数据摘要，生成一份简洁的风险分析报告。
报告应包含：
1. 风险概览（告警总数、分布）
2. 重点关注用户及其异常行为
3. 风险等级评估（低/中/高）
4. 建议措施
用中文回答，保持专业简洁。"""


class ReportGenerator:
    def __init__(self, api_key: str = None, base_url: str = None):
        self.client = OpenAI(
            api_key=api_key or os.getenv("OPENAI_API_KEY", ""),
            base_url=base_url or os.getenv("OPENAI_BASE_URL"),
        )
        self.model = os.getenv("OPENAI_MODEL", "deepseek-chat")

    def generate(self, summary: dict) -> str:
        """调用 LLM 生成风险报告"""
        try:
            response = self.client.chat.completions.create(
                model=self.model,
                messages=[
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": f"以下是风控数据摘要：\n{json.dumps(summary, ensure_ascii=False, indent=2)}"},
                ],
                temperature=0.3,
                max_tokens=1000,
            )
            return response.choices[0].message.content
        except Exception as e:
            return f"[报告生成失败] {e}"

    def generate_offline(self, summary: dict) -> str:
        """离线模式：不调用 LLM，直接生成结构化报告"""
        total = summary.get("total_alerts", 0)
        by_rule = summary.get("alerts_by_rule", {})
        by_user = summary.get("alerts_by_user", {})
        recent = summary.get("recent_alerts", [])

        # 风险等级
        if total == 0:
            level = "低"
        elif total < 10:
            level = "中"
        else:
            level = "高"

        lines = [
            "=" * 50,
            "交易风控日报",
            "=" * 50,
            f"\n风险等级: {level}",
            f"告警总数: {total}",
            "\n--- 按规则分布 ---",
        ]
        for rule, count in by_rule.items():
            lines.append(f"  {rule}: {count} 次")

        lines.append("\n--- 按用户分布 ---")
        for user, count in sorted(by_user.items(), key=lambda x: -x[1]):
            lines.append(f"  {user}: {count} 次告警")

        if recent:
            lines.append("\n--- 最近告警 ---")
            for a in recent[-5:]:
                lines.append(f"  [{a['rule']}] {a['user']}: {a['detail']}")

        lines.append("\n" + "=" * 50)
        return "\n".join(lines)

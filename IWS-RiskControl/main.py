"""IWS-RiskControl 入口"""
import signal
import time
from consumer import RiskConsumer
from rules.engine import RuleEngine

def main():
    engine = RuleEngine()
    consumer = RiskConsumer(engine)
    consumer.start()

    def shutdown(signum, frame):
        print("[RiskControl] 收到停止信号，正在退出...")
        consumer.stop()

    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)

    print("[RiskControl] 服务已启动，按 Ctrl+C 停止")
    while True:
        time.sleep(1)

if __name__ == "__main__":
    main()

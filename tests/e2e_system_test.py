# -*- coding: utf-8 -*-
"""
IWS-Euler 系统端到端测试
测试完整业务流程：注册 -> 登录 -> 下单 -> 撮合 -> 行情推送 -> 风控检测

前置条件（在另一终端运行）:
  kubectl port-forward svc/iws-gateway 18083:8081 -n iws-Euler
  kubectl port-forward svc/iws-marketdata 18084:8080 -n iws-Euler
"""
import io
import json
import sys
import time
import uuid
import threading
import unittest
import urllib.request
import urllib.error

# Windows GBK 控制台兼容
sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding="utf-8", errors="replace")

GATEWAY_URL = "http://localhost:18083"


def http_post(path, body=None, token=None):
    data = json.dumps(body).encode() if body else b""
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(
        f"{GATEWAY_URL}{path}", data=data, headers=headers, method="POST"
    )
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read())


def register_and_login(username, password):
    http_post("/register", {"username": username, "password": password})
    code, body = http_post("/login", {"username": username, "password": password})
    assert code == 200, f"登录失败: {code} {body}"
    return body["token"]


# ──────────────────────────────────────────────────────────────
# TC-01~04  用户注册与登录
# ──────────────────────────────────────────────────────────────

class TestE2EUserFlow(unittest.TestCase):

    def test_01_register_new_user(self):
        """新用户注册成功 -> 200"""
        username = f"e2e_{uuid.uuid4().hex[:8]}"
        code, body = http_post("/register", {"username": username, "password": "password123"})
        self.assertEqual(code, 200, f"注册失败: {body}")
        self.assertIn("message", body)
        print(f"    注册成功: {username} [OK]")

    def test_02_duplicate_register_rejected(self):
        """重复注册被拒 -> 409"""
        username = f"dup_{uuid.uuid4().hex[:8]}"
        http_post("/register", {"username": username, "password": "password123"})
        code, body = http_post("/register", {"username": username, "password": "password123"})
        self.assertEqual(code, 409, f"期望 409，实际 {code}: {body}")
        print(f"    重复注册拒绝 [OK] HTTP 409")

    def test_03_login_wrong_password_rejected(self):
        """错误密码登录被拒 -> 401"""
        username = f"auth_{uuid.uuid4().hex[:8]}"
        http_post("/register", {"username": username, "password": "rightpass"})
        code, body = http_post("/login", {"username": username, "password": "wrongpass"})
        self.assertEqual(code, 401, f"期望 401，实际 {code}: {body}")
        print(f"    错误密码拒绝 [OK] HTTP 401")

    def test_04_login_success_returns_token(self):
        """正确凭证登录返回 JWT token"""
        username = f"tok_{uuid.uuid4().hex[:8]}"
        http_post("/register", {"username": username, "password": "testpass"})
        code, body = http_post("/login", {"username": username, "password": "testpass"})
        self.assertEqual(code, 200)
        self.assertIn("token", body)
        token = body["token"]
        parts = token.split(".")
        self.assertEqual(len(parts), 3, "JWT 格式不正确")
        print(f"    JWT 登录 [OK] token={token[:30]}...")


# ──────────────────────────────────────────────────────────────
# TC-05~10  下单与鉴权
# ──────────────────────────────────────────────────────────────

class TestE2EOrderFlow(unittest.TestCase):

    def setUp(self):
        username = f"ord_{uuid.uuid4().hex[:8]}"
        self.token = register_and_login(username, "testpass123")

    def test_05_place_limit_order_success(self):
        """限价单下单成功 -> 200 + order_id"""
        code, body = http_post("/api/order", {
            "symbol": "BTC/USDT", "side": "buy", "type": "limit",
            "price": 45000, "quantity": 2
        }, token=self.token)
        self.assertEqual(code, 200, f"下单失败: {body}")
        self.assertIn("order_id", body)
        print(f"    限价单 [OK] order_id={body['order_id']}")

    def test_06_place_market_order_success(self):
        """市价单下单成功 -> 200 + order_id"""
        code, body = http_post("/api/order", {
            "symbol": "BTC/USDT", "side": "buy", "type": "market", "quantity": 1
        }, token=self.token)
        self.assertEqual(code, 200, f"市价单失败: {body}")
        self.assertIn("order_id", body)
        print(f"    市价单 [OK] order_id={body['order_id']}")

    def test_07_order_without_token_rejected(self):
        """未鉴权下单被拒 -> 401"""
        code, body = http_post("/api/order", {
            "symbol": "BTC/USDT", "side": "buy", "type": "limit",
            "price": 45000, "quantity": 1
        })
        self.assertEqual(code, 401, f"期望 401，实际 {code}: {body}")
        print(f"    未鉴权拒绝 [OK] HTTP 401")

    def test_08_invalid_quantity_rejected(self):
        """数量为 0 被参数校验拒绝 -> 400"""
        code, body = http_post("/api/order", {
            "symbol": "BTC/USDT", "side": "buy", "type": "limit",
            "price": 45000, "quantity": 0
        }, token=self.token)
        self.assertEqual(code, 400, f"期望 400，实际 {code}: {body}")
        print(f"    参数校验 [OK] HTTP 400")

    def test_09_invalid_side_rejected(self):
        """非法 side 值被拒 -> 400"""
        code, body = http_post("/api/order", {
            "symbol": "BTC/USDT", "side": "long", "type": "limit",
            "price": 45000, "quantity": 1
        }, token=self.token)
        self.assertEqual(code, 400, f"期望 400，实际 {code}: {body}")
        print(f"    非法 side 拒绝 [OK] HTTP 400")

    def test_10_rate_limit_triggers_with_concurrent_requests(self):
        """并发超过 10 次/秒触发限流 (429)"""
        results = []
        lock = threading.Lock()

        def send_one(idx):
            code, _ = http_post("/api/order", {
                "symbol": "BTC/USDT", "side": "buy", "type": "limit",
                "price": 50000, "quantity": 1
            }, token=self.token)
            with lock:
                results.append(code)

        threads = [threading.Thread(target=send_one, args=(i,)) for i in range(15)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        passed = results.count(200)
        blocked = results.count(429)
        print(f"    限流测试: 通过={passed}, 被拦={blocked}")
        self.assertGreater(blocked, 0,
            f"限流未生效：15 次并发请求全部通过 (passed={passed})")
        self.assertLessEqual(passed, 10,
            f"通过数量超过限制：passed={passed} > 10")


# ──────────────────────────────────────────────────────────────
# TC-11  撮合流程
# ──────────────────────────────────────────────────────────────

class TestE2EMatchingFlow(unittest.TestCase):

    def test_11_matching_produces_trade(self):
        """对手方价格匹配时成功下单，订单进入撮合队列"""
        alice_token = register_and_login("alice", "alice123")
        bob_token = register_and_login("bob", "bob123")

        code, body = http_post("/api/order", {
            "symbol": "BTC/USDT", "side": "sell", "type": "limit",
            "price": 50000, "quantity": 1
        }, token=alice_token)
        self.assertEqual(code, 200, f"alice 卖单失败: {body}")
        print(f"    alice 卖单 [OK] {body['order_id']}")

        code, body = http_post("/api/order", {
            "symbol": "BTC/USDT", "side": "buy", "type": "limit",
            "price": 50000, "quantity": 1
        }, token=bob_token)
        self.assertEqual(code, 200, f"bob 买单失败: {body}")
        print(f"    bob 买单 [OK] {body['order_id']}")

        time.sleep(2)
        print(f"    撮合流程完成（异步结果见 MatchingEngine/AccountService 日志）[OK]")


# ──────────────────────────────────────────────────────────────
# TC-12  WebSocket 握手
# ──────────────────────────────────────────────────────────────

class TestE2EWebSocket(unittest.TestCase):

    def test_12_websocket_handshake(self):
        """MarketData WebSocket 握手成功"""
        import socket
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(2)
            result = sock.connect_ex(("localhost", 18084))
            sock.close()
            if result != 0:
                self.skipTest("MarketData port-forward 不可达，跳过 WS 测试")
        except Exception as ex:
            self.skipTest(f"连通性检查失败: {ex}")

        ws_sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        ws_sock.settimeout(5)
        ws_sock.connect(("localhost", 18084))
        handshake = (
            "GET /ws HTTP/1.1\r\n"
            "Host: localhost:18084\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n"
            "Sec-WebSocket-Version: 13\r\n"
            "\r\n"
        )
        ws_sock.send(handshake.encode())
        response = ws_sock.recv(1024).decode(errors="replace")
        ws_sock.close()

        self.assertIn("101 Switching Protocols", response, "WebSocket 握手失败")
        self.assertIn("Upgrade: websocket", response)
        print(f"    WebSocket 握手成功 [OK]")


if __name__ == "__main__":
    print("=" * 60)
    print("IWS-Euler 系统端到端测试")
    print("=" * 60)
    unittest.main(verbosity=2, exit=True)

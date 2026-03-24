# WebApp

前端应用 + Nginx 反向代理，提供交易界面并将 API 请求路由到后端服务。

## 技术栈

- **React 19** + TypeScript
- **Vite 8** 构建工具
- **lightweight-charts** K 线图表库
- **Nginx** 反向代理（生产环境）

## 功能

- 用户注册 / 登录（JWT 存储在 localStorage）
- 下单界面（限价单、市价单）
- 实时 K 线图表（通过 WebSocket 接收行情）
- 成交记录展示

## Nginx 路由规则

| 请求路径 | 转发目标 |
|----------|----------|
| `/login` | `http://gateway:8081/login` |
| `/register` | `http://gateway:8081/register` |
| `/api/*` | `http://gateway:8081/api/*` |
| `/ws` | `ws://marketdata:8080/ws` |
| `/*` | React SPA（index.html） |

## 开发

```bash
cd WebApp
npm install
npm run dev    # 开发服务器（localhost:5173）
npm run build  # 生产构建
npm run lint   # ESLint 检查
```

## Docker 构建

```bash
docker build -t webapp:latest .
```

Dockerfile 使用多阶段构建：
1. `node:18-alpine` 阶段：`npm run build`
2. `nginx:alpine` 阶段：复制构建产物 + nginx.conf

## 访问

- 生产（K8s）：`http://localhost:30080`
- 开发模式：`http://localhost:5173`

## 端口

- 容器端口：80（Nginx）
- K8s NodePort：30080

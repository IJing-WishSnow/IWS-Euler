import { useState } from 'react'
import KLineChart from './components/KLineChart'
import OrderPanel from './components/OrderPanel'
import TradeHistory from './components/TradeHistory'
import './index.css'

const SYMBOL = 'BTC/USDT'

interface Trade {
  side: 'buy' | 'sell'
  price: number
  quantity: number
}

function LoginForm({ onLogin }: { onLogin: (token: string) => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!res.ok) throw new Error('用户名或密码错误')
      const data = await res.json()
      onLogin(data.token)
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const handleRegister = async () => {
    if (!username || !password) return
    try {
      const res = await fetch('/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!res.ok) throw new Error('注册失败')
      setError('注册成功，请登录')
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <div className="login-container">
      <div className="login-box">
        <h2 className="logo">IWS Exchange</h2>
        <form onSubmit={handleLogin} className="order-form">
          <label>用户名</label>
          <input value={username} onChange={e => setUsername(e.target.value)} placeholder="输入用户名" />
          <label>密码</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="输入密码（≥6位）" />
          <button type="submit" className="btn-buy">登录</button>
          <button type="button" className="btn-sell" onClick={handleRegister}>注册</button>
        </form>
        {error && <p className="form-error">{error}</p>}
      </div>
    </div>
  )
}

export default function App() {
  const [token, setToken] = useState<string>(localStorage.getItem('token') || '')
  const [trades, setTrades] = useState<Trade[]>([])

  const handleLogin = (t: string) => {
    localStorage.setItem('token', t)
    setToken(t)
  }

  const handleLogout = () => {
    localStorage.removeItem('token')
    setToken('')
  }

  if (!token) return <LoginForm onLogin={handleLogin} />

  return (
    <div className="app">
      <header className="header">
        <span className="logo">IWS</span>
        <span className="symbol">{SYMBOL}</span>
        <button type="button" onClick={handleLogout} className="logout-btn">退出</button>
      </header>
      <main className="main">
        <div className="chart-container">
          <KLineChart symbol={SYMBOL} />
        </div>
      </main>
      <aside className="sidebar">
        <OrderPanel
          symbol={SYMBOL}
          token={token}
          onOrderPlaced={order => setTrades(prev => [...prev, order])}
        />
        <TradeHistory trades={trades} />
      </aside>
    </div>
  )
}

import { useState } from 'react'

interface Order {
  id: string
  user_id: string
  symbol: string
  side: 'buy' | 'sell'
  type: 'limit'
  price: number
  quantity: number
  created_at: string
}

interface Props {
  symbol: string
  token: string
  onOrderPlaced?: (order: Order) => void
}

export default function OrderPanel({ symbol, token, onOrderPlaced }: Props) {
  const [side, setSide] = useState<'buy' | 'sell'>('buy')
  const [price, setPrice] = useState('')
  const [quantity, setQuantity] = useState('')
  const [status, setStatus] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!price || !quantity) return

    const order: Order = {
      id: `${side}-${Date.now()}`,
      user_id: 'user-1',
      symbol,
      side,
      type: 'limit',
      price: Math.round(parseFloat(price)),
      quantity: Math.round(parseFloat(quantity)),
      created_at: new Date().toISOString(),
    }

    try {
      const res = await fetch('/api/order', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          symbol: order.symbol,
          side: order.side,
          type: order.type,
          price: order.price,
          quantity: order.quantity,
        }),
      })

      if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || '下单失败')
      }

      setStatus(`已提交: ${side.toUpperCase()} ${quantity} @ ${price}`)
      onOrderPlaced?.(order)
      setPrice('')
      setQuantity('')
      setTimeout(() => setStatus(''), 3000)
    } catch (err) {
      setStatus(`失败: ${(err as Error).message}`)
    }
  }

  return (
    <div className="panel">
      <h3>下单</h3>
      <div className="order-tabs">
        <button
          className={`tab ${side === 'buy' ? 'tab-buy active' : 'tab-buy'}`}
          onClick={() => setSide('buy')}
        >买入</button>
        <button
          className={`tab ${side === 'sell' ? 'tab-sell active' : 'tab-sell'}`}
          onClick={() => setSide('sell')}
        >卖出</button>
      </div>
      <form onSubmit={handleSubmit} className="order-form">
        <label>价格 (USDT)</label>
        <input
          type="number"
          placeholder="输入价格"
          value={price}
          onChange={e => setPrice(e.target.value)}
        />
        <label>数量 (BTC)</label>
        <input
          type="number"
          placeholder="输入数量"
          value={quantity}
          onChange={e => setQuantity(e.target.value)}
        />
        <button type="submit" className={side === 'buy' ? 'btn-buy' : 'btn-sell'}>
          {side === 'buy' ? '买入 BTC' : '卖出 BTC'}
        </button>
      </form>
      {status && (
        <p style={{ marginTop: 8, fontSize: 12, color: side === 'buy' ? '#0ecb81' : '#f6465d' }}>
          {status}
        </p>
      )}
    </div>
  )
}

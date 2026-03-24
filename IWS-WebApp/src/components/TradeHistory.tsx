interface Trade {
  side: 'buy' | 'sell'
  price: number
  quantity: number
}

interface Props {
  trades: Trade[]
}

export default function TradeHistory({ trades }: Props) {
  if (!trades || trades.length === 0) {
    return (
      <div className="panel">
        <h3>成交记录</h3>
        <div style={{ color: '#848e9c', fontSize: 12, textAlign: 'center', padding: '16px 0' }}>
          暂无成交
        </div>
      </div>
    )
  }

  return (
    <div className="panel">
      <h3>成交记录</h3>
      <table className="positions-table">
        <thead>
          <tr>
            <th>方向</th>
            <th>价格</th>
            <th>数量</th>
          </tr>
        </thead>
        <tbody>
          {trades.slice().reverse().map((t, i) => (
            <tr key={i}>
              <td className={t.side === 'buy' ? 'text-buy' : 'text-sell'}>
                {t.side === 'buy' ? '买入' : '卖出'}
              </td>
              <td>{t.price?.toLocaleString()}</td>
              <td>{t.quantity}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

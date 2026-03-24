import { useEffect, useRef, useState } from 'react'
import { createChart, CandlestickSeries, IChartApi, ISeriesApi, CandlestickData, UTCTimestamp } from 'lightweight-charts'

type Period = '1m' | '5m' | '1h'

interface KLineMsg {
  symbol: string
  interval: string
  open: number
  high: number
  low: number
  close: number
  volume: number
  start_time: string
}

interface Props {
  symbol: string
}

export default function KLineChart({ symbol }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const [period, setPeriod] = useState<Period>('1m')
  const [connected, setConnected] = useState(false)

  // 初始化图表（只跑一次）
  useEffect(() => {
    if (!containerRef.current) return
    const chart = createChart(containerRef.current, {
      layout: { background: { color: '#1e2026' }, textColor: '#848e9c' },
      grid: { vertLines: { color: '#2b2f36' }, horzLines: { color: '#2b2f36' } },
      timeScale: { borderColor: '#2b2f36', timeVisible: true },
      rightPriceScale: { borderColor: '#2b2f36' },
      crosshair: { mode: 1 },
      width: containerRef.current.clientWidth,
      height: containerRef.current.clientHeight,
    })
    const series = chart.addSeries(CandlestickSeries, {
      upColor: '#0ecb81', downColor: '#f6465d',
      borderUpColor: '#0ecb81', borderDownColor: '#f6465d',
      wickUpColor: '#0ecb81', wickDownColor: '#f6465d',
    })
    chartRef.current = chart
    seriesRef.current = series

    const ro = new ResizeObserver(() => {
      if (!containerRef.current) return
      chart.applyOptions({
        width: containerRef.current.clientWidth,
        height: containerRef.current.clientHeight,
      })
    })
    ro.observe(containerRef.current)
    return () => { ro.disconnect(); chart.remove() }
  }, [])

  // WebSocket 连接（symbol 或 period 变化时重连）
  useEffect(() => {
    const series = seriesRef.current
    if (!series) return

    series.setData([]) // 清空旧数据
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws`)
    const klineMap = new Map<number, CandlestickData>()

    ws.onopen = () => {
      setConnected(true)
      // 发送订阅参数
      ws.send(JSON.stringify({ type: 'subscribe', symbol, period }))
    }

    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data)
      if (msg.type !== 'kline') return
      const k: KLineMsg = msg.data
      if (k.symbol !== symbol || k.interval !== period) return

      const bar: CandlestickData = {
        time: Math.floor(new Date(k.start_time).getTime() / 1000) as UTCTimestamp,
        open: k.open, high: k.high, low: k.low, close: k.close,
      }
      klineMap.set(bar.time as number, bar)

      // 按时间排序后全量更新（数据量小时足够高效）
      const sorted = Array.from(klineMap.values()).sort((a, b) =>
        (a.time as number) - (b.time as number)
      )
      series.setData(sorted)
      chartRef.current?.timeScale().scrollToRealTime()
    }

    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)

    return () => ws.close()
  }, [symbol, period])

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ display: 'flex', gap: 8, padding: '4px 8px', background: '#1e2026' }}>
        {(['1m', '5m', '1h'] as Period[]).map(p => (
          <button
            key={p}
            onClick={() => setPeriod(p)}
            style={{
              padding: '2px 10px', fontSize: 12, cursor: 'pointer',
              background: period === p ? '#f0b90b' : '#2b2f36',
              color: period === p ? '#1e2026' : '#848e9c',
              border: 'none', borderRadius: 4,
            }}
          >{p}</button>
        ))}
        <span style={{ marginLeft: 'auto', fontSize: 11, color: connected ? '#0ecb81' : '#f6465d' }}>
          {connected ? '● 已连接' : '○ 未连接'}
        </span>
      </div>
      <div ref={containerRef} style={{ flex: 1 }} />
    </div>
  )
}

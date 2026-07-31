import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import ReactECharts from 'echarts-for-react'
import { api } from '../api'
import type { History } from '../types'
import RoundsList from '../components/RoundsList'

interface Props {
  toast: (msg: string, type?: string) => void
}

export default function HistoryPage({ toast }: Props) {
  const { code = '' } = useParams()
  const roomCode = code.toUpperCase()
  const navigate = useNavigate()
  const [data, setData] = useState<History | null>(null)

  useEffect(() => {
    api<History>('GET', `/api/rooms/${roomCode}/history`)
      .then(setData)
      .catch((e) => {
        toast(e instanceof Error ? e.message : '加载失败', 'error')
        navigate(`/room/${roomCode}`)
      })
  }, [roomCode, navigate, toast])

  if (!data) {
    return (
      <section className="page active">
        <p className="placeholder">加载中…</p>
      </section>
    )
  }

  const date = data.room.createdAt
    ? new Date(data.room.createdAt).toLocaleDateString('zh-CN')
    : '-'
  const closedDate = data.closedAt
    ? new Date(data.closedAt).toLocaleDateString('zh-CN')
    : '进行中'

  const ranked = [...data.stats].sort((a, b) => a.rank - b.rank)

  const chartOption = {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 20, bottom: 40 },
    xAxis: {
      type: 'category',
      data: data.stats.map((s) => s.name),
      axisLabel: { color: '#8b9cb3' },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#8b9cb3' },
      splitLine: { lineStyle: { color: '#2d3f56' } },
    },
    series: [
      {
        type: 'bar',
        data: data.stats.map((s) => ({
          value: s.totalChange,
          itemStyle: { color: s.totalChange >= 0 ? '#22c55e' : '#ef4444' },
        })),
        barMaxWidth: 48,
      },
    ],
  }

  return (
    <section className="page active">
      <header className="room-header">
        <Link to={`/room/${roomCode}`} className="btn ghost">
          ← 返回控制台
        </Link>
        <h1>历史归档</h1>
      </header>

      <div className="history-summary">
        <div className="stat-box">
          <div className="value">{date}</div>
          <div className="label">创建日期</div>
        </div>
        <div className="stat-box">
          <div className="value">{data.totalRounds}</div>
          <div className="label">总局数</div>
        </div>
        <div className="stat-box">
          <div className="value">{closedDate}</div>
          <div className="label">关闭日期</div>
        </div>
      </div>

      <div className="panel full" style={{ marginBottom: 16 }}>
        <h2 style={{ marginBottom: 12 }}>成员排名</h2>
        <ul className="rank-list">
          {ranked.map((s) => {
            const sign = s.totalChange > 0 ? '+' : ''
            return (
              <li key={s.id}>
                <span className="rank">#{s.rank}</span>
                <span>{s.name}</span>
                <span>
                  {sign}
                  {s.totalChange}
                </span>
              </li>
            )
          })}
        </ul>
      </div>

      <div className="panel full" style={{ marginBottom: 16 }}>
        <h2 style={{ marginBottom: 12 }}>统计图</h2>
        <ReactECharts option={chartOption} style={{ height: 280 }} />
      </div>

      <div className="panel full">
        <h2 style={{ marginBottom: 12 }}>每局明细</h2>
        <RoundsList rounds={data.rounds} />
      </div>
    </section>
  )
}

import ReactECharts from 'echarts-for-react'
import type { Player } from '../types'

interface Props {
  players: Player[]
}

export default function StatsChart({ players }: Props) {
  const option = {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 20, bottom: 40 },
    xAxis: {
      type: 'category',
      data: players.map((p) => p.name),
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
        data: players.map((p) => {
          const v = p.currentChip - p.initialChip
          return {
            value: v,
            itemStyle: { color: v >= 0 ? '#22c55e' : '#ef4444' },
          }
        }),
        barMaxWidth: 48,
      },
    ],
  }

  return <ReactECharts option={option} style={{ height: 280 }} />
}

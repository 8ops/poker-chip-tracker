import type { Player } from '../types'

interface Props {
  players: Player[]
}

export default function PlayerList({ players }: Props) {
  if (players.length === 0) {
    return <p className="placeholder">暂无成员，请添加</p>
  }
  return (
    <div className="player-grid">
      {players.map((p) => {
        const change = p.currentChip - p.initialChip
        const cls = change > 0 ? 'positive' : change < 0 ? 'negative' : ''
        const sign = change > 0 ? '+' : ''
        return (
          <div className="player-card" key={p.id}>
            <div className="name">{p.name}</div>
            <div className="chip">{p.currentChip}</div>
            <div className={`change ${cls}`}>
              {sign}
              {change}
            </div>
          </div>
        )
      })}
    </div>
  )
}

import type { RoundView } from '../types'

interface Props {
  rounds: RoundView[]
}

export default function RoundsList({ rounds }: Props) {
  if (rounds.length === 0) {
    return <p className="placeholder">暂无牌局记录</p>
  }
  return (
    <>
      {rounds.map((round) => (
        <div className="round-item" key={round.id}>
          <div className="round-title">第 {round.roundNo} 局</div>
          <table>
            <tbody>
              {(round.records ?? []).length === 0 ? (
                <tr>
                  <td colSpan={2}>无变化</td>
                </tr>
              ) : (
                (round.records ?? []).map((rec) => {
                  const cls = rec.changeAmount > 0 ? 'positive' : 'negative'
                  const sign = rec.changeAmount > 0 ? '+' : ''
                  return (
                    <tr key={rec.id}>
                      <td>{rec.playerName}</td>
                      <td className={cls}>
                        {sign}
                        {rec.changeAmount}
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      ))}
    </>
  )
}

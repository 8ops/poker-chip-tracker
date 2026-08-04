import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useRoomSocket } from '../hooks/useRoomSocket'
import type { GameRound, Player, RoomDetail } from '../types'
import PlayerList from '../components/PlayerList'
import RoundsList from '../components/RoundsList'
import StatsChart from '../components/StatsChart'

interface Props {
  toast: (msg: string, type?: string) => void
}

export default function RoomPage({ toast }: Props) {
  const { code = '' } = useParams()
  const roomCode = code.toUpperCase()
  const navigate = useNavigate()

  const [detail, setDetail] = useState<RoomDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [playerName, setPlayerName] = useState('')
  const [initialChip, setInitialChip] = useState(1000)
  const [roundId, setRoundId] = useState<string | null>(null)
  const [changes, setChanges] = useState<Record<string, string>>({})

  const load = useCallback(async () => {
    try {
      const data = await api<RoomDetail>('POST', `/api/rooms/${roomCode}/join`)
      setDetail(data)
    } catch (e) {
      toast(e instanceof Error ? e.message : '加载失败', 'error')
      navigate('/')
    } finally {
      setLoading(false)
    }
  }, [roomCode, navigate, toast])

  const refresh = useCallback(async () => {
    try {
      const data = await api<RoomDetail>('GET', `/api/rooms/${roomCode}`)
      setDetail(data)
    } catch {
      /* 后台刷新失败时保留当前页面 */
    }
  }, [roomCode])

  useEffect(() => {
    setLoading(true)
    void load()
  }, [load])

  useRoomSocket(roomCode, () => {
    void refresh()
  })

  const readOnly = detail?.room.status === 'CLOSED'
  const players = detail?.players ?? []
  const rounds = detail?.rounds ?? []

  const sum = useMemo(
    () =>
      Object.values(changes).reduce((total, raw) => {
        const n = raw === '' || raw === '-' ? 0 : parseInt(raw, 10)
        return total + (Number.isNaN(n) ? 0 : n)
      }, 0),
    [changes],
  )

  const startRound = async () => {
    if (readOnly) return toast('房间已关闭', 'error')
    if (players.length === 0) return toast('请先添加成员', 'error')
    try {
      const round = await api<GameRound>('POST', `/api/rooms/${roomCode}/rounds`)
      setRoundId(round.id)
      const init: Record<string, string> = {}
      players.forEach((p) => {
        init[p.id] = '0'
      })
      setChanges(init)
    } catch (e) {
      toast(e instanceof Error ? e.message : '创建失败', 'error')
    }
  }

  const submitRound = async () => {
    if (!roundId) return
    const records = Object.entries(changes)
      .map(([playerId, raw]) => {
        const n = raw === '' || raw === '-' ? 0 : parseInt(raw, 10)
        return { playerId, changeAmount: Number.isNaN(n) ? 0 : n }
      })
      .filter((r) => r.changeAmount !== 0)
    try {
      await api('POST', `/api/rounds/${roundId}/records`, { records })
      setRoundId(null)
      setChanges({})
      toast('本局记录已保存', 'success')
      await load()
    } catch (e) {
      toast(e instanceof Error ? e.message : '提交失败', 'error')
    }
  }

  const addPlayer = async (e: FormEvent) => {
    e.preventDefault()
    try {
      await api('POST', '/api/players', {
        roomCode,
        name: playerName.trim(),
        initialChip: Number(initialChip) || 0,
      })
      setShowAdd(false)
      setPlayerName('')
      setInitialChip(1000)
      toast('成员已添加', 'success')
      await load()
    } catch (err) {
      toast(err instanceof Error ? err.message : '添加失败', 'error')
    }
  }

  const closeRoom = async () => {
    if (!confirm('确定关闭房间？关闭后不可再修改筹码，但可查看历史。')) return
    try {
      await api('POST', `/api/rooms/${roomCode}/close`)
      toast('房间已关闭', 'success')
      await load()
    } catch (e) {
      toast(e instanceof Error ? e.message : '关闭失败', 'error')
    }
  }

  if (loading || !detail) {
    return (
      <section className="page active">
        <p className="placeholder">加载中…</p>
      </section>
    )
  }

  return (
    <section className="page active">
      <header className="room-header">
        <div>
          <Link to="/" className="btn ghost">
            ← 返回
          </Link>
          <h1>{detail.room.name || '房间控制台'}</h1>
        </div>
        <div className="room-meta">
          <span className="badge">{detail.room.roomCode}</span>
          <span className={`badge status${readOnly ? ' closed' : ''}`}>
            {readOnly ? '已关闭' : '进行中'}
          </span>
        </div>
      </header>

      <div className="grid">
        <div className="panel">
          <div className="panel-head">
            <h2>成员列表</h2>
            {!readOnly && (
              <button type="button" className="btn sm" onClick={() => setShowAdd(true)}>
                添加成员
              </button>
            )}
          </div>
          <PlayerList players={players} />
        </div>

        <div className="panel">
          <div className="panel-head">
            <h2>新增一局</h2>
            {!readOnly && (
              <button type="button" className="btn sm primary" onClick={startRound}>
                开始新局
              </button>
            )}
          </div>
          {roundId ? (
            <div className="round-form">
              <p className="hint">
                每局所有成员筹码变化总和必须为 <strong>0</strong>
              </p>
              {players.map((p: Player) => (
                <div className="round-input-row" key={p.id}>
                  <label>{p.name}</label>
                  <input
                    type="number"
                    value={changes[p.id] ?? '0'}
                    onChange={(e) =>
                      setChanges((prev) => ({
                        ...prev,
                        [p.id]: e.target.value,
                      }))
                    }
                  />
                </div>
              ))}
              <div className="sum-row">
                <span>当前总和：</span>
                <span className={`sum-value ${sum === 0 ? 'balanced' : 'unbalanced'}`}>{sum}</span>
              </div>
              <button type="button" className="btn primary" disabled={sum !== 0} onClick={submitRound}>
                提交本局
              </button>
            </div>
          ) : (
            <p className="placeholder">点击「开始新局」记录筹码变化</p>
          )}
        </div>

        <div className="panel full">
          <h2>实时统计</h2>
          <StatsChart players={players} />
        </div>

        <div className="panel full">
          <div className="panel-head">
            <h2>本局明细</h2>
            <Link to={`/room/${roomCode}/history`} className="btn sm ghost">
              查看历史归档
            </Link>
          </div>
          <RoundsList rounds={rounds} />
        </div>
      </div>

      {!readOnly && (
        <footer className="room-footer">
          <button type="button" className="btn danger" onClick={closeRoom}>
            关闭房间
          </button>
        </footer>
      )}

      {showAdd && (
        <div className="modal-backdrop" onClick={() => setShowAdd(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()} role="dialog">
            <form onSubmit={addPlayer}>
              <h3>添加成员</h3>
              <label>
                成员名称
                <input
                  required
                  maxLength={20}
                  value={playerName}
                  onChange={(e) => setPlayerName(e.target.value)}
                />
              </label>
              <label>
                初始筹码
                <input
                  type="number"
                  required
                  value={initialChip}
                  onChange={(e) => setInitialChip(parseInt(e.target.value, 10) || 0)}
                />
              </label>
              <div className="dialog-actions">
                <button type="button" className="btn ghost" onClick={() => setShowAdd(false)}>
                  取消
                </button>
                <button type="submit" className="btn primary">
                  确认
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </section>
  )
}

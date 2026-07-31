import { FormEvent, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'

interface Props {
  toast: (msg: string, type?: string) => void
}

export default function HomePage({ toast }: Props) {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [joinCode, setJoinCode] = useState('')
  const [loading, setLoading] = useState(false)

  const createRoom = async () => {
    setLoading(true)
    try {
      const data = await api<{ roomCode: string }>('POST', '/api/rooms', { name: name.trim() })
      navigate(`/room/${data.roomCode}`)
    } catch (e) {
      toast(e instanceof Error ? e.message : '创建失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const joinRoom = (e?: FormEvent) => {
    e?.preventDefault()
    const code = joinCode.trim().toUpperCase()
    if (!code) {
      toast('请输入房间号', 'error')
      return
    }
    navigate(`/room/${code}`)
  }

  return (
    <section className="page active">
      <header className="hero">
        <h1>棋牌筹码记录器</h1>
        <p>多人实时记录 · 自动平衡校验 · 完整历史归档</p>
      </header>
      <div className="cards">
        <div className="card">
          <h2>创建房间</h2>
          <input
            type="text"
            placeholder="房间名称（可选）"
            maxLength={50}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <button className="btn primary" onClick={createRoom} disabled={loading}>
            创建房间
          </button>
        </div>
        <div className="card">
          <h2>加入房间</h2>
          <form onSubmit={joinRoom}>
            <input
              type="text"
              placeholder="输入房间号"
              maxLength={6}
              style={{ textTransform: 'uppercase' }}
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value)}
            />
            <button type="submit" className="btn">
              加入房间
            </button>
          </form>
        </div>
      </div>
    </section>
  )
}

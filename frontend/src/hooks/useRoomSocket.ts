import { useEffect, useRef } from 'react'

export function useRoomSocket(roomCode: string | undefined, onEvent: (event: string, data: unknown) => void) {
  const cb = useRef(onEvent)
  cb.current = onEvent

  useEffect(() => {
    if (!roomCode) return
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    let ws: WebSocket | null = null
    let timer: number | undefined
    let closed = false

    const connect = () => {
      ws = new WebSocket(`${proto}://${location.host}/ws/rooms/${roomCode}`)
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data) as { event: string; data: unknown }
          cb.current(msg.event, msg.data)
        } catch {
          /* ignore */
        }
      }
      ws.onclose = () => {
        if (!closed) timer = window.setTimeout(connect, 3000)
      }
    }

    connect()
    return () => {
      closed = true
      if (timer) clearTimeout(timer)
      ws?.close()
    }
  }, [roomCode])
}

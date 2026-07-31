import { useState, useCallback } from 'react'
import './Toast.css'

export function useToast() {
  const [msg, setMsg] = useState<{ text: string; type: string } | null>(null)

  const toast = useCallback((text: string, type = '') => {
    setMsg({ text, type })
    window.setTimeout(() => setMsg(null), 3000)
  }, [])

  const Toast = () =>
    msg ? <div className={`toast ${msg.type}`}>{msg.text}</div> : null

  return { toast, Toast }
}

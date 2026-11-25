import { useState, useEffect, useRef, useCallback } from 'react'

export interface ParsedPayload {
  output?: string
  isError?: boolean
  message?: string
  [key: string]: unknown
}

export interface WSMessage {
  _seq: number          // monotonic counter — forces new object reference every time
  type: 'output' | 'error' | 'event'
  payload: ParsedPayload
}

interface UseBeaconSocketReturn {
  connected: boolean
  lastMessage: WSMessage | null
  wsSend: (cmd: string, beaconId: string) => void
}

export function useBeaconSocket(token: string | null, labId: string): UseBeaconSocketReturn {
  const [connected, setConnected]       = useState(false)
  const [lastMessage, setLastMessage]   = useState<WSMessage | null>(null)
  const wsRef                           = useRef<WebSocket | null>(null)
  const seqRef                          = useRef(0)

  useEffect(() => {
    if (!token) return

    let cancelled = false
    let retryTimer: ReturnType<typeof setTimeout> | null = null

    function connect() {
      if (cancelled) return

      const url = `ws://localhost:8080/ws?token=${encodeURIComponent(token!)}&lab_id=${encodeURIComponent(labId)}`
      const ws  = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        if (cancelled) { ws.close(); return }
        console.log('[useBeaconSocket] connected')
        setConnected(true)
      }

      ws.onclose = () => {
        setConnected(false)
        wsRef.current = null
        if (!cancelled) {
          console.log('[useBeaconSocket] closed — retrying in 2s')
          retryTimer = setTimeout(connect, 2000)
        }
      }

      ws.onerror = (e) => {
        console.error('[useBeaconSocket] WS error', e)
        // onclose fires after onerror, retry handled there
      }

      ws.onmessage = (ev: MessageEvent) => {
        let raw: { type: string; payload?: unknown; event_type?: string } | null = null
        try { raw = JSON.parse(ev.data as string) } catch { return }

        if (!raw) return

        seqRef.current += 1
        const seq = seqRef.current

        if (raw.type === 'response') {
          const text = typeof raw.payload === 'string' ? raw.payload : JSON.stringify(raw.payload)
          setLastMessage({ _seq: seq, type: 'output', payload: { output: text, isError: false } })
        } else if (raw.type === 'event') {
          const p = (raw.payload ?? {}) as ParsedPayload
          setLastMessage({ _seq: seq, type: 'event', payload: p })
        } else if (raw.type === 'error') {
          const msg = typeof raw.payload === 'string' ? raw.payload : (raw.payload as ParsedPayload)?.message ?? 'unknown'
          setLastMessage({ _seq: seq, type: 'error', payload: { message: msg } })
        }
      }
    }

    connect()

    return () => {
      cancelled = true
      if (retryTimer) clearTimeout(retryTimer)
      if (wsRef.current) {
        wsRef.current.onclose = null   // prevent retry on intentional close
        wsRef.current.close()
        wsRef.current = null
      }
      setConnected(false)
    }
  }, [token, labId])

  const wsSend = useCallback((cmd: string, beaconId: string) => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.warn('[useBeaconSocket] WS not open, dropping command:', cmd)
      return
    }
    ws.send(JSON.stringify({ type: 'command', payload: cmd, beacon_id: beaconId }))
  }, [])

  return { connected, lastMessage, wsSend }
}

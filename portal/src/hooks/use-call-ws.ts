import { useEffect, useRef, useState, useCallback } from 'react'

export type WsStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

export interface CallEvent {
  type: 'call_started' | 'call_answered' | 'call_ended' | 'call_failed'
  call_id: string
  caller?: string
  callee?: string
  status?: string
  duration?: number
  timestamp: string
}

interface UseCallWsOptions {
  onEvent?: (event: CallEvent) => void
  enabled?: boolean
}

const MAX_RECONNECT_DELAY = 30_000
const BASE_RECONNECT_DELAY = 1_000

export function useCallWs(options: UseCallWsOptions = {}) {
  const { onEvent, enabled = true } = options
  const [status, setStatus] = useState<WsStatus>('disconnected')
  const [lastEvent, setLastEvent] = useState<CallEvent | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const onEventRef = useRef(onEvent)
  onEventRef.current = onEvent

  const connect = useCallback(() => {
    if (!enabled) return

    const token = localStorage.getItem('bos3000-portal-token')
    if (!token) {
      setStatus('disconnected')
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws/calls?token=${encodeURIComponent(token)}`

    setStatus('connecting')
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setStatus('connected')
      reconnectAttemptRef.current = 0
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as CallEvent
        setLastEvent(data)
        onEventRef.current?.(data)
      } catch {
        // ignore malformed messages
      }
    }

    ws.onerror = () => {
      setStatus('error')
    }

    ws.onclose = (event) => {
      wsRef.current = null
      setStatus('disconnected')

      // Don't reconnect on clean close or auth failure
      if (event.code === 1000 || event.code === 4401) return

      // Exponential backoff reconnect
      const delay = Math.min(
        BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttemptRef.current),
        MAX_RECONNECT_DELAY,
      )
      reconnectAttemptRef.current += 1
      reconnectTimerRef.current = setTimeout(connect, delay)
    }
  }, [enabled])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close(1000)
        wsRef.current = null
      }
    }
  }, [connect])

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close(1000)
      wsRef.current = null
    }
    setStatus('disconnected')
  }, [])

  return { status, lastEvent, disconnect }
}

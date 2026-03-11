import { useEffect, useRef, useState, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'

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

const MAX_RECONNECT_DELAY = 30_000
const BASE_RECONNECT_DELAY = 1_000

export function useCallWs(enabled = true) {
  const [status, setStatus] = useState<WsStatus>('disconnected')
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const queryClient = useQueryClient()

  const connect = useCallback(() => {
    if (!enabled) return

    // Admin uses session cookie; extract JWT for WS auth
    const token = document.cookie
      .split('; ')
      .find((c) => c.startsWith('session='))
      ?.split('=')[1]

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = token
      ? `${protocol}//${window.location.host}/ws/calls?token=${encodeURIComponent(token)}`
      : `${protocol}//${window.location.host}/ws/calls`

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
        // Update active calls cache
        if (data.type === 'call_started' || data.type === 'call_answered') {
          queryClient.invalidateQueries({ queryKey: ['admin', 'active-calls'] })
        }
        if (data.type === 'call_ended' || data.type === 'call_failed') {
          queryClient.invalidateQueries({ queryKey: ['admin', 'active-calls'] })
          queryClient.invalidateQueries({ queryKey: ['admin', 'dashboard'] })
        }
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

      if (event.code === 1000 || event.code === 4401) return

      const delay = Math.min(
        BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttemptRef.current),
        MAX_RECONNECT_DELAY,
      )
      reconnectAttemptRef.current += 1
      reconnectTimerRef.current = setTimeout(connect, delay)
    }
  }, [enabled, queryClient])

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

  return { status }
}

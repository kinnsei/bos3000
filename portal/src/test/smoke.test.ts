import { describe, it, expect } from 'vitest'
import { createMockApiClient } from './mocks/api'
import { MockWebSocket } from './mocks/websocket'

describe('Test infrastructure', () => {
  it('mock API client is callable', () => {
    const api = createMockApiClient()
    expect(api.auth.login).toBeDefined()
    expect(api.callback.initiate).toBeDefined()
    expect(api.billing.getBalance).toBeDefined()
  })

  it('mock WebSocket works', () => {
    const ws = new MockWebSocket('ws://localhost/ws/calls')
    expect(ws.url).toBe('ws://localhost/ws/calls')
    expect(ws.readyState).toBe(MockWebSocket.CONNECTING)

    ws.simulateOpen()
    expect(ws.readyState).toBe(MockWebSocket.OPEN)

    ws.simulateClose()
    expect(ws.readyState).toBe(MockWebSocket.CLOSED)
  })
})

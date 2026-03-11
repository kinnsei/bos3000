import { describe, it, expect } from 'vitest'
import { createMockApiClient } from './mocks/api'

describe('smoke test', () => {
  it('creates mock API client', () => {
    const client = createMockApiClient()
    expect(client).toBeDefined()
    expect(client.auth).toBeDefined()
    expect(client.billing).toBeDefined()
    expect(client.routing).toBeDefined()
    expect(client.callback).toBeDefined()
    expect(client.compliance).toBeDefined()
  })

  it('mock API methods return expected values', async () => {
    const client = createMockApiClient()
    const overview = await client.billing.GetOverview()
    expect(overview).toHaveProperty('concurrentCalls')
    expect(overview).toHaveProperty('bridgeSuccessRate')
  })
})

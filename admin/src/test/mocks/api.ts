import { vi } from 'vitest'

export function createMockApiClient() {
  return {
    auth: {
      Login: vi.fn().mockResolvedValue({ token: 'mock-token' }),
      Me: vi.fn().mockResolvedValue({ id: 'admin-1', username: 'admin', role: 'admin' }),
      ListUsers: vi.fn().mockResolvedValue({ users: [], total: 0 }),
      CreateUser: vi.fn().mockResolvedValue({ id: 'user-1' }),
      FreezeUser: vi.fn().mockResolvedValue({}),
      UnfreezeUser: vi.fn().mockResolvedValue({}),
    },
    billing: {
      GetOverview: vi.fn().mockResolvedValue({
        concurrentCalls: 0,
        todayRevenue: 0,
        todayWastage: 0,
        bridgeSuccessRate: 0,
      }),
      TopUp: vi.fn().mockResolvedValue({}),
      Deduct: vi.fn().mockResolvedValue({}),
      ListTransactions: vi.fn().mockResolvedValue({ transactions: [], total: 0 }),
      ListRatePlans: vi.fn().mockResolvedValue({ plans: [] }),
      CreateRatePlan: vi.fn().mockResolvedValue({ id: 'plan-1' }),
    },
    routing: {
      ListGateways: vi.fn().mockResolvedValue({ gateways: [] }),
      ToggleGateway: vi.fn().mockResolvedValue({}),
      TestCall: vi.fn().mockResolvedValue({}),
      ListDIDs: vi.fn().mockResolvedValue({ dids: [], total: 0 }),
      ImportDIDs: vi.fn().mockResolvedValue({ imported: 0 }),
      HealthCheck: vi.fn().mockResolvedValue({ status: 'ok' }),
    },
    callback: {
      ListCDRs: vi.fn().mockResolvedValue({ cdrs: [], total: 0 }),
      ListActiveCalls: vi.fn().mockResolvedValue({ calls: [] }),
      HangupCall: vi.fn().mockResolvedValue({}),
    },
    compliance: {
      ListBlacklist: vi.fn().mockResolvedValue({ entries: [], total: 0 }),
      AddBlacklist: vi.fn().mockResolvedValue({}),
      RemoveBlacklist: vi.fn().mockResolvedValue({}),
      ListAuditLogs: vi.fn().mockResolvedValue({ logs: [], total: 0 }),
    },
  }
}

export type MockApiClient = ReturnType<typeof createMockApiClient>

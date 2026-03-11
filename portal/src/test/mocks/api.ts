import { vi } from 'vitest'

export function createMockApiClient() {
  return {
    auth: {
      login: vi.fn(),
      me: vi.fn(),
      updateProfile: vi.fn(),
      changePassword: vi.fn(),
      regenerateApiKey: vi.fn(),
    },
    callback: {
      initiate: vi.fn(),
      listCDRs: vi.fn(),
      listActiveCalls: vi.fn(),
      hangupCall: vi.fn(),
    },
    billing: {
      getBalance: vi.fn(),
      listTransactions: vi.fn(),
      getUsageSummary: vi.fn(),
    },
    wastage: {
      getSummary: vi.fn(),
      getTrend: vi.fn(),
      getDistribution: vi.fn(),
    },
  }
}

export type MockApiClient = ReturnType<typeof createMockApiClient>

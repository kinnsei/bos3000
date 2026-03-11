// Portal API client - customer-facing endpoints
// Uses Bearer token auth (stored in localStorage)

function getToken(): string | null {
  return localStorage.getItem('bos3000-portal-token')
}

export function setToken(token: string) {
  localStorage.setItem('bos3000-portal-token', token)
}

export function clearToken() {
  localStorage.removeItem('bos3000-portal-token')
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options?.headers as Record<string, string>),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const resp = await fetch(path, {
    ...options,
    headers,
  })

  if (!resp.ok) {
    const body = await resp.json().catch(() => ({ code: 'UNKNOWN', message: resp.statusText }))
    throw body
  }

  // Handle 204 No Content
  if (resp.status === 204) return undefined as T

  return resp.json()
}

// --- Types ---

export interface LoginParams {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
}

export interface PortalUser {
  id: string
  username: string
  email: string
  phone: string
  status: string
  balance: number
  credit_limit: number
  concurrent_limit: number
  daily_limit: number
  api_key: string
  ip_whitelist: string[]
  created_at: string
  updated_at: string
}

export interface CDR {
  id: string
  call_id: string
  caller: string
  callee: string
  status: string
  duration: number
  cost: number
  started_at: string
  ended_at: string
}

export interface PaginatedParams {
  page: number
  limit: number
}

export interface Transaction {
  id: string
  type: string
  amount: number
  balance_after: number
  description: string
  created_at: string
}

export interface CallbackInitParams {
  caller: string
  callee: string
}

export interface CallbackInitResponse {
  call_id: string
  status: string
}

export interface UsageSummary {
  today_calls: number
  today_duration: number
  today_cost: number
  balance: number
  concurrent_active: number
  concurrent_limit: number
}

export interface WastageSummary {
  today_wastage_cost: number
  today_wastage_rate: number
  top_failure_reason: string
}

// --- Service classes ---

class AuthService {
  async login(params: LoginParams): Promise<LoginResponse> {
    return request('/api/auth.Login', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async me(): Promise<PortalUser> {
    return request('/api/auth.Me')
  }

  async updateProfile(params: { phone?: string }): Promise<void> {
    return request('/api/auth.UpdateProfile', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async changePassword(params: { old_password: string; new_password: string }): Promise<void> {
    return request('/api/auth.ChangePassword', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async regenerateApiKey(): Promise<{ api_key: string }> {
    return request('/api/auth.RegenerateApiKey', {
      method: 'POST',
      body: JSON.stringify({}),
    })
  }
}

class CallbackService {
  async initiate(params: CallbackInitParams): Promise<CallbackInitResponse> {
    return request('/api/callback.Initiate', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async listCDRs(params: PaginatedParams & { start_date?: string; end_date?: string }): Promise<{ cdrs: CDR[]; total: number }> {
    return request('/api/callback.ListCDRs', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async listActiveCalls(): Promise<{ calls: CDR[] }> {
    return request('/api/callback.ListActiveCalls')
  }

  async hangupCall(params: { call_id: string }): Promise<void> {
    return request('/api/callback.HangupCall', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }
}

class BillingService {
  async getBalance(): Promise<{ balance: number; credit_limit: number }> {
    return request('/api/billing.GetBalance')
  }

  async listTransactions(params: PaginatedParams): Promise<{ transactions: Transaction[]; total: number }> {
    return request('/api/billing.ListTransactions', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async getUsageSummary(): Promise<UsageSummary> {
    return request('/api/billing.GetUsageSummary')
  }
}

export default class PortalClient {
  auth = new AuthService()
  callback = new CallbackService()
  billing = new BillingService()
}

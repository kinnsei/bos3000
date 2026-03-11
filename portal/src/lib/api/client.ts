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

  if (resp.status === 204) return undefined as T
  return resp.json()
}

function qs(params: Record<string, unknown>): string {
  const parts = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
  return parts.length ? '?' + parts.join('&') : ''
}

// --- Types ---

export interface LoginParams {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: string
}

export interface PortalUser {
  user_id: number
  username: string
  email: string
  role: string
  status: string
}

export interface CDR {
  id: string
  call_id: string
  a_number: string
  b_number: string
  status: string
  duration: number
  cost: number
  created_at: string
  ended_at: string
}

export interface PaginatedParams {
  page: number
  limit: number
}

export interface Transaction {
  id: number
  type: string
  amount: number
  balance_after: number
  description: string
  created_at: string
}

export interface CallbackInitParams {
  a_number: string
  b_number: string
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

export interface RateQueryResult {
  plan_name: string
  rate_a: number
  rate_b: number
  billing_unit: number
  effective_date: string
}

export interface WebhookConfig {
  webhook_url: string
  webhook_secret: string
}

export interface WebhookDelivery {
  id: string
  event: string
  url: string
  status_code: number
  success: boolean
  created_at: string
}

export interface WebhookTestResult {
  success: boolean
  status_code: number
  message: string
}

// --- Service classes ---

class AuthService {
  async login(params: LoginParams): Promise<LoginResponse> {
    return request('/api/auth/client/login', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async me(): Promise<PortalUser> {
    return request('/api/auth/me')
  }

  async updateProfile(params: { username?: string; webhook_url?: string }): Promise<void> {
    return request('/api/auth/profile', {
      method: 'PUT',
      body: JSON.stringify(params),
    })
  }

  async changePassword(params: { current_password: string; new_password: string }): Promise<void> {
    return request('/api/auth/profile/password', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async regenerateApiKey(): Promise<{ api_key: string }> {
    return request('/api/auth/api-keys', {
      method: 'POST',
      body: JSON.stringify({}),
    })
  }

  async addIp(params: { ip: string }): Promise<void> {
    // TODO: Need to know API key ID - for now mock
    return undefined as void
  }

  async removeIp(params: { ip: string }): Promise<void> {
    // TODO: Need to know API key ID - for now mock
    return undefined as void
  }
}

class CallbackService {
  async initiate(params: CallbackInitParams): Promise<CallbackInitResponse> {
    return request('/api/callbacks', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async listCDRs(params: PaginatedParams & { start_date?: string; end_date?: string; status?: string; search?: string }): Promise<{ cdrs: CDR[]; total: number }> {
    const resp = await request<{ callbacks: CDR[]; total: number }>(
      `/api/callbacks${qs({
        page: params.page,
        page_size: params.limit,
        date_from: params.start_date,
        date_to: params.end_date,
        status: params.status,
      })}`
    )
    return { cdrs: resp.callbacks || [], total: resp.total || 0 }
  }

  async listActiveCalls(): Promise<{ calls: CDR[] }> {
    const resp = await request<{ callbacks: CDR[]; total: number }>(
      `/api/callbacks${qs({ page: 1, page_size: 100, status: 'in_progress' })}`
    )
    return { calls: resp.callbacks || [] }
  }

  async hangupCall(params: { call_id: string }): Promise<void> {
    return request(`/api/callbacks/${params.call_id}/hangup`, {
      method: 'POST',
    })
  }
}

class BillingService {
  async getBalance(): Promise<{ balance: number; credit_limit: number }> {
    // Uses the auth user's own account
    return request('/api/auth/me').then(async (user: any) => {
      const account = await request<any>(`/api/billing/accounts/${user.user_id}`)
      return { balance: account.balance, credit_limit: account.credit_limit }
    })
  }

  async listTransactions(params: PaginatedParams): Promise<{ transactions: Transaction[]; total: number }> {
    // Get user ID first, then fetch transactions
    const user = await request<any>('/api/auth/me')
    const resp = await request<any>(`/api/billing/accounts/${user.user_id}/transactions${qs({ page: params.page, page_size: params.limit })}`)
    return { transactions: resp.transactions || [], total: resp.total || 0 }
  }

  async getUsageSummary(): Promise<UsageSummary> {
    // TODO: No real endpoint yet, return mock
    return {
      today_calls: 0,
      today_duration: 0,
      today_cost: 0,
      balance: 0,
      concurrent_active: 0,
      concurrent_limit: 10,
    }
  }

  async queryRate(prefix: string): Promise<RateQueryResult | null> {
    // TODO: resolve-rate is a private endpoint
    return null
  }
}

class WebhookService {
  async getConfig(): Promise<WebhookConfig> {
    // TODO: No dedicated GET endpoint yet
    return { webhook_url: '', webhook_secret: '' }
  }

  async saveConfig(params: { webhook_url: string }): Promise<void> {
    return request('/api/webhooks/config', {
      method: 'PUT',
      body: JSON.stringify(params),
    })
  }

  async test(): Promise<WebhookTestResult> {
    // TODO: No test endpoint yet
    return { success: false, status_code: 0, message: 'Not implemented yet' }
  }

  async listDeliveries(): Promise<{ deliveries: WebhookDelivery[] }> {
    const resp = await request<{ deliveries: WebhookDelivery[] }>('/api/webhooks/deliveries')
    return { deliveries: resp.deliveries || [] }
  }
}

export default class PortalClient {
  auth = new AuthService()
  callback = new CallbackService()
  billing = new BillingService()
  webhook = new WebhookService()
}

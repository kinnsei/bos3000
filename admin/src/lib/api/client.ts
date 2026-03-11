// Admin API client - maps to actual Encore backend REST endpoints.
// All paths prefixed with /api/ which Vite proxy strips to /.

export interface LoginParams {
  email: string
  password: string
}

export interface LoginResponse {
  message: string
  expires_at: string
}

export interface User {
  id: number
  username: string
  email: string
  role: string
  status: string
  balance: number
  credit_limit: number
  max_concurrent: number
  daily_limit: number
  created_at: string
  updated_at: string
}

export interface ListUsersParams {
  page: number
  limit: number
  search?: string
  status?: string
}

export interface ListUsersResponse {
  users: User[]
  total: number
  page: number
  limit: number
}

export interface OverviewResponse {
  concurrent_calls: number
  today_revenue: number
  today_wastage: number
  bridge_success_rate: number
  alerts: Alert[]
}

export interface Alert {
  type: 'bridge_rate_low' | 'balance_low' | 'gateway_down' | 'wastage_high'
  message: string
  severity: 'warning' | 'critical'
}

export interface CustomerDetail {
  id: number
  username: string
  email: string
  role: string
  status: string
  balance: number
  credit_limit: number
  max_concurrent: number
  daily_limit: number
  rate_plan_id: number | null
  a_leg_rate: number | null
  b_leg_rate: number | null
  webhook_url: string
  created_at: string
  updated_at: string
}

export interface CreateUserParams {
  username: string
  email: string
  password: string
  credit_limit?: number
  max_concurrent?: number
  daily_limit?: number
  rate_plan_id?: number | null
  a_leg_rate?: number | null
  b_leg_rate?: number | null
  webhook_url?: string
}

export interface Gateway {
  id: number
  name: string
  type: string
  host: string
  port: number
  enabled: boolean
  weight: number
  max_concurrent: number
  health_status: string
  created_at: string
}

export interface CreateGatewayParams {
  name: string
  type: string
  host: string
  port: number
  weight: number
  max_concurrent: number
  enabled: boolean
}

export interface TestOriginateResult {
  success: boolean
  message: string
  duration_ms: number
}

export interface CDR {
  call_id: string
  user_id: number
  a_number: string
  b_number: string
  status: string
  bridge_duration_ms: number
  total_cost: number
  failure_reason?: string
  created_at: string
  // computed helpers
  caller: string
  callee: string
  duration: number
  cost: number
}

export interface Transaction {
  id: number
  user_id: number
  type: string
  amount: number
  balance_after: number
  description: string
  created_at: string
}

export interface RatePlan {
  id: number
  name: string
  mode: string
  uniform_a_rate: number
  uniform_b_rate: number
  description?: string
  created_at: string
  updated_at: string
}

export interface DIDNumber {
  id: number
  number: string
  status: string
  assigned_user_id?: number | null
  created_at: string
}

export interface BlacklistEntry {
  id: number
  number: string
  user_id: number | null
  reason: string
  created_by: number
}

export interface AuditLog {
  id: number
  action: string
  actor_id: number
  actor_name?: string
  target_type: string
  target_id: string
  details: string
  ip_address?: string
  created_at: string
}

export interface PaginatedParams {
  page: number
  limit: number
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(path, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(options?.headers as Record<string, string>),
    },
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

class AuthService {
  async Login(params: LoginParams): Promise<LoginResponse> {
    // Admin login is a raw endpoint that sets a session cookie
    const resp = await fetch('/api/auth/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async Me(): Promise<User> {
    return request('/api/auth/me')
  }

  async ListUsers(params: ListUsersParams): Promise<ListUsersResponse> {
    return request(`/api/auth/admin/users${qs(params)}`)
  }

  async GetUser(params: { user_id: string }): Promise<CustomerDetail> {
    return request(`/api/auth/admin/users/${params.user_id}`)
  }

  async CreateUser(params: CreateUserParams): Promise<{ id: number; email: string; username: string }> {
    return request('/api/auth/admin/users', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async FreezeUser(params: { user_id: string }): Promise<void> {
    return request(`/api/auth/admin/users/${params.user_id}/freeze`, {
      method: 'POST',
    })
  }

  async UnfreezeUser(params: { user_id: string }): Promise<void> {
    return request(`/api/auth/admin/users/${params.user_id}/unfreeze`, {
      method: 'POST',
    })
  }

  async RegenerateApiKey(params: { user_id: string }): Promise<{ api_key: string }> {
    return request(`/api/auth/api-keys`, {
      method: 'POST',
      body: JSON.stringify({}),
    })
  }
}

class BillingService {
  async GetOverview(): Promise<OverviewResponse> {
    // TODO: No real endpoint yet, return mock data
    return {
      concurrent_calls: 0,
      today_revenue: 0,
      today_wastage: 0,
      bridge_success_rate: 100,
      alerts: [],
    }
  }

  async TopUp(params: { user_id: string; amount: number }): Promise<void> {
    return request(`/api/billing/accounts/${params.user_id}/topup`, {
      method: 'POST',
      body: JSON.stringify({ amount: params.amount }),
    })
  }

  async Deduct(params: { user_id: string; amount: number; reason: string }): Promise<void> {
    return request(`/api/billing/accounts/${params.user_id}/deduct`, {
      method: 'POST',
      body: JSON.stringify({ amount: params.amount, reason: params.reason }),
    })
  }

  async ListTransactions(params: PaginatedParams & { user_id?: string }): Promise<{ transactions: Transaction[]; total: number }> {
    const userId = params.user_id || '0'
    return request(`/api/billing/accounts/${userId}/transactions${qs({ page: params.page, page_size: params.limit })}`)
  }

  async ListRatePlans(): Promise<{ plans: RatePlan[] }> {
    return request('/api/billing/rate-plans')
  }

  async CreateRatePlan(params: Omit<RatePlan, 'id'>): Promise<{ id: string }> {
    return request('/api/billing/rate-plans', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }
}

class RoutingService {
  async ListGateways(): Promise<{ gateways: Gateway[] }> {
    return request('/api/routing/gateways')
  }

  async ToggleGateway(params: { gateway_id: string; enabled: boolean }): Promise<void> {
    return request(`/api/routing/gateways/${params.gateway_id}/toggle`, {
      method: 'POST',
      body: JSON.stringify({ enabled: params.enabled }),
    })
  }

  async ListDIDs(params: PaginatedParams): Promise<{ dids: DIDNumber[]; total: number }> {
    const resp = await request<{ dids: DIDNumber[]; total_count: number }>(`/api/routing/dids${qs({ page: params.page, page_size: params.limit })}`)
    return { dids: resp.dids || [], total: resp.total_count || 0 }
  }

  async ImportDIDs(params: { numbers: string[] }): Promise<{ imported: number }> {
    return request('/api/routing/did-import', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async HealthCheck(): Promise<{ status: string; gateways: Array<{ id: string; status: string }> }> {
    return request('/api/routing/health-check', { method: 'POST' })
  }

  async CreateGateway(params: CreateGatewayParams): Promise<{ id: string }> {
    return request('/api/routing/gateways', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async UpdateGateway(params: { gateway_id: string } & Partial<CreateGatewayParams>): Promise<void> {
    const { gateway_id, ...body } = params
    return request(`/api/routing/gateways/${gateway_id}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    })
  }

  async TestOriginate(params: { gateway_id: string; phone_number: string }): Promise<TestOriginateResult> {
    // TODO: No real endpoint yet
    return { success: false, message: 'Test originate not yet implemented', duration_ms: 0 }
  }
}

class CallbackService {
  async ListCDRs(params: PaginatedParams & { start_date?: string; end_date?: string; status?: string }): Promise<{ cdrs: CDR[]; total: number }> {
    const resp = await request<{ items: Array<Omit<CDR, 'caller' | 'callee' | 'duration' | 'cost'>>; total: number }>(
      `/api/callbacks${qs({ page: params.page, page_size: params.limit, date_from: params.start_date, date_to: params.end_date, status: params.status })}`
    )
    const cdrs = (resp.items || []).map((item) => ({
      ...item,
      caller: item.a_number,
      callee: item.b_number,
      duration: Math.round(item.bridge_duration_ms / 1000),
      cost: item.total_cost / 100,
    }))
    return { cdrs, total: resp.total || 0 }
  }

  async ListActiveCalls(): Promise<{ calls: CDR[] }> {
    const resp = await request<{ items: Array<Omit<CDR, 'caller' | 'callee' | 'duration' | 'cost'>>; total: number }>(
      `/api/callbacks${qs({ page: 1, page_size: 100, status: 'in_progress' })}`
    )
    const calls = (resp.items || []).map((item) => ({
      ...item,
      caller: item.a_number,
      callee: item.b_number,
      duration: Math.round(item.bridge_duration_ms / 1000),
      cost: item.total_cost / 100,
    }))
    return { calls }
  }

  async HangupCall(params: { call_id: string }): Promise<void> {
    return request(`/api/callbacks/${params.call_id}/hangup`, {
      method: 'POST',
    })
  }
}

class ComplianceService {
  async ListBlacklist(params: PaginatedParams): Promise<{ entries: BlacklistEntry[]; total: number }> {
    return request(`/api/compliance/blacklist${qs(params)}`)
  }

  async AddBlacklist(params: { number: string; reason: string }): Promise<void> {
    return request('/api/compliance/blacklist', {
      method: 'POST',
      body: JSON.stringify(params),
    })
  }

  async RemoveBlacklist(params: { id: string }): Promise<void> {
    return request(`/api/compliance/blacklist/${params.id}`, {
      method: 'DELETE',
    })
  }

  async ListAuditLogs(params: PaginatedParams): Promise<{ logs: AuditLog[]; total: number }> {
    return request(`/api/compliance/audit-logs${qs(params)}`)
  }
}

export default class Client {
  auth = new AuthService()
  billing = new BillingService()
  routing = new RoutingService()
  callback = new CallbackService()
  compliance = new ComplianceService()
}

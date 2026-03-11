// Placeholder for Encore-generated client.
// Replace with: encore gen client bos3000 --lang=typescript --output=./src/lib/api/client.ts --env=local
//
// This stub provides the same interface shape so hooks compile before the backend is available.

export interface LoginParams {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
}

export interface User {
  id: string
  username: string
  email: string
  role: string
  status: string
  created_at: string
}

export interface ListUsersParams {
  page: number
  limit: number
  search?: string
}

export interface ListUsersResponse {
  users: User[]
  total: number
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
  id: string
  username: string
  email: string
  phone: string
  role: string
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

export interface CreateUserParams {
  username: string
  email: string
  password: string
  phone?: string
  concurrent_limit: number
  daily_limit: number
  initial_balance: number
}

export interface Gateway {
  id: string
  name: string
  type: 'a_leg' | 'b_leg'
  host: string
  port: number
  status: 'up' | 'down' | 'disabled'
  weight: number
  prefix: string
  failover_gateway_id: string
  concurrent_calls: number
  max_concurrent: number
}

export interface CreateGatewayParams {
  name: string
  type: 'a_leg' | 'b_leg'
  host: string
  port: number
  weight: number
  prefix: string
  failover_gateway_id: string
  max_concurrent: number
  enabled: boolean
}

export interface TestOriginateResult {
  success: boolean
  message: string
  duration_ms: number
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

export interface Transaction {
  id: string
  user_id: string
  type: string
  amount: number
  balance_after: number
  description: string
  created_at: string
}

export interface RatePlan {
  id: string
  name: string
  rate_per_minute: number
  billing_increment: number
  connection_fee: number
}

export interface DIDNumber {
  id: string
  number: string
  status: string
  assigned_to: string
}

export interface BlacklistEntry {
  id: string
  number: string
  reason: string
  created_at: string
}

export interface AuditLog {
  id: string
  action: string
  actor: string
  target: string
  details: string
  created_at: string
}

export interface PaginatedParams {
  page: number
  limit: number
}

class AuthService {
  async Login(_params: LoginParams): Promise<LoginResponse> {
    const resp = await fetch('/api/auth.Login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(_params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async Me(): Promise<User> {
    const resp = await fetch('/api/auth.Me', { credentials: 'include' })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async ListUsers(params: ListUsersParams): Promise<ListUsersResponse> {
    const resp = await fetch('/api/auth.ListUsers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async FreezeUser(params: { user_id: string }): Promise<void> {
    const resp = await fetch('/api/auth.FreezeUser', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async UnfreezeUser(params: { user_id: string }): Promise<void> {
    const resp = await fetch('/api/auth.UnfreezeUser', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async GetUser(params: { user_id: string }): Promise<CustomerDetail> {
    const resp = await fetch('/api/auth.GetUser', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async CreateUser(params: CreateUserParams): Promise<{ id: string }> {
    const resp = await fetch('/api/auth.CreateUser', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async RegenerateApiKey(params: { user_id: string }): Promise<{ api_key: string }> {
    const resp = await fetch('/api/auth.RegenerateApiKey', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }
}

class BillingService {
  async GetOverview(): Promise<OverviewResponse> {
    const resp = await fetch('/api/billing.GetOverview', { credentials: 'include' })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async TopUp(params: { user_id: string; amount: number }): Promise<void> {
    const resp = await fetch('/api/billing.TopUp', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async Deduct(params: { user_id: string; amount: number; reason: string }): Promise<void> {
    const resp = await fetch('/api/billing.Deduct', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async ListTransactions(params: PaginatedParams & { user_id?: string }): Promise<{ transactions: Transaction[]; total: number }> {
    const resp = await fetch('/api/billing.ListTransactions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async ListRatePlans(): Promise<{ plans: RatePlan[] }> {
    const resp = await fetch('/api/billing.ListRatePlans', { credentials: 'include' })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async CreateRatePlan(params: Omit<RatePlan, 'id'>): Promise<{ id: string }> {
    const resp = await fetch('/api/billing.CreateRatePlan', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }
}

class RoutingService {
  async ListGateways(): Promise<{ gateways: Gateway[] }> {
    const resp = await fetch('/api/routing.ListGateways', { credentials: 'include' })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async ToggleGateway(params: { gateway_id: string; enabled: boolean }): Promise<void> {
    const resp = await fetch('/api/routing.ToggleGateway', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async ListDIDs(params: PaginatedParams): Promise<{ dids: DIDNumber[]; total: number }> {
    const resp = await fetch('/api/routing.ListDIDs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async ImportDIDs(params: { numbers: string[] }): Promise<{ imported: number }> {
    const resp = await fetch('/api/routing.ImportDIDs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async HealthCheck(): Promise<{ status: string; gateways: Array<{ id: string; status: string }> }> {
    const resp = await fetch('/api/routing.HealthCheck', { credentials: 'include' })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async CreateGateway(params: CreateGatewayParams): Promise<{ id: string }> {
    const resp = await fetch('/api/routing.CreateGateway', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async UpdateGateway(params: { gateway_id: string } & Partial<CreateGatewayParams>): Promise<void> {
    const resp = await fetch('/api/routing.UpdateGateway', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async TestOriginate(params: { gateway_id: string; phone_number: string }): Promise<TestOriginateResult> {
    const resp = await fetch('/api/routing.TestOriginate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }
}

class CallbackService {
  async ListCDRs(params: PaginatedParams & { start_date?: string; end_date?: string }): Promise<{ cdrs: CDR[]; total: number }> {
    const resp = await fetch('/api/callback.ListCDRs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async ListActiveCalls(): Promise<{ calls: CDR[] }> {
    const resp = await fetch('/api/callback.ListActiveCalls', { credentials: 'include' })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async HangupCall(params: { call_id: string }): Promise<void> {
    const resp = await fetch('/api/callback.HangupCall', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }
}

class ComplianceService {
  async ListBlacklist(params: PaginatedParams): Promise<{ entries: BlacklistEntry[]; total: number }> {
    const resp = await fetch('/api/compliance.ListBlacklist', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }

  async AddBlacklist(params: { number: string; reason: string }): Promise<void> {
    const resp = await fetch('/api/compliance.AddBlacklist', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async RemoveBlacklist(params: { id: string }): Promise<void> {
    const resp = await fetch('/api/compliance.RemoveBlacklist', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
  }

  async ListAuditLogs(params: PaginatedParams): Promise<{ logs: AuditLog[]; total: number }> {
    const resp = await fetch('/api/compliance.ListAuditLogs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
      credentials: 'include',
    })
    if (!resp.ok) throw await resp.json()
    return resp.json()
  }
}

export default class Client {
  auth = new AuthService()
  billing = new BillingService()
  routing = new RoutingService()
  callback = new CallbackService()
  compliance = new ComplianceService()
}

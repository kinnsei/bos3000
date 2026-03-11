import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import PortalClient, { setToken } from './client'
import type { PaginatedParams, CallbackInitParams } from './client'

const api = new PortalClient()

// --- Auth ---

export function useAuth() {
  return useQuery({
    queryKey: ['portal', 'auth', 'me'],
    queryFn: () => api.auth.me(),
    retry: false,
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (params: { email: string; password: string }) => {
      const resp = await api.auth.login(params)
      setToken(resp.token)
      return resp
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'auth'] })
    },
  })
}

// --- Dashboard ---

export function useUsageSummary() {
  return useQuery({
    queryKey: ['portal', 'usage-summary'],
    queryFn: () => api.billing.getUsageSummary(),
    refetchInterval: 30_000,
  })
}

export function useBalance() {
  return useQuery({
    queryKey: ['portal', 'balance'],
    queryFn: () => api.billing.getBalance(),
    refetchInterval: 60_000,
  })
}

export function useDashboardOverview() {
  const usage = useUsageSummary()
  const balance = useBalance()

  return {
    data: usage.data && balance.data
      ? {
          today_calls: usage.data.today_calls,
          today_duration: usage.data.today_duration,
          today_cost: usage.data.today_cost,
          success_rate: usage.data.today_calls > 0
            ? +((usage.data.today_calls - Math.floor(usage.data.today_calls * 0.06)) / usage.data.today_calls * 100).toFixed(1)
            : 100,
          wastage_rate: usage.data.today_calls > 0 ? 6.2 : 0,
          balance: balance.data.balance,
          concurrent_active: usage.data.concurrent_active,
          concurrent_limit: usage.data.concurrent_limit,
        }
      : undefined,
    isLoading: usage.isLoading || balance.isLoading,
  }
}

export function useDashboardTrends() {
  return useQuery({
    queryKey: ['portal', 'dashboard', 'trends'],
    queryFn: async () => {
      // TODO: Replace with real API when backend provides trend endpoints
      const days = ['03-05', '03-06', '03-07', '03-08', '03-09', '03-10', '03-11']
      return {
        calls: days.map((d) => ({ date: d, calls: Math.floor(20 + Math.random() * 80) })),
        cost: days.map((d) => ({ date: d, cost: Math.floor(100 + Math.random() * 500) })),
      }
    },
    staleTime: 60_000,
  })
}

// --- Callback ---

export function useInitiateCallback() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: CallbackInitParams) => api.callback.initiate(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'active-calls'] })
    },
  })
}

export function useActiveCalls() {
  return useQuery({
    queryKey: ['portal', 'active-calls'],
    queryFn: () => api.callback.listActiveCalls(),
    refetchInterval: 10_000,
  })
}

export function useHangupCall() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { call_id: string }) => api.callback.hangupCall(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'active-calls'] })
      queryClient.invalidateQueries({ queryKey: ['portal', 'cdr'] })
    },
  })
}

// --- CDR ---

export function useRecentCallbacks(params: PaginatedParams = { page: 1, limit: 10 }) {
  return useQuery({
    queryKey: ['portal', 'cdr', 'recent', params],
    queryFn: () => api.callback.listCDRs(params),
  })
}

export function useCDRs(params: PaginatedParams & { start_date?: string; end_date?: string }) {
  return useQuery({
    queryKey: ['portal', 'cdr', params],
    queryFn: () => api.callback.listCDRs(params),
  })
}

export interface CDRListParams extends PaginatedParams {
  start_date?: string
  end_date?: string
  status?: string
  search?: string
}

export function useCDRList(params: CDRListParams) {
  const cleanParams = {
    ...params,
    status: params.status === 'all' ? undefined : params.status,
  }
  return useQuery({
    queryKey: ['portal', 'cdr', 'list', cleanParams],
    queryFn: () => api.callback.listCDRs(cleanParams),
  })
}

// --- Finance ---

export function useTransactions(params: PaginatedParams) {
  return useQuery({
    queryKey: ['portal', 'transactions', params],
    queryFn: () => api.billing.listTransactions(params),
  })
}

// --- Wastage ---

export function useWastageSummary() {
  return useQuery({
    queryKey: ['portal', 'wastage', 'summary'],
    queryFn: async () => {
      // TODO: Replace with real API
      return {
        today_wastage_cost: 45.67,
        today_wastage_rate: 6.2,
        top_failure_reason: 'B 路无应答',
      }
    },
  })
}

export function useWastageTrend(period: 'day' | 'week' | 'month') {
  return useQuery({
    queryKey: ['portal', 'wastage', 'trend', period],
    queryFn: async () => {
      // TODO: Replace with real API
      const counts = { day: 24, week: 7, month: 30 }
      const n = counts[period]
      return Array.from({ length: n }, (_, i) => ({
        date: period === 'day' ? `${i}:00` : period === 'week' ? `${i + 1}` : `${i + 1}`,
        wastage_rate: +(3 + Math.random() * 8).toFixed(1),
        bridge_rate: +(75 + Math.random() * 20).toFixed(1),
      }))
    },
  })
}

export function useWastageDistribution() {
  return useQuery({
    queryKey: ['portal', 'wastage', 'distribution'],
    queryFn: async () => {
      // TODO: Replace with real API
      return [
        { reason: 'B路无应答', count: 42 },
        { reason: 'A路接通B路失败', count: 28 },
        { reason: '桥接后短时挂断', count: 16 },
        { reason: 'B路忙线', count: 9 },
        { reason: '其他', count: 5 },
      ]
    },
  })
}

export interface WastageDetail {
  id: string
  call_id: string
  caller: string
  callee: string
  wastage_type: string
  wastage_cost: number
  b_leg_reason: string
  started_at: string
}

export function useWastageDetail(params: PaginatedParams & { period?: string }) {
  return useQuery({
    queryKey: ['portal', 'wastage', 'detail', params],
    queryFn: async () => {
      // TODO: Replace with real API
      const items: WastageDetail[] = Array.from({ length: params.limit }, (_, i) => {
        const idx = (params.page - 1) * params.limit + i
        const types = ['A路接通B路失败', '桥接后短时挂断', 'B路无应答', 'B路忙线']
        const reasons = ['用户未接听', '号码不存在', '网络异常', '用户拒接', '占线']
        return {
          id: `wd-${idx}`,
          call_id: `call-${1000 + idx}`,
          caller: `138${String(10000000 + idx).slice(-8)}`,
          callee: `139${String(20000000 + idx).slice(-8)}`,
          wastage_type: types[idx % types.length],
          wastage_cost: +(0.1 + Math.random() * 0.5).toFixed(2),
          b_leg_reason: reasons[idx % reasons.length],
          started_at: new Date(Date.now() - idx * 3600000).toISOString(),
        }
      })
      return { items, total: 86 }
    },
  })
}

// --- Settings ---

export function useUpdateProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { phone?: string }) => api.auth.updateProfile(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'auth'] })
    },
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (params: { old_password: string; new_password: string }) =>
      api.auth.changePassword(params),
  })
}

export function useRegenerateApiKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.auth.regenerateApiKey(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'auth'] })
    },
  })
}

// --- Rate Query ---

export function useRateQuery(prefix: string) {
  return useQuery({
    queryKey: ['portal', 'rate-query', prefix],
    queryFn: async () => {
      // Mock data until backend is ready
      if (!prefix || prefix.length < 2) return null
      return {
        plan_name: '标准套餐A',
        rate_a: 0.06,
        rate_b: 0.08,
        billing_unit: 60,
        effective_date: '2026-01-01',
      } satisfies import('./client').RateQueryResult
    },
    enabled: false,
  })
}

// --- Webhook ---

export function useWebhookConfig() {
  return useQuery({
    queryKey: ['portal', 'webhook', 'config'],
    queryFn: async () => {
      // Mock data until backend is ready
      return {
        webhook_url: '',
        webhook_secret: 'whsec_mock_abc123def456',
      } satisfies import('./client').WebhookConfig
    },
  })
}

export function useSaveWebhookConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (_params: { webhook_url: string }) => {
      // Mock mutation until backend is ready
      await new Promise((r) => setTimeout(r, 500))
      return undefined
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'webhook', 'config'] })
    },
  })
}

export function useTestWebhook() {
  return useMutation({
    mutationFn: async () => {
      // Mock mutation until backend is ready
      await new Promise((r) => setTimeout(r, 1000))
      return {
        success: true,
        status_code: 200,
        message: 'Webhook delivered successfully',
      } satisfies import('./client').WebhookTestResult
    },
  })
}

export function useRecentDeliveries() {
  return useQuery({
    queryKey: ['portal', 'webhook', 'deliveries'],
    queryFn: async () => {
      // Mock data until backend is ready
      return Array.from({ length: 5 }, (_, i) => ({
        id: `del-${i}`,
        event: ['call.completed', 'call.failed', 'balance.low', 'call.started', 'call.completed'][i],
        url: 'https://example.com/webhook',
        status_code: i === 1 ? 500 : 200,
        success: i !== 1,
        created_at: new Date(Date.now() - i * 3600000).toISOString(),
      })) satisfies import('./client').WebhookDelivery[]
    },
  })
}

// --- IP Whitelist ---

export function useAddIp() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (_params: { ip: string }) => {
      // Mock mutation until backend is ready
      await new Promise((r) => setTimeout(r, 500))
      return undefined
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'auth'] })
    },
  })
}

export function useRemoveIp() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (_params: { ip: string }) => {
      // Mock mutation until backend is ready
      await new Promise((r) => setTimeout(r, 500))
      return undefined
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'auth'] })
    },
  })
}

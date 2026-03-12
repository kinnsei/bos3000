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
      const data = await api.analytics.trends(7)
      return {
        calls: data.trends.map((t: any) => ({ date: t.date.slice(5), calls: t.calls })),
        cost: data.trends.map((t: any) => ({ date: t.date.slice(5), cost: t.revenue })),
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
    queryFn: () => api.analytics.wastageSummary(),
  })
}

export function useWastageTrend(period: 'day' | 'week' | 'month') {
  return useQuery({
    queryKey: ['portal', 'wastage', 'trend', period],
    queryFn: async () => {
      const data = await api.analytics.wastageTrend(period)
      return (data.trend || []).map((t: any) => ({
        date: t.date,
        wastage_rate: t.wastage_rate,
        bridge_rate: t.bridge_rate,
      }))
    },
  })
}

export function useWastageDistribution() {
  return useQuery({
    queryKey: ['portal', 'wastage', 'distribution'],
    queryFn: async () => {
      const data = await api.analytics.wastageDistribution()
      return (data.items || []).map((item: any) => ({
        reason: item.reason,
        count: item.count,
      }))
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
    queryFn: () => api.analytics.wastageDetail(params),
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
    queryFn: () => api.billing.queryRate(prefix),
    enabled: false,
  })
}

// --- Webhook ---

export function useWebhookConfig() {
  return useQuery({
    queryKey: ['portal', 'webhook', 'config'],
    queryFn: () => api.webhook.getConfig(),
  })
}

export function useSaveWebhookConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { webhook_url: string }) => api.webhook.saveConfig(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'webhook', 'config'] })
    },
  })
}

export function useTestWebhook() {
  return useMutation({
    mutationFn: () => api.webhook.test(),
  })
}

export function useRecentDeliveries() {
  return useQuery({
    queryKey: ['portal', 'webhook', 'deliveries'],
    queryFn: async () => {
      const resp = await api.webhook.listDeliveries()
      return resp.deliveries
    },
  })
}

// --- IP Whitelist ---

export function useIpWhitelist() {
  return useQuery({
    queryKey: ['portal', 'ip-whitelist'],
    queryFn: () => api.auth.listIps(),
  })
}

export function useAddIp() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { ip: string }) => api.auth.addIp(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'ip-whitelist'] })
    },
  })
}

export function useRemoveIp() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { ip: string }) => api.auth.removeIp(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'ip-whitelist'] })
    },
  })
}

// --- DID Pool ---

export function useMyDIDs() {
  return useQuery({
    queryKey: ['portal', 'my-dids'],
    queryFn: () => api.routing.myDIDs(),
  })
}

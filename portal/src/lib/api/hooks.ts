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

export function useCDRs(params: PaginatedParams & { start_date?: string; end_date?: string }) {
  return useQuery({
    queryKey: ['portal', 'cdr', params],
    queryFn: () => api.callback.listCDRs(params),
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

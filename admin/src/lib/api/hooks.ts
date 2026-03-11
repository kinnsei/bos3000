import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import Client from './client'
import type { ListUsersParams, PaginatedParams } from './client'

const api = new Client()

// --- Auth ---

export function useAuth() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => api.auth.Me(),
    retry: false,
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { email: string; password: string }) => api.auth.Login(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth'] })
    },
  })
}

// --- Dashboard ---

export function useDashboardOverview() {
  return useQuery({
    queryKey: ['dashboard', 'overview'],
    queryFn: () => api.billing.GetOverview(),
    refetchInterval: 30_000,
  })
}

export function useDashboardTrends() {
  return useQuery({
    queryKey: ['dashboard', 'trends'],
    queryFn: async () => {
      // TODO: Replace with real API when backend provides trend endpoints
      const days = ['03-05', '03-06', '03-07', '03-08', '03-09', '03-10', '03-11']
      return {
        revenue: days.map((d) => ({ date: d, revenue: Math.floor(5000 + Math.random() * 15000) })),
        calls: days.map((d) => ({ date: d, calls: Math.floor(200 + Math.random() * 800) })),
      }
    },
    staleTime: 60_000,
  })
}

// --- Customers ---

export function useCustomers(params: ListUsersParams) {
  return useQuery({
    queryKey: ['customers', params],
    queryFn: () => api.auth.ListUsers(params),
  })
}

export function useTopUpCustomer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { user_id: string; amount: number }) => api.billing.TopUp(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
    },
  })
}

export function useDeductCustomer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { user_id: string; amount: number; reason: string }) => api.billing.Deduct(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
    },
  })
}

export function useFreezeUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { user_id: string }) => api.auth.FreezeUser(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
    },
  })
}

export function useUnfreezeUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { user_id: string }) => api.auth.UnfreezeUser(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
    },
  })
}

// --- Gateways ---

export function useGateways() {
  return useQuery({
    queryKey: ['gateways'],
    queryFn: () => api.routing.ListGateways(),
  })
}

export function useToggleGateway() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { gateway_id: string; enabled: boolean }) => api.routing.ToggleGateway(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gateways'] })
    },
  })
}

// --- CDR ---

export function useCDRs(params: PaginatedParams & { start_date?: string; end_date?: string }) {
  return useQuery({
    queryKey: ['cdr', params],
    queryFn: () => api.callback.ListCDRs(params),
  })
}

export function useActiveCalls() {
  return useQuery({
    queryKey: ['active-calls'],
    queryFn: () => api.callback.ListActiveCalls(),
    refetchInterval: 10_000,
  })
}

export function useHangupCall() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { call_id: string }) => api.callback.HangupCall(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-calls'] })
      queryClient.invalidateQueries({ queryKey: ['cdr'] })
    },
  })
}

// --- Finance ---

export function useTransactions(params: PaginatedParams & { user_id?: string }) {
  return useQuery({
    queryKey: ['transactions', params],
    queryFn: () => api.billing.ListTransactions(params),
  })
}

export function useRatePlans() {
  return useQuery({
    queryKey: ['rate-plans'],
    queryFn: () => api.billing.ListRatePlans(),
  })
}

export function useCreateRatePlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.billing.CreateRatePlan.bind(api.billing),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rate-plans'] })
    },
  })
}

// --- DID ---

export function useDIDs(params: PaginatedParams) {
  return useQuery({
    queryKey: ['dids', params],
    queryFn: () => api.routing.ListDIDs(params),
  })
}

export function useImportDIDs() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { numbers: string[] }) => api.routing.ImportDIDs(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dids'] })
    },
  })
}

// --- Compliance ---

export function useBlacklist(params: PaginatedParams) {
  return useQuery({
    queryKey: ['blacklist', params],
    queryFn: () => api.compliance.ListBlacklist(params),
  })
}

export function useAddBlacklist() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { number: string; reason: string }) => api.compliance.AddBlacklist(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['blacklist'] })
    },
  })
}

export function useRemoveBlacklist() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { id: string }) => api.compliance.RemoveBlacklist(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['blacklist'] })
    },
  })
}

export function useAuditLogs(params: PaginatedParams) {
  return useQuery({
    queryKey: ['audit-logs', params],
    queryFn: () => api.compliance.ListAuditLogs(params),
  })
}

// --- Wastage Analysis ---

export function useWastageSummary() {
  return useQuery({
    queryKey: ['wastage', 'summary'],
    queryFn: async () => {
      // TODO: Replace with real API
      return {
        today_wastage_cost: 1234.56,
        today_wastage_rate: 8.3,
        top_failure_reason: 'B 路无应答',
      }
    },
  })
}

export function useWastageTrend(period: 'day' | 'week' | 'month') {
  return useQuery({
    queryKey: ['wastage', 'trend', period],
    queryFn: async () => {
      // TODO: Replace with real API
      const counts = { day: 24, week: 7, month: 30 }
      const n = counts[period]
      return Array.from({ length: n }, (_, i) => ({
        date: period === 'day' ? `${i}:00` : period === 'week' ? `第${i + 1}天` : `${i + 1}日`,
        wastage_rate: +(5 + Math.random() * 10).toFixed(1),
        bridge_rate: +(70 + Math.random() * 25).toFixed(1),
      }))
    },
  })
}

export function useWastageRanking() {
  return useQuery({
    queryKey: ['wastage', 'ranking'],
    queryFn: async () => {
      // TODO: Replace with real API
      const names = ['客户A', '客户B', '客户C', '客户D', '客户E', '客户F', '客户G', '客户H', '客户I', '客户J']
      return names.map((name, i) => ({
        customer_name: name,
        wastage_cost: Math.floor(5000 - i * 400 + Math.random() * 200),
      }))
    },
  })
}

export function useWastageDistribution() {
  return useQuery({
    queryKey: ['wastage', 'distribution'],
    queryFn: async () => {
      // TODO: Replace with real API
      return [
        { reason: 'b_no_answer', count: 342 },
        { reason: 'a_connected_b_failed', count: 218 },
        { reason: 'bridge_broken_early', count: 156 },
        { reason: 'b_busy', count: 89 },
        { reason: 'b_rejected', count: 67 },
        { reason: 'other', count: 43 },
      ]
    },
  })
}

// --- Ops ---

export function useHealthCheck() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => api.routing.HealthCheck(),
    refetchInterval: 15_000,
  })
}

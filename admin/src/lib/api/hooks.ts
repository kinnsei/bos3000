import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import Client from './client'
import type { ListUsersParams, PaginatedParams, CreateUserParams, CreateGatewayParams } from './client'

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
      const data = await api.analytics.GetTrends(7)
      return {
        revenue: data.trends.map((t: any) => ({ date: t.date.slice(5), revenue: t.revenue })),
        calls: data.trends.map((t: any) => ({ date: t.date.slice(5), calls: t.calls })),
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

export function useCustomer(userId: string) {
  return useQuery({
    queryKey: ['customers', userId],
    queryFn: () => api.auth.GetUser({ user_id: userId }),
    enabled: !!userId,
  })
}

export function useCreateCustomer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: CreateUserParams) => api.auth.CreateUser(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
    },
  })
}

export function useRegenerateApiKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { user_id: string }) => api.auth.RegenerateApiKey(params),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['customers', variables.user_id] })
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

export function useCreateGateway() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: CreateGatewayParams) => api.routing.CreateGateway(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gateways'] })
    },
  })
}

export function useUpdateGateway() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { gateway_id: string } & Partial<CreateGatewayParams>) => api.routing.UpdateGateway(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gateways'] })
    },
  })
}

export function useTestOriginate() {
  return useMutation({
    mutationFn: (params: { gateway_id: string; phone_number: string }) => api.routing.TestOriginate(params),
  })
}

// --- CDR ---

export function useCDRs(params: PaginatedParams & { start_date?: string; end_date?: string; status?: string }) {
  return useQuery({
    queryKey: ['cdr', params],
    queryFn: () => api.callback.ListCDRs(params),
  })
}

export function useActiveCalls() {
  return useQuery({
    queryKey: ['admin', 'active-calls'],
    queryFn: () => api.callback.ListActiveCalls(),
    // WebSocket invalidates this query; no polling needed
  })
}

export function useHangupCall() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { call_id: string }) => api.callback.HangupCall(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'active-calls'] })
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

export function useUpdateRatePlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.billing.UpdateRatePlan.bind(api.billing),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rate-plans'] })
    },
  })
}

export function useDeleteRatePlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { id: string }) => api.billing.DeleteRatePlan(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rate-plans'] })
    },
  })
}

export function useProfitByCustomer(period: 'today' | 'week' | 'month') {
  return useQuery({
    queryKey: ['profit', 'by-customer', period],
    queryFn: async () => {
      const data = await api.analytics.GetProfitByCustomer(period)
      return (data.items ?? []).map((item: any) => ({
        name: String(item.user_id),
        revenue: item.revenue,
        cost: item.cost,
        profit: item.profit,
      }))
    },
  })
}

export function useProfitByGateway(period: 'today' | 'week' | 'month') {
  return useQuery({
    queryKey: ['profit', 'by-gateway', period],
    queryFn: async () => {
      const data = await api.analytics.GetProfitByGateway(period)
      return (data.items ?? []).map((item: any) => ({
        name: item.gateway_name,
        revenue: item.revenue,
        cost: item.cost,
        profit: item.profit,
      }))
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
    queryFn: () => api.analytics.GetWastageSummary(),
  })
}

export function useWastageTrend(period: 'day' | 'week' | 'month') {
  return useQuery({
    queryKey: ['wastage', 'trend', period],
    queryFn: async () => {
      const data = await api.analytics.GetWastageTrend(period)
      return data.trend ?? []
    },
  })
}

export function useWastageRanking() {
  return useQuery({
    queryKey: ['wastage', 'ranking'],
    queryFn: async () => {
      const data = await api.analytics.GetWastageRanking()
      return (data.items ?? []).map((item: any) => ({
        customer_name: String(item.user_id),
        wastage_cost: item.wastage_cost,
      }))
    },
  })
}

export function useWastageDistribution() {
  return useQuery({
    queryKey: ['wastage', 'distribution'],
    queryFn: async () => {
      const data = await api.analytics.GetWastageDistribution()
      return data.items ?? []
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

export function useFSStatus() {
  return useQuery({
    queryKey: ['ops', 'fs-status'],
    queryFn: () => api.ops.GetFSStatus(),
    refetchInterval: 10_000,
  })
}

export function useSystemHealth() {
  return useQuery({
    queryKey: ['ops', 'system-health'],
    queryFn: () => api.ops.GetSystemHealth(),
    refetchInterval: 30_000,
  })
}

export function useSystemConfigs() {
  return useQuery({
    queryKey: ['settings', 'configs'],
    queryFn: async () => {
      const data = await api.settings.ListConfigs()
      return data.configs ?? []
    },
  })
}

export function useUpdateConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { key: string; value: string }) => api.settings.UpdateConfig(params.key, params.value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings', 'configs'] })
    },
  })
}

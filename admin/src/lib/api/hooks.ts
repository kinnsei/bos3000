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

export function useUpdateRatePlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.billing.CreateRatePlan.bind(api.billing), // TODO: Replace with UpdateRatePlan API
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rate-plans'] })
    },
  })
}

export function useProfitByCustomer(period: 'today' | 'week' | 'month') {
  return useQuery({
    queryKey: ['profit', 'by-customer', period],
    queryFn: async () => {
      // TODO: Replace with real API
      const names = ['示例科技', '通达通信', '星辰网络', '云桥通讯', '汇联科技', '盛达科技', '明远通讯', '华信网络', '联创科技', '金桥通信']
      return names.map((name) => ({
        name,
        revenue: Math.floor(3000 + Math.random() * 20000),
        cost: Math.floor(1500 + Math.random() * 10000),
        profit: Math.floor(1000 + Math.random() * 10000),
      }))
    },
  })
}

export function useProfitByGateway(period: 'today' | 'week' | 'month') {
  return useQuery({
    queryKey: ['profit', 'by-gateway', period],
    queryFn: async () => {
      // TODO: Replace with real API
      const gateways = ['GW-SH-01', 'GW-BJ-02', 'GW-GZ-03', 'GW-SZ-04', 'GW-CD-05']
      return gateways.map((name) => ({
        name,
        revenue: Math.floor(5000 + Math.random() * 30000),
        cost: Math.floor(2500 + Math.random() * 15000),
        profit: Math.floor(2000 + Math.random() * 15000),
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

export function useFSStatus() {
  return useQuery({
    queryKey: ['ops', 'fs-status'],
    queryFn: async () => ({
      instances: [
        { hostname: 'fs-primary-01', connected: true, active_sessions: 42, uptime: '15d 8h 32m', last_check: new Date().toISOString() },
        { hostname: 'fs-standby-02', connected: true, active_sessions: 0, uptime: '15d 8h 30m', last_check: new Date().toISOString() },
      ],
    }),
    refetchInterval: 10_000,
  })
}

export function useSystemHealth() {
  return useQuery({
    queryKey: ['ops', 'system-health'],
    queryFn: async () => ({
      database: { active: 8, idle: 12, max: 50, latency_ms: 3.2 },
      redis: { connected: true, memory_mb: 128 },
      api: { requests_per_min: 340, error_rate: 0.3, avg_latency_ms: 45 },
    }),
    refetchInterval: 30_000,
  })
}

export function useSystemConfigs() {
  return useQuery({
    queryKey: ['settings', 'configs'],
    queryFn: async () => ([
      { key: 'call.max_duration', value: '3600', type: 'number', description: '最大通话时长(秒)', category: 'call', updated_at: '2026-03-10T10:00:00Z' },
      { key: 'call.ring_timeout', value: '30', type: 'number', description: '振铃超时(秒)', category: 'call', updated_at: '2026-03-09T14:00:00Z' },
      { key: 'billing.auto_deduct', value: 'true', type: 'boolean', description: '自动扣费', category: 'billing', updated_at: '2026-03-08T09:00:00Z' },
      { key: 'billing.min_balance', value: '10', type: 'number', description: '最低余额警告(CNY)', category: 'billing', updated_at: '2026-03-07T16:00:00Z' },
      { key: 'compliance.blacklist_enabled', value: 'true', type: 'boolean', description: '启用黑名单过滤', category: 'compliance', updated_at: '2026-03-06T11:00:00Z' },
      { key: 'system.log_level', value: 'info', type: 'string', description: '日志级别', category: 'system', updated_at: '2026-03-05T08:00:00Z' },
    ]),
  })
}

export function useUpdateConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (_params: { key: string; value: string }) => {},
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings', 'configs'] })
    },
  })
}

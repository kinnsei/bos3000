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

// --- Ops ---

export function useHealthCheck() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => api.routing.HealthCheck(),
    refetchInterval: 15_000,
  })
}

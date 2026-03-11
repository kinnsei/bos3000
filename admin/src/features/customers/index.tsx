import { useState, useCallback } from 'react'
import type { PaginationState, SortingState } from '@tanstack/react-table'
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
import { toast } from 'sonner'
import { CustomerTable, type Customer } from './components/customer-table'
import { CreateCustomerSheet, type CreateCustomerData } from './components/create-customer-sheet'
import { BalanceDialog } from './components/balance-dialog'
import { CustomerDetail } from './components/customer-detail'
import {
  useCustomers,
  useCreateCustomer,
  useTopUpCustomer,
  useDeductCustomer,
  useFreezeUser,
  useUnfreezeUser,
} from '@/lib/api/hooks'

export default function Customers() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [sorting, setSorting] = useState<SortingState>([])
  const [search, setSearch] = useState('')

  const [createOpen, setCreateOpen] = useState(false)
  const [balanceOpen, setBalanceOpen] = useState(false)
  const [balanceMode, setBalanceMode] = useState<'topup' | 'deduct'>('topup')
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null)
  const [detailId, setDetailId] = useState<string | null>(null)

  const { data, isLoading } = useCustomers({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    search,
  })

  const createMutation = useCreateCustomer()
  const topUpMutation = useTopUpCustomer()
  const deductMutation = useDeductCustomer()
  const freezeMutation = useFreezeUser()
  const unfreezeMutation = useUnfreezeUser()

  // Map API response to table format
  const customers: Customer[] = (data?.users || []).map((u) => ({
    id: String(u.id),
    company: u.username,
    email: u.email,
    balance: u.balance / 100, // cents to yuan
    status: u.status,
    max_concurrent: u.max_concurrent,
    daily_limit: u.daily_limit,
    created_at: u.created_at,
  }))

  const handleCreateSubmit = useCallback(
    async (formData: CreateCustomerData) => {
      try {
        await createMutation.mutateAsync({
          username: formData.company,
          email: formData.email,
          password: formData.password,
          credit_limit: (formData.credit_limit || 0) * 100,
          max_concurrent: formData.max_concurrent || 10,
          daily_limit: formData.daily_limit || 1000,
        })
        toast.success('客户创建成功')
        setCreateOpen(false)
      } catch (err: any) {
        toast.error(err?.message || '创建失败')
      }
    },
    [createMutation],
  )

  const handleTopUp = useCallback((customer: Customer) => {
    setSelectedCustomer(customer)
    setBalanceMode('topup')
    setBalanceOpen(true)
  }, [])

  const handleDeduct = useCallback((customer: Customer) => {
    setSelectedCustomer(customer)
    setBalanceMode('deduct')
    setBalanceOpen(true)
  }, [])

  const handleFreeze = useCallback(
    async (customer: Customer) => {
      try {
        await freezeMutation.mutateAsync({ user_id: customer.id })
        toast.success(`${customer.company} 已冻结`)
      } catch (err: any) {
        toast.error(err?.message || '冻结失败')
      }
    },
    [freezeMutation],
  )

  const handleUnfreeze = useCallback(
    async (customer: Customer) => {
      try {
        await unfreezeMutation.mutateAsync({ user_id: customer.id })
        toast.success(`${customer.company} 已解冻`)
      } catch (err: any) {
        toast.error(err?.message || '解冻失败')
      }
    },
    [unfreezeMutation],
  )

  const handleBalanceSubmit = useCallback(
    async (formData: { amount: number; remark?: string }) => {
      if (!selectedCustomer) return
      try {
        if (balanceMode === 'topup') {
          await topUpMutation.mutateAsync({
            user_id: selectedCustomer.id,
            amount: formData.amount * 100, // yuan to cents
          })
        } else {
          await deductMutation.mutateAsync({
            user_id: selectedCustomer.id,
            amount: formData.amount * 100,
            reason: formData.remark || '管理员扣款',
          })
        }
        const action = balanceMode === 'topup' ? '充值' : '扣款'
        toast.success(`${action} ¥${formData.amount.toFixed(2)} 成功`)
        setBalanceOpen(false)
      } catch (err: any) {
        toast.error(err?.message || '操作失败')
      }
    },
    [selectedCustomer, balanceMode, topUpMutation, deductMutation],
  )

  if (detailId) {
    return (
      <div>
        <CustomerDetail customerId={detailId} onBack={() => setDetailId(null)} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">客户管理</h1>
          <p className="text-sm text-muted-foreground">客户列表、开户、充值与冻结</p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-1.5 h-4 w-4" />
          新建客户
        </Button>
      </div>

      <CustomerTable
        data={customers}
        totalCount={data?.total || 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        sorting={sorting}
        onSortingChange={setSorting}
        search={search}
        onSearchChange={setSearch}
        isLoading={isLoading}
        onViewDetail={setDetailId}
        onTopUp={handleTopUp}
        onDeduct={handleDeduct}
        onFreeze={handleFreeze}
        onUnfreeze={handleUnfreeze}
      />

      <CreateCustomerSheet
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSubmit={handleCreateSubmit}
      />

      <BalanceDialog
        open={balanceOpen}
        onOpenChange={setBalanceOpen}
        mode={balanceMode}
        customer={
          selectedCustomer
            ? { id: selectedCustomer.id, username: selectedCustomer.company, balance: selectedCustomer.balance }
            : null
        }
        onSubmit={handleBalanceSubmit}
      />
    </div>
  )
}

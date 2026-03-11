import { useState, useCallback } from 'react'
import type { PaginationState, SortingState } from '@tanstack/react-table'
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
import { toast } from 'sonner'
import { CustomerTable, type Customer } from './components/customer-table'
import { CreateCustomerSheet, type CreateCustomerData } from './components/create-customer-sheet'
import { BalanceDialog } from './components/balance-dialog'
import { CustomerDetail } from './components/customer-detail'

// Mock customer data
const mockCustomers: Customer[] = [
  { id: '1', company: '示例科技有限公司', email: 'admin@example.com', balance: 12500.50, status: 'active', max_concurrent: 50, daily_limit: 2000, created_at: '2025-12-01T08:00:00Z' },
  { id: '2', company: '通达通信', email: 'ops@tongda.com', balance: 3200.00, status: 'active', max_concurrent: 20, daily_limit: 500, created_at: '2026-01-15T10:30:00Z' },
  { id: '3', company: '汇联科技', email: 'tech@huilian.cn', balance: 0, status: 'frozen', max_concurrent: 10, daily_limit: 100, created_at: '2026-02-20T14:00:00Z' },
  { id: '4', company: '星辰网络', email: 'admin@xingchen.net', balance: 45800.00, status: 'active', max_concurrent: 100, daily_limit: 5000, created_at: '2025-11-10T09:00:00Z' },
  { id: '5', company: '云桥通讯', email: 'support@yunqiao.com', balance: 780.25, status: 'active', max_concurrent: 15, daily_limit: 300, created_at: '2026-03-01T16:45:00Z' },
]

export default function Customers() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [sorting, setSorting] = useState<SortingState>([])
  const [search, setSearch] = useState('')

  const [createOpen, setCreateOpen] = useState(false)
  const [balanceOpen, setBalanceOpen] = useState(false)
  const [balanceMode, setBalanceMode] = useState<'topup' | 'deduct'>('topup')
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null)
  const [detailId, setDetailId] = useState<string | null>(null)

  // Filter mock data by search
  const filtered = mockCustomers.filter((c) => {
    if (!search) return true
    const q = search.toLowerCase()
    return c.company.toLowerCase().includes(q) || c.email.toLowerCase().includes(q)
  })

  const handleCreateSubmit = useCallback(async (_data: CreateCustomerData) => {
    // Mock submit
    toast.success('客户创建成功')
  }, [])

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

  const handleFreeze = useCallback((customer: Customer) => {
    toast.success(`${customer.company} 已冻结`)
  }, [])

  const handleUnfreeze = useCallback((customer: Customer) => {
    toast.success(`${customer.company} 已解冻`)
  }, [])

  const handleBalanceSubmit = useCallback(
    async (data: { amount: number; remark?: string }) => {
      if (!selectedCustomer) return
      const action = balanceMode === 'topup' ? '充值' : '扣款'
      toast.success(`${action} ¥${data.amount.toFixed(2)} 成功`)
    },
    [selectedCustomer, balanceMode],
  )

  // Show detail view
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
          <p className="text-muted-foreground">客户列表、开户、充值与冻结</p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-1.5 h-4 w-4" />
          新建客户
        </Button>
      </div>

      <CustomerTable
        data={filtered}
        totalCount={filtered.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        sorting={sorting}
        onSortingChange={setSorting}
        search={search}
        onSearchChange={setSearch}
        isLoading={false}
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

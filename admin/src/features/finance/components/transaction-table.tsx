import { useState, useMemo } from 'react'
import type { ColumnDef, PaginationState, SortingState } from '@tanstack/react-table'
import { DataTable } from '@/components/shared/data-table'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'

export interface Transaction {
  id: string
  user_id: string
  customer_name: string
  type: 'topup' | 'deduction' | 'call_charge' | 'refund'
  amount: number
  balance_after: number
  description: string
  created_at: string
}

const TYPE_CONFIG: Record<Transaction['type'], { label: string; color: string }> = {
  topup: { label: '充值', color: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  deduction: { label: '扣款', color: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400' },
  call_charge: { label: '通话扣费', color: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  refund: { label: '退款', color: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400' },
}

const mockTransactions: Transaction[] = [
  { id: 'txn_001abc', user_id: '1', customer_name: '示例科技有限公司', type: 'topup', amount: 5000, balance_after: 17500.50, description: '在线充值', created_at: '2026-03-10T14:30:00Z' },
  { id: 'txn_002def', user_id: '2', customer_name: '通达通信', type: 'call_charge', amount: -12.50, balance_after: 3187.50, description: '通话扣费 010-12345678', created_at: '2026-03-10T13:45:00Z' },
  { id: 'txn_003ghi', user_id: '1', customer_name: '示例科技有限公司', type: 'deduction', amount: -200, balance_after: 12300.50, description: '手动扣款 - 违约金', created_at: '2026-03-10T12:00:00Z' },
  { id: 'txn_004jkl', user_id: '4', customer_name: '星辰网络', type: 'topup', amount: 10000, balance_after: 55800, description: '银行转账充值', created_at: '2026-03-09T16:00:00Z' },
  { id: 'txn_005mno', user_id: '2', customer_name: '通达通信', type: 'refund', amount: 50, balance_after: 3250, description: '通话失败退款', created_at: '2026-03-09T10:20:00Z' },
  { id: 'txn_006pqr', user_id: '5', customer_name: '云桥通讯', type: 'call_charge', amount: -8.75, balance_after: 771.50, description: '通话扣费 021-87654321', created_at: '2026-03-09T09:15:00Z' },
  { id: 'txn_007stu', user_id: '4', customer_name: '星辰网络', type: 'call_charge', amount: -35.20, balance_after: 45764.80, description: '通话扣费 0755-11223344', created_at: '2026-03-08T18:30:00Z' },
  { id: 'txn_008vwx', user_id: '1', customer_name: '示例科技有限公司', type: 'deduction', amount: -500, balance_after: 12000.50, description: '月度服务费', created_at: '2026-03-08T08:00:00Z' },
  { id: 'txn_009yza', user_id: '3', customer_name: '汇联科技', type: 'topup', amount: 2000, balance_after: 2000, description: '在线充值', created_at: '2026-03-07T15:00:00Z' },
  { id: 'txn_010bcd', user_id: '5', customer_name: '云桥通讯', type: 'call_charge', amount: -6.30, balance_after: 773.95, description: '通话扣费 020-55667788', created_at: '2026-03-07T11:45:00Z' },
]

const columns: ColumnDef<Transaction, unknown>[] = [
  {
    accessorKey: 'id',
    header: '交易ID',
    cell: ({ row }) => (
      <span className="font-mono text-xs text-muted-foreground" title={row.original.id}>
        {row.original.id.slice(0, 11)}...
      </span>
    ),
  },
  {
    accessorKey: 'customer_name',
    header: '客户名称',
  },
  {
    accessorKey: 'type',
    header: '类型',
    cell: ({ row }) => {
      const cfg = TYPE_CONFIG[row.original.type]
      return <Badge variant="outline" className={cfg.color}>{cfg.label}</Badge>
    },
  },
  {
    accessorKey: 'amount',
    header: '金额',
    enableSorting: true,
    cell: ({ row }) => {
      const amount = row.original.amount
      const isPositive = amount > 0
      return (
        <span className={cn('font-mono font-medium', isPositive ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400')}>
          {isPositive ? '+' : ''}{amount.toFixed(2)}
        </span>
      )
    },
  },
  {
    accessorKey: 'balance_after',
    header: '余额',
    cell: ({ row }) => <span className="font-mono">¥{row.original.balance_after.toFixed(2)}</span>,
  },
  {
    accessorKey: 'description',
    header: '备注',
  },
  {
    accessorKey: 'created_at',
    header: '时间',
    enableSorting: true,
    cell: ({ row }) => new Date(row.original.created_at).toLocaleString('zh-CN'),
  },
]

const CUSTOMER_OPTIONS = [
  { value: 'all', label: '全部客户' },
  { value: '1', label: '示例科技有限公司' },
  { value: '2', label: '通达通信' },
  { value: '3', label: '汇联科技' },
  { value: '4', label: '星辰网络' },
  { value: '5', label: '云桥通讯' },
]

const TYPE_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'topup', label: '充值' },
  { value: 'deduction', label: '扣款' },
  { value: 'call_charge', label: '通话扣费' },
  { value: 'refund', label: '退款' },
]

export function TransactionTable() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [sorting, setSorting] = useState<SortingState>([{ id: 'created_at', desc: true }])
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [customerId, setCustomerId] = useState('all')
  const [typeFilter, setTypeFilter] = useState('all')

  const filtered = useMemo(() => {
    return mockTransactions.filter((t) => {
      if (customerId !== 'all' && t.user_id !== customerId) return false
      if (typeFilter !== 'all' && t.type !== typeFilter) return false
      if (dateFrom && t.created_at < dateFrom) return false
      if (dateTo && t.created_at > dateTo + 'T23:59:59Z') return false
      return true
    })
  }, [customerId, typeFilter, dateFrom, dateTo])

  const summary = useMemo(() => {
    const totalIn = filtered.filter((t) => t.amount > 0).reduce((s, t) => s + t.amount, 0)
    const totalOut = filtered.filter((t) => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)
    return { totalIn, totalOut, net: totalIn - totalOut }
  }, [filtered])

  const toolbar = (
    <div className="space-y-3">
      <div className="flex flex-wrap items-end gap-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">开始日期</Label>
          <Input type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} className="h-8 w-36" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">结束日期</Label>
          <Input type="date" value={dateTo} onChange={(e) => setDateTo(e.target.value)} className="h-8 w-36" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">客户</Label>
          <Select value={customerId} onValueChange={setCustomerId}>
            <SelectTrigger className="h-8 w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CUSTOMER_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">类型</Label>
          <Select value={typeFilter} onValueChange={setTypeFilter}>
            <SelectTrigger className="h-8 w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TYPE_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="flex gap-4 text-sm">
        <span>
          总入账: <span className="font-mono font-medium text-green-600 dark:text-green-400">+¥{summary.totalIn.toFixed(2)}</span>
        </span>
        <span>
          总出账: <span className="font-mono font-medium text-red-600 dark:text-red-400">-¥{summary.totalOut.toFixed(2)}</span>
        </span>
        <span>
          净额: <span className={cn('font-mono font-medium', summary.net >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400')}>
            {summary.net >= 0 ? '+' : ''}¥{summary.net.toFixed(2)}
          </span>
        </span>
      </div>
    </div>
  )

  return (
    <DataTable
      columns={columns}
      data={filtered}
      totalCount={filtered.length}
      pagination={pagination}
      onPaginationChange={setPagination}
      sorting={sorting}
      onSortingChange={setSorting}
      isLoading={false}
      toolbar={toolbar}
    />
  )
}

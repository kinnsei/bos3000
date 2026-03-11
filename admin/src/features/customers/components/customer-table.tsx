import { useMemo } from 'react'
import type { ColumnDef, PaginationState, SortingState, OnChangeFn } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { DataTable } from '@/components/shared/data-table'
import { MoreHorizontal, Search } from 'lucide-react'

export interface Customer {
  id: string
  company: string
  email: string
  balance: number
  status: 'active' | 'frozen'
  max_concurrent: number
  daily_limit: number
  created_at: string
}

interface CustomerTableProps {
  data: Customer[]
  totalCount: number
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  sorting: SortingState
  onSortingChange: OnChangeFn<SortingState>
  search: string
  onSearchChange: (value: string) => void
  isLoading?: boolean
  onViewDetail: (id: string) => void
  onTopUp: (customer: Customer) => void
  onDeduct: (customer: Customer) => void
  onFreeze: (customer: Customer) => void
  onUnfreeze: (customer: Customer) => void
}

export function CustomerTable({
  data,
  totalCount,
  pagination,
  onPaginationChange,
  sorting,
  onSortingChange,
  search,
  onSearchChange,
  isLoading,
  onViewDetail,
  onTopUp,
  onDeduct,
  onFreeze,
  onUnfreeze,
}: CustomerTableProps) {
  const columns = useMemo<ColumnDef<Customer, unknown>[]>(
    () => [
      {
        accessorKey: 'company',
        header: '公司名称',
        enableSorting: false,
      },
      {
        accessorKey: 'email',
        header: '邮箱',
        enableSorting: false,
      },
      {
        accessorKey: 'balance',
        header: '余额',
        cell: ({ row }) => (
          <span className="font-mono">
            ¥{row.original.balance.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}
          </span>
        ),
      },
      {
        accessorKey: 'status',
        header: '状态',
        enableSorting: false,
        cell: ({ row }) => {
          const status = row.original.status
          return (
            <Badge className={status === 'active' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'}>
              {status === 'active' ? '正常' : '已冻结'}
            </Badge>
          )
        },
      },
      {
        accessorKey: 'max_concurrent',
        header: '并发上限',
        enableSorting: false,
      },
      {
        accessorKey: 'daily_limit',
        header: '日限额',
        enableSorting: false,
      },
      {
        accessorKey: 'created_at',
        header: '创建时间',
        cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString('zh-CN'),
      },
      {
        id: 'actions',
        header: '操作',
        enableSorting: false,
        cell: ({ row }) => {
          const customer = row.original
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onViewDetail(customer.id)}>
                  查看详情
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => onTopUp(customer)}>
                  充值
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onDeduct(customer)}>
                  扣款
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                {customer.status === 'active' ? (
                  <DropdownMenuItem
                    className="text-destructive"
                    onClick={() => onFreeze(customer)}
                  >
                    冻结
                  </DropdownMenuItem>
                ) : (
                  <DropdownMenuItem onClick={() => onUnfreeze(customer)}>
                    解冻
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      },
    ],
    [onViewDetail, onTopUp, onDeduct, onFreeze, onUnfreeze],
  )

  const toolbar = (
    <div className="flex items-center gap-2">
      <div className="relative flex-1 max-w-sm">
        <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="搜索公司名称或邮箱..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-8"
        />
      </div>
    </div>
  )

  return (
    <DataTable
      columns={columns}
      data={data}
      totalCount={totalCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      sorting={sorting}
      onSortingChange={onSortingChange}
      isLoading={isLoading}
      toolbar={toolbar}
    />
  )
}

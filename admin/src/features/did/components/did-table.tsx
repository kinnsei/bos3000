import { useMemo } from 'react'
import type { ColumnDef, PaginationState, SortingState, OnChangeFn } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { DataTable } from '@/components/shared/data-table'
import { MoreHorizontal } from 'lucide-react'

export interface DIDRow {
  id: string
  number: string
  status: 'active' | 'inactive'
  assigned_to: string
  assigned_name: string
  pool_type: 'dedicated' | 'shared'
  created_at: string
}

interface DidTableProps {
  data: DIDRow[]
  totalCount: number
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  sorting: SortingState
  onSortingChange: OnChangeFn<SortingState>
  isLoading?: boolean
  assignFilter: string
  onAssignFilterChange: (value: string) => void
  customerFilter: string
  onCustomerFilterChange: (value: string) => void
  customers: { id: string; name: string }[]
  onAssign: (did: DIDRow) => void
  onUnassign: (did: DIDRow) => void
  onToggle: (did: DIDRow) => void
}

export function DidTable({
  data,
  totalCount,
  pagination,
  onPaginationChange,
  sorting,
  onSortingChange,
  isLoading,
  assignFilter,
  onAssignFilterChange,
  customerFilter,
  onCustomerFilterChange,
  customers,
  onAssign,
  onUnassign,
  onToggle,
}: DidTableProps) {
  const columns = useMemo<ColumnDef<DIDRow, unknown>[]>(
    () => [
      {
        accessorKey: 'number',
        header: 'DID号码',
        enableSorting: false,
        cell: ({ row }) => (
          <span className="font-mono text-sm">{row.original.number}</span>
        ),
      },
      {
        accessorKey: 'assigned_name',
        header: '分配客户',
        enableSorting: false,
        cell: ({ row }) =>
          row.original.assigned_to ? (
            <span>{row.original.assigned_name}</span>
          ) : (
            <span className="text-muted-foreground">未分配</span>
          ),
      },
      {
        accessorKey: 'pool_type',
        header: '号码池',
        enableSorting: false,
        cell: ({ row }) => {
          const pool = row.original.pool_type
          return (
            <Badge
              variant="secondary"
              className={
                pool === 'dedicated'
                  ? 'bg-blue-500/10 text-blue-600'
                  : 'bg-gray-500/10 text-gray-600'
              }
            >
              {pool === 'dedicated' ? '专属' : '共享'}
            </Badge>
          )
        },
      },
      {
        accessorKey: 'status',
        header: '状态',
        enableSorting: false,
        cell: ({ row }) => {
          const status = row.original.status
          return (
            <Badge
              className={
                status === 'active'
                  ? 'bg-green-500/10 text-green-600'
                  : 'bg-red-500/10 text-red-600'
              }
            >
              {status === 'active' ? '启用' : '停用'}
            </Badge>
          )
        },
      },
      {
        accessorKey: 'created_at',
        header: '创建时间',
        cell: ({ row }) =>
          new Date(row.original.created_at).toLocaleDateString('zh-CN'),
      },
      {
        id: 'actions',
        header: '操作',
        enableSorting: false,
        cell: ({ row }) => {
          const did = row.original
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onAssign(did)}>
                  分配客户
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!did.assigned_to}
                  onClick={() => onUnassign(did)}
                >
                  取消分配
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className={did.status === 'active' ? 'text-destructive' : ''}
                  onClick={() => onToggle(did)}
                >
                  {did.status === 'active' ? '停用' : '启用'}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      },
    ],
    [onAssign, onUnassign, onToggle],
  )

  const toolbar = (
    <div className="flex items-center gap-3">
      <Select value={assignFilter} onValueChange={onAssignFilterChange}>
        <SelectTrigger className="w-[130px]">
          <SelectValue placeholder="分配状态" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部</SelectItem>
          <SelectItem value="assigned">已分配</SelectItem>
          <SelectItem value="unassigned">未分配</SelectItem>
        </SelectContent>
      </Select>
      <Select value={customerFilter} onValueChange={onCustomerFilterChange}>
        <SelectTrigger className="w-[160px]">
          <SelectValue placeholder="选择客户" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部客户</SelectItem>
          {customers.map((c) => (
            <SelectItem key={c.id} value={c.id}>
              {c.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
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

import { type ColumnDef, type PaginationState, type SortingState, type OnChangeFn } from '@tanstack/react-table'
import { DataTable } from '@/components/shared/data-table'
import { Badge } from '@/components/ui/badge'
import type { CDR } from '@/lib/api/client'
import type { ReactNode } from 'react'
import { formatDuration } from '../utils'

const STATUS_MAP: Record<string, { label: string; className: string }> = {
  completed: { label: '已完成', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  failed: { label: '失败', className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
  'in-progress': { label: '进行中', className: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' },
}

// Mock gateway/customer/hangup-reason enrichment
const MOCK_GATEWAYS = ['网关-北京01', '网关-上海02', '网关-广州03']
const MOCK_CUSTOMERS = ['客户A', '客户B', '客户C', '客户D', '客户E']
const MOCK_HANGUP_REASONS: Record<string, string> = {
  completed: '正常挂断',
  failed: 'B路无应答',
  'in-progress': '-',
}

function hashPick(id: string, arr: string[]) {
  let h = 0
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) | 0
  return arr[Math.abs(h) % arr.length]
}

const columns: ColumnDef<CDR, unknown>[] = [
  {
    accessorKey: 'call_id',
    header: '呼叫ID',
    cell: ({ row }) => (
      <span className="font-mono text-xs" title={row.original.call_id}>
        {row.original.call_id.slice(0, 8)}
      </span>
    ),
    enableSorting: false,
  },
  {
    id: 'customer',
    header: '客户',
    cell: ({ row }) => hashPick(row.original.id, MOCK_CUSTOMERS),
    enableSorting: false,
  },
  {
    accessorKey: 'caller',
    header: 'A路号码',
    enableSorting: false,
  },
  {
    accessorKey: 'callee',
    header: 'B路号码',
    enableSorting: false,
  },
  {
    accessorKey: 'status',
    header: '状态',
    cell: ({ row }) => {
      const s = STATUS_MAP[row.original.status] ?? { label: row.original.status, className: '' }
      return <Badge variant="outline" className={s.className}>{s.label}</Badge>
    },
    enableSorting: false,
  },
  {
    accessorKey: 'started_at',
    header: '开始时间',
    cell: ({ row }) => {
      const d = new Date(row.original.started_at)
      return d.toLocaleString('zh-CN', { hour12: false })
    },
    enableSorting: true,
  },
  {
    accessorKey: 'duration',
    header: '时长',
    cell: ({ row }) => formatDuration(row.original.duration),
    enableSorting: true,
  },
  {
    id: 'gateway',
    header: '网关',
    cell: ({ row }) => hashPick(row.original.call_id, MOCK_GATEWAYS),
    enableSorting: false,
  },
  {
    accessorKey: 'cost',
    header: '费用',
    cell: ({ row }) => `¥${row.original.cost.toFixed(2)}`,
    enableSorting: true,
  },
  {
    id: 'hangup_reason',
    header: '挂断原因',
    cell: ({ row }) => MOCK_HANGUP_REASONS[row.original.status] ?? '-',
    enableSorting: false,
  },
]

interface CdrTableProps {
  data: CDR[]
  totalCount: number
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  sorting: SortingState
  onSortingChange: OnChangeFn<SortingState>
  isLoading?: boolean
  toolbar?: ReactNode
}

export function CdrTable({
  data,
  totalCount,
  pagination,
  onPaginationChange,
  sorting,
  onSortingChange,
  isLoading,
  toolbar,
}: CdrTableProps) {
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

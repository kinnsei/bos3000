import { type ColumnDef, type PaginationState, type SortingState, type OnChangeFn } from '@tanstack/react-table'
import { DataTable } from '@/components/shared/data-table'
import { Badge } from '@/components/ui/badge'
import type { CDR } from '@/lib/api/client'
import type { ReactNode } from 'react'
import { formatDuration } from '../utils'

const STATUS_MAP: Record<string, { label: string; className: string }> = {
  finished: { label: '已完成', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  completed: { label: '已完成', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  failed: { label: '失败', className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
  in_progress: { label: '进行中', className: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' },
  'in-progress': { label: '进行中', className: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' },
  ringing: { label: '振铃中', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  bridged: { label: '通话中', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
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
    id: 'a_number',
    header: 'A路号码',
    cell: ({ row }) => row.original.a_number,
    enableSorting: false,
  },
  {
    id: 'b_number',
    header: 'B路号码',
    cell: ({ row }) => row.original.b_number,
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
    accessorKey: 'created_at',
    header: '开始时间',
    cell: ({ row }) => {
      const d = new Date(row.original.created_at)
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
    accessorKey: 'cost',
    header: '费用',
    cell: ({ row }) => `¥${row.original.cost.toFixed(2)}`,
    enableSorting: true,
  },
  {
    id: 'failure_reason',
    header: '挂断原因',
    cell: ({ row }) => row.original.failure_reason || (row.original.status === 'finished' ? '正常挂断' : '-'),
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

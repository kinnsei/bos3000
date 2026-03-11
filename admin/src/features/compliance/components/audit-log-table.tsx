import { useState } from 'react'
import type { ColumnDef, PaginationState } from '@tanstack/react-table'
import { DataTable } from '@/components/shared/data-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Eye } from 'lucide-react'
import { useAuditLogs } from '@/lib/api/hooks'
import type { AuditLog } from '@/lib/api/client'

const ACTION_BADGE: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline'; className?: string }> = {
  create: { label: '创建', variant: 'default', className: 'bg-green-600 hover:bg-green-700' },
  update: { label: '更新', variant: 'default', className: 'bg-blue-600 hover:bg-blue-700' },
  delete: { label: '删除', variant: 'destructive' },
  login: { label: '登录', variant: 'secondary' },
  freeze: { label: '冻结', variant: 'destructive' },
  unfreeze: { label: '解冻', variant: 'default', className: 'bg-green-600 hover:bg-green-700' },
  topup: { label: '充值', variant: 'default', className: 'bg-green-600 hover:bg-green-700' },
  deduct: { label: '扣款', variant: 'destructive' },
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function AuditLogTable() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [detailItem, setDetailItem] = useState<AuditLog | null>(null)

  const { data, isLoading } = useAuditLogs({ page: pagination.pageIndex + 1, limit: pagination.pageSize })

  const logs = data?.logs ?? []
  const totalCount = data?.total ?? 0

  const columns: ColumnDef<AuditLog, unknown>[] = [
    {
      accessorKey: 'created_at',
      header: '时间',
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{formatDate(row.original.created_at)}</span>
      ),
    },
    {
      accessorKey: 'actor_id',
      header: '操作人',
      cell: ({ row }) => row.original.actor_name || `用户#${row.original.actor_id}`,
    },
    {
      accessorKey: 'action',
      header: '操作类型',
      cell: ({ row }) => {
        const config = ACTION_BADGE[row.original.action] ?? { label: row.original.action, variant: 'outline' as const }
        return (
          <Badge variant={config.variant} className={config.className}>
            {config.label}
          </Badge>
        )
      },
    },
    {
      accessorKey: 'target_type',
      header: '资源类型',
    },
    {
      accessorKey: 'target_id',
      header: '资源ID',
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.target_id}</span>
      ),
    },
    {
      accessorKey: 'ip_address',
      header: 'IP 地址',
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.ip_address || '-'}</span>
      ),
    },
    {
      id: 'actions',
      header: '详情',
      cell: ({ row }) => (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setDetailItem(row.original)}
        >
          <Eye className="mr-1 h-4 w-4" />
          查看详情
        </Button>
      ),
    },
  ]

  return (
    <>
      <DataTable
        columns={columns}
        data={logs}
        totalCount={totalCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        isLoading={isLoading}
      />

      <Dialog open={!!detailItem} onOpenChange={(open) => { if (!open) setDetailItem(null) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>操作详情</DialogTitle>
          </DialogHeader>
          {detailItem && (
            <div className="space-y-3 text-sm">
              <div className="grid grid-cols-[80px_1fr] gap-y-2">
                <span className="text-muted-foreground">时间</span>
                <span>{formatDate(detailItem.created_at)}</span>
                <span className="text-muted-foreground">操作人</span>
                <span>{detailItem.actor_name || `用户#${detailItem.actor_id}`}</span>
                <span className="text-muted-foreground">操作类型</span>
                <span>{ACTION_BADGE[detailItem.action]?.label ?? detailItem.action}</span>
                <span className="text-muted-foreground">资源</span>
                <span>{detailItem.target_type} / {detailItem.target_id}</span>
                <span className="text-muted-foreground">IP 地址</span>
                <span className="font-mono">{detailItem.ip_address || '-'}</span>
              </div>
              <div>
                <span className="text-muted-foreground">详细数据</span>
                <pre className="mt-1 rounded-md bg-muted p-3 text-xs overflow-auto max-h-60">
                  {detailItem.details || '无'}
                </pre>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}

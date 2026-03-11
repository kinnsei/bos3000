import { useState, useMemo } from 'react'
import type { ColumnDef, PaginationState } from '@tanstack/react-table'
import { DataTable } from '@/components/shared/data-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Eye } from 'lucide-react'

interface AuditLogItem {
  id: string
  timestamp: string
  actor: string
  action_type: 'create' | 'update' | 'delete' | 'login'
  resource_type: string
  resource_id: string
  ip_address: string
  details: Record<string, unknown>
}

const MOCK_DATA: AuditLogItem[] = [
  { id: '1', timestamp: '2026-03-11T09:15:00Z', actor: 'admin', action_type: 'login', resource_type: 'auth', resource_id: '-', ip_address: '192.168.1.100', details: { method: 'password', user_agent: 'Chrome/120' } },
  { id: '2', timestamp: '2026-03-11T09:20:00Z', actor: 'admin', action_type: 'create', resource_type: 'user', resource_id: 'usr_0042', ip_address: '192.168.1.100', details: { username: 'wangwu', email: 'wangwu@example.com', role: 'customer' } },
  { id: '3', timestamp: '2026-03-11T08:30:00Z', actor: 'zhangsan', action_type: 'update', resource_type: 'gateway', resource_id: 'gw_003', ip_address: '10.0.0.55', details: { before: { weight: 50, max_concurrent: 100 }, after: { weight: 80, max_concurrent: 200 } } },
  { id: '4', timestamp: '2026-03-10T17:45:00Z', actor: 'admin', action_type: 'delete', resource_type: 'blacklist', resource_id: 'bl_015', ip_address: '192.168.1.100', details: { number: '13800138001', reason: '误加' } },
  { id: '5', timestamp: '2026-03-10T16:00:00Z', actor: 'lisi', action_type: 'create', resource_type: 'blacklist', resource_id: 'bl_016', ip_address: '10.0.0.60', details: { number: '15099998888', reason: '骚扰电话' } },
  { id: '6', timestamp: '2026-03-10T14:20:00Z', actor: 'admin', action_type: 'update', resource_type: 'user', resource_id: 'usr_0038', ip_address: '192.168.1.100', details: { before: { status: 'active', credit_limit: 1000 }, after: { status: 'frozen', credit_limit: 1000 } } },
  { id: '7', timestamp: '2026-03-10T12:10:00Z', actor: 'zhangsan', action_type: 'login', resource_type: 'auth', resource_id: '-', ip_address: '10.0.0.55', details: { method: 'password', user_agent: 'Firefox/115' } },
  { id: '8', timestamp: '2026-03-10T10:05:00Z', actor: 'admin', action_type: 'create', resource_type: 'gateway', resource_id: 'gw_008', ip_address: '192.168.1.100', details: { name: '联通线路-华南', type: 'b_leg', host: '10.20.30.40' } },
  { id: '9', timestamp: '2026-03-09T18:30:00Z', actor: 'lisi', action_type: 'update', resource_type: 'rate_plan', resource_id: 'rp_005', ip_address: '10.0.0.60', details: { before: { rate_per_minute: 0.08 }, after: { rate_per_minute: 0.06 } } },
  { id: '10', timestamp: '2026-03-09T15:00:00Z', actor: 'admin', action_type: 'delete', resource_type: 'user', resource_id: 'usr_0012', ip_address: '192.168.1.100', details: { username: 'test_user', reason: '测试账号清理' } },
  { id: '11', timestamp: '2026-03-09T11:45:00Z', actor: 'zhangsan', action_type: 'create', resource_type: 'did', resource_id: 'did_099', ip_address: '10.0.0.55', details: { number: '02188776655', assigned_to: 'usr_0038' } },
  { id: '12', timestamp: '2026-03-09T09:00:00Z', actor: 'admin', action_type: 'login', resource_type: 'auth', resource_id: '-', ip_address: '192.168.1.100', details: { method: 'api_key', user_agent: 'curl/8.0' } },
  { id: '13', timestamp: '2026-03-08T16:30:00Z', actor: 'lisi', action_type: 'update', resource_type: 'gateway', resource_id: 'gw_001', ip_address: '10.0.0.60', details: { before: { status: 'up', weight: 100 }, after: { status: 'disabled', weight: 0 } } },
  { id: '14', timestamp: '2026-03-08T14:00:00Z', actor: 'admin', action_type: 'create', resource_type: 'blacklist', resource_id: 'bl_014', ip_address: '192.168.1.100', details: { number: '17711223344', reason: '欺诈行为', source: '批量导入' } },
  { id: '15', timestamp: '2026-03-08T10:15:00Z', actor: 'zhangsan', action_type: 'update', resource_type: 'user', resource_id: 'usr_0025', ip_address: '10.0.0.55', details: { before: { concurrent_limit: 10 }, after: { concurrent_limit: 20 } } },
  { id: '16', timestamp: '2026-03-07T17:20:00Z', actor: 'admin', action_type: 'delete', resource_type: 'did', resource_id: 'did_045', ip_address: '192.168.1.100', details: { number: '02155443322', reason: '号码回收' } },
  { id: '17', timestamp: '2026-03-07T13:40:00Z', actor: 'lisi', action_type: 'login', resource_type: 'auth', resource_id: '-', ip_address: '10.0.0.60', details: { method: 'password', user_agent: 'Safari/17' } },
  { id: '18', timestamp: '2026-03-07T09:30:00Z', actor: 'admin', action_type: 'create', resource_type: 'rate_plan', resource_id: 'rp_006', ip_address: '192.168.1.100', details: { name: '标准计费-v2', rate_per_minute: 0.05, billing_increment: 6 } },
  { id: '19', timestamp: '2026-03-06T15:50:00Z', actor: 'zhangsan', action_type: 'update', resource_type: 'gateway', resource_id: 'gw_005', ip_address: '10.0.0.55', details: { before: { host: '10.0.1.1', port: 5060 }, after: { host: '10.0.1.2', port: 5060 } } },
  { id: '20', timestamp: '2026-03-06T11:00:00Z', actor: 'admin', action_type: 'create', resource_type: 'user', resource_id: 'usr_0041', ip_address: '192.168.1.100', details: { username: 'client_hk', email: 'hk@example.com', role: 'customer' } },
]

const ACTION_TYPE_MAP: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline'; className?: string }> = {
  create: { label: '创建', variant: 'default', className: 'bg-green-600 hover:bg-green-700' },
  update: { label: '更新', variant: 'default', className: 'bg-blue-600 hover:bg-blue-700' },
  delete: { label: '删除', variant: 'destructive' },
  login: { label: '登录', variant: 'secondary' },
}

const RESOURCE_TYPES = ['全部', 'auth', 'user', 'gateway', 'blacklist', 'did', 'rate_plan']
const ACTION_TYPES = [
  { value: '全部', label: '全部' },
  { value: 'create', label: '创建' },
  { value: 'update', label: '更新' },
  { value: 'delete', label: '删除' },
  { value: 'login', label: '登录' },
]
const ACTORS = ['全部', 'admin', 'zhangsan', 'lisi']

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
  const [detailItem, setDetailItem] = useState<AuditLogItem | null>(null)

  // Filters
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [filterActor, setFilterActor] = useState('全部')
  const [filterAction, setFilterAction] = useState('全部')
  const [filterResource, setFilterResource] = useState('全部')

  const filteredData = useMemo(() => {
    return MOCK_DATA.filter((item) => {
      if (filterActor !== '全部' && item.actor !== filterActor) return false
      if (filterAction !== '全部' && item.action_type !== filterAction) return false
      if (filterResource !== '全部' && item.resource_type !== filterResource) return false
      if (dateFrom && item.timestamp < new Date(dateFrom).toISOString()) return false
      if (dateTo && item.timestamp > new Date(dateTo + 'T23:59:59').toISOString()) return false
      return true
    })
  }, [filterActor, filterAction, filterResource, dateFrom, dateTo])

  const paginatedData = useMemo(() => {
    const start = pagination.pageIndex * pagination.pageSize
    return filteredData.slice(start, start + pagination.pageSize)
  }, [filteredData, pagination])

  const columns: ColumnDef<AuditLogItem, unknown>[] = [
    {
      accessorKey: 'timestamp',
      header: '时间',
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{formatDate(row.original.timestamp)}</span>
      ),
    },
    {
      accessorKey: 'actor',
      header: '操作人',
    },
    {
      accessorKey: 'action_type',
      header: '操作类型',
      cell: ({ row }) => {
        const config = ACTION_TYPE_MAP[row.original.action_type]
        return (
          <Badge variant={config.variant} className={config.className}>
            {config.label}
          </Badge>
        )
      },
    },
    {
      accessorKey: 'resource_type',
      header: '资源类型',
    },
    {
      accessorKey: 'resource_id',
      header: '资源ID',
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.resource_id}</span>
      ),
    },
    {
      accessorKey: 'ip_address',
      header: 'IP 地址',
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.ip_address}</span>
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

  const toolbar = (
    <div className="flex flex-wrap items-end gap-3">
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">开始日期</label>
        <Input
          type="date"
          value={dateFrom}
          onChange={(e) => { setDateFrom(e.target.value); setPagination((p) => ({ ...p, pageIndex: 0 })) }}
          className="h-8 w-36"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">结束日期</label>
        <Input
          type="date"
          value={dateTo}
          onChange={(e) => { setDateTo(e.target.value); setPagination((p) => ({ ...p, pageIndex: 0 })) }}
          className="h-8 w-36"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">操作人</label>
        <Select value={filterActor} onValueChange={(v) => { setFilterActor(v); setPagination((p) => ({ ...p, pageIndex: 0 })) }}>
          <SelectTrigger className="h-8 w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {ACTORS.map((a) => (
              <SelectItem key={a} value={a}>{a}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">操作类型</label>
        <Select value={filterAction} onValueChange={(v) => { setFilterAction(v); setPagination((p) => ({ ...p, pageIndex: 0 })) }}>
          <SelectTrigger className="h-8 w-24">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {ACTION_TYPES.map((a) => (
              <SelectItem key={a.value} value={a.value}>{a.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">资源类型</label>
        <Select value={filterResource} onValueChange={(v) => { setFilterResource(v); setPagination((p) => ({ ...p, pageIndex: 0 })) }}>
          <SelectTrigger className="h-8 w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {RESOURCE_TYPES.map((r) => (
              <SelectItem key={r} value={r}>{r}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )

  return (
    <>
      <DataTable
        columns={columns}
        data={paginatedData}
        totalCount={filteredData.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        toolbar={toolbar}
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
                <span>{formatDate(detailItem.timestamp)}</span>
                <span className="text-muted-foreground">操作人</span>
                <span>{detailItem.actor}</span>
                <span className="text-muted-foreground">操作类型</span>
                <span>{ACTION_TYPE_MAP[detailItem.action_type].label}</span>
                <span className="text-muted-foreground">资源</span>
                <span>{detailItem.resource_type} / {detailItem.resource_id}</span>
                <span className="text-muted-foreground">IP 地址</span>
                <span className="font-mono">{detailItem.ip_address}</span>
              </div>
              <div>
                <span className="text-muted-foreground">详细数据</span>
                <pre className="mt-1 rounded-md bg-muted p-3 text-xs overflow-auto max-h-60">
                  {JSON.stringify(detailItem.details, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}

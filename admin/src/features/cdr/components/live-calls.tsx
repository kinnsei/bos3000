import { useState, useEffect } from 'react'
import { type ColumnDef, type PaginationState } from '@tanstack/react-table'
import { DataTable } from '@/components/shared/data-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/shared/confirm-dialog'
import { PhoneOff } from 'lucide-react'
import { toast } from 'sonner'
import { useHangupCall } from '@/lib/api/hooks'
import { formatDuration } from '../utils'

interface ActiveCall {
  call_id: string
  customer: string
  a_leg_status: string
  b_leg_status: string
  started_at: string
  gateway: string
}

const LEG_STATUS_MAP: Record<string, { label: string; className: string }> = {
  ringing: { label: '振铃中', className: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' },
  answered: { label: '已接通', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  bridged: { label: '桥接中', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  failed: { label: '失败', className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
}

function LiveDuration({ startedAt }: { startedAt: string }) {
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    const start = new Date(startedAt).getTime()
    const tick = () => setElapsed(Math.floor((Date.now() - start) / 1000))
    tick()
    const id = setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [startedAt])

  return <span className="font-mono text-sm">{formatDuration(elapsed)}</span>
}

function HangupAction({ callId, onSuccess }: { callId: string; onSuccess: () => void }) {
  const hangup = useHangupCall()

  const handleConfirm = async () => {
    try {
      await hangup.mutateAsync({ call_id: callId })
      toast.success('已成功挂断通话')
      onSuccess()
    } catch {
      toast.error('挂断失败，请重试')
    }
  }

  return (
    <ConfirmDialog
      trigger={
        <Button variant="destructive" size="sm" className="h-7 px-2">
          <PhoneOff className="mr-1 h-3.5 w-3.5" />
          强制挂断
        </Button>
      }
      title="确认强制挂断通话？"
      description="这将断开双方通话。"
      confirmText="确认挂断"
      variant="danger"
      onConfirm={handleConfirm}
      disabled={hangup.isPending}
    />
  )
}

// Mock active calls data
function generateMockActiveCalls(): ActiveCall[] {
  const customers = ['客户A', '客户B', '客户C', '客户D', '客户E']
  const gateways = ['网关-北京01', '网关-上海02', '网关-广州03']
  const statuses = ['ringing', 'answered', 'bridged']
  const now = Date.now()

  return Array.from({ length: 8 }, (_, i) => ({
    call_id: `call-${crypto.randomUUID().slice(0, 8)}-${i}`,
    customer: customers[i % customers.length],
    a_leg_status: i < 2 ? 'ringing' : 'bridged',
    b_leg_status: statuses[i % statuses.length],
    started_at: new Date(now - (30 + i * 45) * 1000).toISOString(),
    gateway: gateways[i % gateways.length],
  }))
}

export function LiveCalls() {
  const [data] = useState<ActiveCall[]>(generateMockActiveCalls)
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 20 })

  const columns: ColumnDef<ActiveCall, unknown>[] = [
    {
      accessorKey: 'call_id',
      header: '呼叫ID',
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.call_id.slice(0, 12)}</span>
      ),
    },
    {
      accessorKey: 'customer',
      header: '客户',
    },
    {
      accessorKey: 'a_leg_status',
      header: 'A路状态',
      cell: ({ row }) => {
        const s = LEG_STATUS_MAP[row.original.a_leg_status] ?? { label: row.original.a_leg_status, className: '' }
        return <Badge variant="outline" className={s.className}>{s.label}</Badge>
      },
    },
    {
      accessorKey: 'b_leg_status',
      header: 'B路状态',
      cell: ({ row }) => {
        const s = LEG_STATUS_MAP[row.original.b_leg_status] ?? { label: row.original.b_leg_status, className: '' }
        return <Badge variant="outline" className={s.className}>{s.label}</Badge>
      },
    },
    {
      id: 'duration',
      header: '时长',
      cell: ({ row }) => <LiveDuration startedAt={row.original.started_at} />,
    },
    {
      accessorKey: 'gateway',
      header: '网关',
    },
    {
      id: 'actions',
      header: '操作',
      cell: ({ row }) => (
        <HangupAction
          callId={row.original.call_id}
          onSuccess={() => {}}
        />
      ),
    },
  ]

  return (
    <DataTable
      columns={columns}
      data={data}
      totalCount={data.length}
      pagination={pagination}
      onPaginationChange={setPagination}
    />
  )
}

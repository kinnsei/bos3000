import { useState, useMemo } from 'react'
import type { ColumnDef, PaginationState } from '@tanstack/react-table'
import { DataTable } from '@/components/shared/data-table'
import { ConfirmDialog } from '@/components/shared/confirm-dialog'
import { CsvUpload } from '@/components/shared/csv-upload'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Plus, Upload, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

interface BlacklistItem {
  id: string
  number: string
  reason: string
  added_by: string
  created_at: string
}

const MOCK_DATA: BlacklistItem[] = [
  { id: '1', number: '13800138001', reason: '骚扰电话投诉', added_by: 'admin', created_at: '2026-03-10T08:12:00Z' },
  { id: '2', number: '13900139002', reason: '欺诈行为', added_by: 'admin', created_at: '2026-03-09T14:30:00Z' },
  { id: '3', number: '15012345678', reason: '恶意呼叫', added_by: 'zhangsan', created_at: '2026-03-09T10:05:00Z' },
  { id: '4', number: '18611112222', reason: '违规营销', added_by: 'admin', created_at: '2026-03-08T16:45:00Z' },
  { id: '5', number: '13512348765', reason: '用户投诉多次', added_by: 'lisi', created_at: '2026-03-08T09:20:00Z' },
  { id: '6', number: '17788889999', reason: '机器人拨号', added_by: 'admin', created_at: '2026-03-07T11:30:00Z' },
  { id: '7', number: '13611223344', reason: '骚扰电话投诉', added_by: 'zhangsan', created_at: '2026-03-07T08:00:00Z' },
  { id: '8', number: '15899887766', reason: '欺诈行为', added_by: 'admin', created_at: '2026-03-06T15:10:00Z' },
  { id: '9', number: '18233445566', reason: '号码异常', added_by: 'lisi', created_at: '2026-03-06T12:25:00Z' },
  { id: '10', number: '13055667788', reason: '违规营销', added_by: 'admin', created_at: '2026-03-05T17:40:00Z' },
  { id: '11', number: '17600112233', reason: '恶意呼叫', added_by: 'zhangsan', created_at: '2026-03-05T09:55:00Z' },
  { id: '12', number: '13344556677', reason: '骚扰电话投诉', added_by: 'admin', created_at: '2026-03-04T14:15:00Z' },
  { id: '13', number: '15188990011', reason: '用户投诉多次', added_by: 'lisi', created_at: '2026-03-04T10:30:00Z' },
  { id: '14', number: '18922334455', reason: '欺诈行为', added_by: 'admin', created_at: '2026-03-03T16:00:00Z' },
  { id: '15', number: '13766778899', reason: '机器人拨号', added_by: 'zhangsan', created_at: '2026-03-03T08:45:00Z' },
]

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function BlacklistTable() {
  const [data, setData] = useState<BlacklistItem[]>(MOCK_DATA)
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [addOpen, setAddOpen] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [newNumber, setNewNumber] = useState('')
  const [newReason, setNewReason] = useState('')

  const paginatedData = useMemo(() => {
    const start = pagination.pageIndex * pagination.pageSize
    return data.slice(start, start + pagination.pageSize)
  }, [data, pagination])

  const handleRemove = (item: BlacklistItem) => {
    setData((prev) => prev.filter((d) => d.id !== item.id))
    toast.success(`已移除号码 ${item.number}`)
  }

  const handleAdd = () => {
    if (!newNumber.trim()) return
    const entry: BlacklistItem = {
      id: String(Date.now()),
      number: newNumber.trim(),
      reason: newReason.trim() || '手动添加',
      added_by: 'admin',
      created_at: new Date().toISOString(),
    }
    setData((prev) => [entry, ...prev])
    setNewNumber('')
    setNewReason('')
    setAddOpen(false)
    toast.success(`已添加号码 ${entry.number} 到黑名单`)
  }

  const handleBulkImport = (rows: Record<string, string>[]) => {
    const entries: BlacklistItem[] = rows.map((row, i) => ({
      id: String(Date.now() + i),
      number: row.number,
      reason: row.reason || '批量导入',
      added_by: 'admin',
      created_at: new Date().toISOString(),
    }))
    setData((prev) => [...entries, ...prev])
    setImportOpen(false)
    toast.success(`已导入 ${entries.length} 条记录`)
  }

  const columns: ColumnDef<BlacklistItem, unknown>[] = [
    {
      accessorKey: 'number',
      header: '号码',
      cell: ({ row }) => (
        <span className="font-mono">{row.original.number}</span>
      ),
    },
    {
      accessorKey: 'reason',
      header: '原因',
    },
    {
      accessorKey: 'added_by',
      header: '添加人',
    },
    {
      accessorKey: 'created_at',
      header: '添加时间',
      cell: ({ row }) => formatDate(row.original.created_at),
    },
    {
      id: 'actions',
      header: '操作',
      cell: ({ row }) => (
        <ConfirmDialog
          trigger={
            <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive">
              <Trash2 className="mr-1 h-4 w-4" />
              移除
            </Button>
          }
          title="移除黑名单号码"
          description={`确认从黑名单中移除号码 ${row.original.number}？`}
          confirmText="确认移除"
          variant="danger"
          onConfirm={() => handleRemove(row.original)}
        />
      ),
    },
  ]

  const toolbar = (
    <div className="flex items-center gap-2">
      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogTrigger asChild>
          <Button size="sm">
            <Plus className="mr-1 h-4 w-4" />
            添加号码
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加黑名单号码</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>电话号码</Label>
              <Input
                placeholder="请输入电话号码"
                value={newNumber}
                onChange={(e) => setNewNumber(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>原因</Label>
              <Input
                placeholder="请输入拉黑原因"
                value={newReason}
                onChange={(e) => setNewReason(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddOpen(false)}>
              取消
            </Button>
            <Button onClick={handleAdd} disabled={!newNumber.trim()}>
              确认添加
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={importOpen} onOpenChange={setImportOpen}>
        <DialogTrigger asChild>
          <Button variant="outline" size="sm">
            <Upload className="mr-1 h-4 w-4" />
            批量导入
          </Button>
        </DialogTrigger>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>批量导入黑名单</DialogTitle>
          </DialogHeader>
          <CsvUpload columns={['number', 'reason']} onImport={handleBulkImport} />
        </DialogContent>
      </Dialog>
    </div>
  )

  return (
    <DataTable
      columns={columns}
      data={paginatedData}
      totalCount={data.length}
      pagination={pagination}
      onPaginationChange={setPagination}
      toolbar={toolbar}
    />
  )
}

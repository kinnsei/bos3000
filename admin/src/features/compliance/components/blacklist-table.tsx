import { useState } from 'react'
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
import { useBlacklist, useAddBlacklist, useRemoveBlacklist } from '@/lib/api/hooks'
import type { BlacklistEntry } from '@/lib/api/client'

export function BlacklistTable() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [addOpen, setAddOpen] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [newNumber, setNewNumber] = useState('')
  const [newReason, setNewReason] = useState('')

  const { data, isLoading } = useBlacklist({ page: pagination.pageIndex + 1, limit: pagination.pageSize })
  const addMutation = useAddBlacklist()
  const removeMutation = useRemoveBlacklist()

  const entries = data?.entries ?? []
  const totalCount = data?.total ?? entries.length

  const handleRemove = async (item: BlacklistEntry) => {
    try {
      await removeMutation.mutateAsync({ id: String(item.id) })
      toast.success(`已移除号码 ${item.number}`)
    } catch {
      toast.error('移除失败')
    }
  }

  const handleAdd = async () => {
    if (!newNumber.trim()) return
    try {
      await addMutation.mutateAsync({
        number: newNumber.trim(),
        reason: newReason.trim() || '手动添加',
      })
      setNewNumber('')
      setNewReason('')
      setAddOpen(false)
      toast.success(`已添加号码 ${newNumber.trim()} 到黑名单`)
    } catch {
      toast.error('添加失败')
    }
  }

  const handleBulkImport = async (rows: Record<string, string>[]) => {
    let success = 0
    for (const row of rows) {
      try {
        await addMutation.mutateAsync({
          number: row.number,
          reason: row.reason || '批量导入',
        })
        success++
      } catch { /* continue */ }
    }
    setImportOpen(false)
    toast.success(`已导入 ${success} 条记录`)
  }

  const columns: ColumnDef<BlacklistEntry, unknown>[] = [
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
      accessorKey: 'created_by',
      header: '添加人',
      cell: ({ row }) => `用户#${row.original.created_by}`,
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
            <Button onClick={handleAdd} disabled={!newNumber.trim() || addMutation.isPending}>
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
      data={entries}
      totalCount={totalCount}
      pagination={pagination}
      onPaginationChange={setPagination}
      isLoading={isLoading}
      toolbar={toolbar}
    />
  )
}

import { useState, useCallback } from 'react'
import type { PaginationState, SortingState } from '@tanstack/react-table'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { StatCard } from '@/components/shared/stat-card'
import { CsvUpload } from '@/components/shared/csv-upload'
import { Phone, PhoneCall, PhoneOff, Upload, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { DidTable, type DIDRow } from './components/did-table'
import { DidAssignDialog } from './components/did-assign-dialog'
import { useDIDs, useImportDIDs } from '@/lib/api/hooks'

export default function DID() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [sorting, setSorting] = useState<SortingState>([])
  const [assignFilter, setAssignFilter] = useState('all')
  const [customerFilter, setCustomerFilter] = useState('all')

  const [importOpen, setImportOpen] = useState(false)
  const [assignOpen, setAssignOpen] = useState(false)
  const [selectedDid, setSelectedDid] = useState<DIDRow | null>(null)

  const { data, isLoading } = useDIDs({ page: pagination.pageIndex + 1, limit: pagination.pageSize })
  const importMutation = useImportDIDs()

  const dids: DIDRow[] = (data?.dids ?? []).map((d) => ({
    id: String(d.id),
    number: d.number,
    status: d.status === 'available' ? 'active' as const : 'inactive' as const,
    assigned_to: d.assigned_user_id ? String(d.assigned_user_id) : '',
    assigned_name: '',
    pool_type: 'dedicated' as const,
    created_at: d.created_at,
  }))

  const totalCount = data?.total ?? 0
  const assignedCount = dids.filter((d) => d.assigned_to).length
  const unassignedCount = totalCount - assignedCount

  const handleAssign = useCallback((did: DIDRow) => {
    setSelectedDid(did)
    setAssignOpen(true)
  }, [])

  const handleUnassign = useCallback((did: DIDRow) => {
    toast.success(`${did.number} 已取消分配`)
  }, [])

  const handleToggle = useCallback((did: DIDRow) => {
    const action = did.status === 'active' ? '停用' : '启用'
    toast.success(`${did.number} 已${action}`)
  }, [])

  const handleAssignSubmit = useCallback(
    (_didId: string, _customerId: string, poolType: 'dedicated' | 'shared') => {
      const poolLabel = poolType === 'dedicated' ? '专属' : '共享'
      toast.success(`号码已分配（${poolLabel}）`)
    },
    [],
  )

  const handleImport = useCallback(async (rows: Record<string, string>[]) => {
    const numbers = rows.map((r) => r.number).filter(Boolean)
    if (numbers.length === 0) {
      toast.error('没有有效号码')
      return
    }
    try {
      const result = await importMutation.mutateAsync({ numbers })
      toast.success(`成功导入 ${result.imported} 个号码`)
      setImportOpen(false)
    } catch {
      toast.error('导入失败')
    }
  }, [importMutation])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">DID 管理</h1>
          <p className="text-sm text-muted-foreground">号码列表、批量导入和分配管理</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setImportOpen(true)}>
            <Upload className="mr-1.5 h-4 w-4" />
            批量导入
          </Button>
          <Button onClick={() => toast.info('添加号码功能开发中')}>
            <Plus className="mr-1.5 h-4 w-4" />
            添加号码
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard title="总号码数" value={totalCount} icon={Phone} />
        <StatCard title="已分配" value={assignedCount} icon={PhoneCall} />
        <StatCard title="未分配" value={unassignedCount} icon={PhoneOff} />
      </div>

      <DidTable
        data={dids}
        totalCount={totalCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        sorting={sorting}
        onSortingChange={setSorting}
        isLoading={isLoading}
        assignFilter={assignFilter}
        onAssignFilterChange={setAssignFilter}
        customerFilter={customerFilter}
        onCustomerFilterChange={setCustomerFilter}
        customers={[]}
        onAssign={handleAssign}
        onUnassign={handleUnassign}
        onToggle={handleToggle}
      />

      <Dialog open={importOpen} onOpenChange={setImportOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>批量导入 DID 号码</DialogTitle>
          </DialogHeader>
          <CsvUpload columns={['number', 'pool_type']} onImport={handleImport} />
        </DialogContent>
      </Dialog>

      <DidAssignDialog
        open={assignOpen}
        onOpenChange={setAssignOpen}
        did={selectedDid ? { id: selectedDid.id, number: selectedDid.number } : null}
        onSubmit={handleAssignSubmit}
      />
    </div>
  )
}

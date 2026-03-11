import { useState, useCallback, useMemo } from 'react'
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
import { ConfirmDialog } from '@/components/shared/confirm-dialog'
import { Phone, PhoneCall, PhoneOff, Upload, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { DidTable, type DIDRow } from './components/did-table'
import { DidAssignDialog } from './components/did-assign-dialog'

// Mock customers for filters and assignment
const mockCustomers = [
  { id: 'c1', name: '示例科技有限公司' },
  { id: 'c2', name: '通达通信' },
  { id: 'c3', name: '汇联科技' },
  { id: 'c4', name: '星辰网络' },
  { id: 'c5', name: '云桥通讯' },
  { id: 'c6', name: '盛达科技' },
  { id: 'c7', name: '明远通讯' },
]

// Generate ~30 mock DIDs
const mockDIDs: DIDRow[] = Array.from({ length: 30 }, (_, i) => {
  const assigned = i % 3 !== 0
  const customer = assigned ? mockCustomers[i % mockCustomers.length] : null
  return {
    id: `did-${i + 1}`,
    number: `+8610${String(80000000 + i * 1111).slice(0, 8)}`,
    status: i % 7 === 0 ? 'inactive' as const : 'active' as const,
    assigned_to: customer?.id ?? '',
    assigned_name: customer?.name ?? '',
    pool_type: i % 4 === 0 ? 'shared' as const : 'dedicated' as const,
    created_at: new Date(2026, 0, 1 + i).toISOString(),
  }
})

export default function DID() {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [sorting, setSorting] = useState<SortingState>([])
  const [assignFilter, setAssignFilter] = useState('all')
  const [customerFilter, setCustomerFilter] = useState('all')

  const [importOpen, setImportOpen] = useState(false)
  const [assignOpen, setAssignOpen] = useState(false)
  const [selectedDid, setSelectedDid] = useState<DIDRow | null>(null)

  // Filter data
  const filtered = useMemo(() => {
    let result = mockDIDs
    if (assignFilter === 'assigned') result = result.filter((d) => d.assigned_to)
    if (assignFilter === 'unassigned') result = result.filter((d) => !d.assigned_to)
    if (customerFilter !== 'all') result = result.filter((d) => d.assigned_to === customerFilter)
    return result
  }, [assignFilter, customerFilter])

  // Stats
  const totalCount = mockDIDs.length
  const assignedCount = mockDIDs.filter((d) => d.assigned_to).length
  const unassignedCount = totalCount - assignedCount

  // Paginate
  const paged = useMemo(() => {
    const start = pagination.pageIndex * pagination.pageSize
    return filtered.slice(start, start + pagination.pageSize)
  }, [filtered, pagination])

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
    (didId: string, customerId: string, poolType: 'dedicated' | 'shared') => {
      const customer = mockCustomers.find((c) => c.id === customerId)
      const poolLabel = poolType === 'dedicated' ? '专属' : '共享'
      toast.success(`已将号码分配给 ${customer?.name}（${poolLabel}）`)
    },
    [],
  )

  const handleImport = useCallback(async (data: Record<string, string>[]) => {
    toast.success(`成功导入 ${data.length} 个号码`)
    setImportOpen(false)
  }, [])

  return (
    <div className="space-y-6">
      {/* Header */}
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

      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard title="总号码数" value={totalCount} icon={Phone} />
        <StatCard title="已分配" value={assignedCount} icon={PhoneCall} />
        <StatCard title="未分配" value={unassignedCount} icon={PhoneOff} />
      </div>

      {/* Table */}
      <DidTable
        data={paged}
        totalCount={filtered.length}
        pagination={pagination}
        onPaginationChange={setPagination}
        sorting={sorting}
        onSortingChange={setSorting}
        isLoading={false}
        assignFilter={assignFilter}
        onAssignFilterChange={setAssignFilter}
        customerFilter={customerFilter}
        onCustomerFilterChange={setCustomerFilter}
        customers={mockCustomers}
        onAssign={handleAssign}
        onUnassign={handleUnassign}
        onToggle={handleToggle}
      />

      {/* Import Dialog */}
      <Dialog open={importOpen} onOpenChange={setImportOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>批量导入 DID 号码</DialogTitle>
          </DialogHeader>
          <CsvUpload columns={['number', 'pool_type']} onImport={handleImport} />
        </DialogContent>
      </Dialog>

      {/* Assign Dialog */}
      <DidAssignDialog
        open={assignOpen}
        onOpenChange={setAssignOpen}
        did={selectedDid ? { id: selectedDid.id, number: selectedDid.number } : null}
        onSubmit={handleAssignSubmit}
      />
    </div>
  )
}

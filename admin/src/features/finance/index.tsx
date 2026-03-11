import { useState, useCallback, useMemo } from 'react'
import type { ColumnDef, PaginationState, SortingState } from '@tanstack/react-table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/shared/data-table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Plus, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { TransactionTable } from './components/transaction-table'
import { RatePlanSheet, type RatePlanFormData, type RatePlanForEdit } from './components/rate-plan-sheet'
import { ProfitAnalysis } from './components/profit-analysis'

interface RatePlan {
  id: string
  name: string
  description?: string
  rate_per_minute: number
  billing_increment: number
  connection_fee: number
  b_leg_rates?: { prefix: string; rate: number }[]
  min_billing_duration?: number
}

const mockRatePlans: RatePlan[] = [
  { id: 'rp_01', name: '标准套餐', description: '适用于普通客户', rate_per_minute: 0.15, billing_increment: 6, connection_fee: 0.05, b_leg_rates: [{ prefix: '010', rate: 0.08 }], min_billing_duration: 6 },
  { id: 'rp_02', name: 'VIP套餐', description: '大客户专享', rate_per_minute: 0.10, billing_increment: 1, connection_fee: 0, b_leg_rates: [{ prefix: '010', rate: 0.05 }, { prefix: '021', rate: 0.06 }], min_billing_duration: 0 },
  { id: 'rp_03', name: '国际线路', description: '国际呼叫', rate_per_minute: 0.80, billing_increment: 60, connection_fee: 0.50, b_leg_rates: [], min_billing_duration: 60 },
  { id: 'rp_04', name: '经济套餐', rate_per_minute: 0.20, billing_increment: 6, connection_fee: 0.10, b_leg_rates: [], min_billing_duration: 6 },
]

const ratePlanColumns: ColumnDef<RatePlan, unknown>[] = [
  { accessorKey: 'name', header: '名称' },
  {
    accessorKey: 'rate_per_minute',
    header: '费率/分钟',
    cell: ({ row }) => <span className="font-mono">¥{row.original.rate_per_minute.toFixed(2)}</span>,
  },
  {
    accessorKey: 'billing_increment',
    header: '计费增量',
    cell: ({ row }) => `${row.original.billing_increment}秒`,
  },
  {
    accessorKey: 'connection_fee',
    header: '接通费',
    cell: ({ row }) => <span className="font-mono">¥{row.original.connection_fee.toFixed(2)}</span>,
  },
]

export default function Finance() {
  const [rpPagination, setRpPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [rpSorting, setRpSorting] = useState<SortingState>([])
  const [sheetOpen, setSheetOpen] = useState(false)
  const [editingPlan, setEditingPlan] = useState<RatePlanForEdit | undefined>(undefined)

  const handleCreatePlan = useCallback(() => {
    setEditingPlan(undefined)
    setSheetOpen(true)
  }, [])

  const handleEditPlan = useCallback((plan: RatePlan) => {
    setEditingPlan(plan)
    setSheetOpen(true)
  }, [])

  const handleDeletePlan = useCallback((plan: RatePlan) => {
    toast.success(`已删除模板: ${plan.name}`)
  }, [])

  const handleSubmitPlan = useCallback(async (_data: RatePlanFormData) => {
    toast.success(editingPlan ? '模板已更新' : '模板已创建')
  }, [editingPlan])

  const rpColumnsWithActions = useMemo<ColumnDef<RatePlan, unknown>[]>(() => [
    ...ratePlanColumns,
    {
      id: 'actions',
      header: '操作',
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-8 w-8">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => handleEditPlan(row.original)}>
              <Pencil className="mr-2 h-3.5 w-3.5" />
              编辑
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={() => handleDeletePlan(row.original)}
            >
              <Trash2 className="mr-2 h-3.5 w-3.5" />
              删除
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ], [handleEditPlan, handleDeletePlan])

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">财务中心</h1>
        <p className="text-sm text-muted-foreground">对账、流水、费率模板和毛利分析</p>
      </div>

      <Tabs defaultValue="transactions">
        <TabsList>
          <TabsTrigger value="transactions">交易流水</TabsTrigger>
          <TabsTrigger value="rate-plans">费率模板</TabsTrigger>
          <TabsTrigger value="profit">毛利分析</TabsTrigger>
        </TabsList>

        <TabsContent value="transactions" className="mt-4">
          <TransactionTable />
        </TabsContent>

        <TabsContent value="rate-plans" className="mt-4">
          <DataTable
            columns={rpColumnsWithActions}
            data={mockRatePlans}
            totalCount={mockRatePlans.length}
            pagination={rpPagination}
            onPaginationChange={setRpPagination}
            sorting={rpSorting}
            onSortingChange={setRpSorting}
            isLoading={false}
            toolbar={
              <div className="flex justify-end">
                <Button onClick={handleCreatePlan}>
                  <Plus className="mr-1.5 h-4 w-4" />
                  新建模板
                </Button>
              </div>
            }
          />
          <RatePlanSheet
            open={sheetOpen}
            onOpenChange={setSheetOpen}
            ratePlan={editingPlan}
            onSubmit={handleSubmitPlan}
          />
        </TabsContent>

        <TabsContent value="profit" className="mt-4">
          <ProfitAnalysis />
        </TabsContent>
      </Tabs>
    </div>
  )
}

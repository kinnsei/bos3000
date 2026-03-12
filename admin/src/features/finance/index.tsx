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
import { useRatePlans, useCreateRatePlan, useUpdateRatePlan, useDeleteRatePlan } from '@/lib/api/hooks'
import type { RatePlan } from '@/lib/api/client'

const ratePlanColumns: ColumnDef<RatePlan, unknown>[] = [
  { accessorKey: 'name', header: '名称' },
  {
    accessorKey: 'mode',
    header: '模式',
  },
  {
    accessorKey: 'uniform_a_rate',
    header: 'A路费率(分/分钟)',
    cell: ({ row }) => <span className="font-mono">{row.original.uniform_a_rate}</span>,
  },
  {
    accessorKey: 'uniform_b_rate',
    header: 'B路费率(分/分钟)',
    cell: ({ row }) => <span className="font-mono">{row.original.uniform_b_rate}</span>,
  },
  {
    accessorKey: 'description',
    header: '描述',
  },
]

export default function Finance() {
  const [rpPagination, setRpPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [rpSorting, setRpSorting] = useState<SortingState>([])
  const [sheetOpen, setSheetOpen] = useState(false)
  const [editingPlan, setEditingPlan] = useState<RatePlanForEdit | undefined>(undefined)

  const { data: ratePlanData, isLoading } = useRatePlans()
  const createMutation = useCreateRatePlan()
  const updateMutation = useUpdateRatePlan()
  const deleteMutation = useDeleteRatePlan()

  const ratePlans = ratePlanData?.plans ?? []

  const handleCreatePlan = useCallback(() => {
    setEditingPlan(undefined)
    setSheetOpen(true)
  }, [])

  const handleEditPlan = useCallback((plan: RatePlan) => {
    setEditingPlan({
      id: String(plan.id),
      name: plan.name,
      description: plan.description,
      rate_per_minute: plan.uniform_a_rate / 100,
      billing_increment: 6,
      connection_fee: 0,
    })
    setSheetOpen(true)
  }, [])

  const handleDeletePlan = useCallback(async (plan: RatePlan) => {
    try {
      await deleteMutation.mutateAsync({ id: String(plan.id) })
      toast.success(`已删除模板: ${plan.name}`)
    } catch {
      toast.error('删除失败')
    }
  }, [deleteMutation])

  const handleSubmitPlan = useCallback(async (data: RatePlanFormData) => {
    try {
      if (editingPlan) {
        await updateMutation.mutateAsync({
          id: editingPlan.id,
          name: data.name,
          mode: 'uniform',
          uniform_a_rate: Math.round(data.rate_per_minute * 100),
          uniform_b_rate: Math.round(data.rate_per_minute * 100),
          description: data.description,
        })
      } else {
        await createMutation.mutateAsync({
          name: data.name,
          mode: 'uniform',
          uniform_a_rate: Math.round(data.rate_per_minute * 100),
          uniform_b_rate: Math.round(data.rate_per_minute * 100),
          description: data.description,
          created_at: '',
          updated_at: '',
        })
      }
      toast.success(editingPlan ? '模板已更新' : '模板已创建')
    } catch {
      toast.error('操作失败')
    }
  }, [editingPlan, createMutation, updateMutation])

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
            data={ratePlans}
            totalCount={ratePlans.length}
            pagination={rpPagination}
            onPaginationChange={setRpPagination}
            sorting={rpSorting}
            onSortingChange={setRpSorting}
            isLoading={isLoading}
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

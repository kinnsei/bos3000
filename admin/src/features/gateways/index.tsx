import { useState, useMemo, useCallback } from 'react'
import type { PaginationState, SortingState } from '@tanstack/react-table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
import { toast } from 'sonner'
import { useGateways, useToggleGateway } from '@/lib/api/hooks'
import { GatewayTable } from './components/gateway-table'
import { GatewayConfigSheet } from './components/gateway-config-sheet'
import { TestOriginateDialog } from './components/test-originate-dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import type { Gateway, TestOriginateResult } from '@/lib/api/client'

export default function Gateways() {
  const [activeTab, setActiveTab] = useState<'a_leg' | 'b_leg'>('a_leg')

  // Pagination per tab
  const [aLegPagination, setALegPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 20 })
  const [bLegPagination, setBLegPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 20 })
  const [aLegSorting, setALegSorting] = useState<SortingState>([])
  const [bLegSorting, setBLegSorting] = useState<SortingState>([])

  // Sheet state
  const [sheetOpen, setSheetOpen] = useState(false)
  const [editingGateway, setEditingGateway] = useState<Gateway | undefined>()
  const [sheetType, setSheetType] = useState<'a_leg' | 'b_leg'>('a_leg')

  // Test dialog state
  const [testDialogOpen, setTestDialogOpen] = useState(false)
  const [testingGateway, setTestingGateway] = useState<Gateway | null>(null)

  // Toggle confirm state
  const [toggleTarget, setToggleTarget] = useState<Gateway | null>(null)

  // Data
  const { data, isLoading } = useGateways()
  const toggleMutation = useToggleGateway()

  const allGateways = data?.gateways ?? []
  const aLegGateways = useMemo(() => allGateways.filter((gw) => gw.type === 'a_leg'), [allGateways])
  const bLegGateways = useMemo(() => allGateways.filter((gw) => gw.type === 'b_leg'), [allGateways])

  const handleAddGateway = () => {
    setEditingGateway(undefined)
    setSheetType(activeTab)
    setSheetOpen(true)
  }

  const handleEdit = useCallback((gateway: Gateway) => {
    setEditingGateway(gateway)
    setSheetType(gateway.type)
    setSheetOpen(true)
  }, [])

  const handleToggle = useCallback((gateway: Gateway) => {
    setToggleTarget(gateway)
  }, [])

  const confirmToggle = async () => {
    if (!toggleTarget) return
    const enabling = !toggleTarget.enabled
    try {
      await toggleMutation.mutateAsync({
        gateway_id: toggleTarget.id,
        enabled: enabling,
      })
      toast.success(enabling ? '网关已上线' : '网关已下线')
    } catch {
      toast.error('操作失败，请重试')
    }
    setToggleTarget(null)
  }

  const handleSwitchToggle = useCallback((gateway: Gateway, enabled: boolean) => {
    toggleMutation.mutate(
      { gateway_id: gateway.id, enabled },
      {
        onSuccess: () => toast.success(enabled ? '网关已启用' : '网关已禁用'),
        onError: () => toast.error('操作失败，请重试'),
      },
    )
  }, [toggleMutation])

  const handleTest = useCallback((gateway: Gateway) => {
    setTestingGateway(gateway)
    setTestDialogOpen(true)
  }, [])

  const handleTestOriginate = async (
    _gatewayId: string,
    _phoneNumber: string,
  ): Promise<TestOriginateResult> => {
    // TODO: Wire to real API
    return { success: true, message: '测试呼叫成功', duration_ms: 1200 }
  }

  const handleSubmitConfig = async (formData: Record<string, unknown>) => {
    // TODO: Wire to create/update gateway API
    console.log('submit gateway config', formData)
    toast.success(editingGateway ? '网关配置已更新' : '网关已创建')
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">网关管理</h1>
          <p className="text-sm text-muted-foreground">A/B 路网关池和健康状态管理</p>
        </div>
        <Button onClick={handleAddGateway}>
          <Plus className="mr-1.5 h-4 w-4" />
          添加网关
        </Button>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'a_leg' | 'b_leg')}>
        <TabsList>
          <TabsTrigger value="a_leg">A 路网关池</TabsTrigger>
          <TabsTrigger value="b_leg">B 路网关池</TabsTrigger>
        </TabsList>

        <TabsContent value="a_leg" className="mt-4">
          <GatewayTable
            data={aLegGateways}
            totalCount={aLegGateways.length}
            gatewayType="a_leg"
            pagination={aLegPagination}
            onPaginationChange={setALegPagination}
            sorting={aLegSorting}
            onSortingChange={setALegSorting}
            isLoading={isLoading}
            allGateways={allGateways}
            onEdit={handleEdit}
            onToggle={handleToggle}
            onTest={handleTest}
            onSwitchToggle={handleSwitchToggle}
          />
        </TabsContent>

        <TabsContent value="b_leg" className="mt-4">
          <GatewayTable
            data={bLegGateways}
            totalCount={bLegGateways.length}
            gatewayType="b_leg"
            pagination={bLegPagination}
            onPaginationChange={setBLegPagination}
            sorting={bLegSorting}
            onSortingChange={setBLegSorting}
            isLoading={isLoading}
            allGateways={allGateways}
            onEdit={handleEdit}
            onToggle={handleToggle}
            onTest={handleTest}
            onSwitchToggle={handleSwitchToggle}
          />
        </TabsContent>
      </Tabs>

      <GatewayConfigSheet
        open={sheetOpen}
        onOpenChange={setSheetOpen}
        gateway={editingGateway}
        gatewayType={sheetType}
        otherGateways={allGateways}
        onSubmit={handleSubmitConfig}
      />

      <TestOriginateDialog
        open={testDialogOpen}
        onOpenChange={setTestDialogOpen}
        gateway={testingGateway}
        onTest={handleTestOriginate}
      />

      <AlertDialog open={!!toggleTarget} onOpenChange={(open) => { if (!open) setToggleTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {toggleTarget?.health_status === 'up' ? '确认下线网关' : '确认上线网关'}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {toggleTarget?.health_status === 'up'
                ? `确认将网关 ${toggleTarget?.name} 下线？下线后该网关将不再接收新呼叫。`
                : `确认将网关 ${toggleTarget?.name} 上线？`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmToggle}
              className="bg-orange-600 text-white hover:bg-orange-700"
            >
              确认
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

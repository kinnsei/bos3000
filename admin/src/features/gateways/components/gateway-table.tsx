import { useMemo } from 'react'
import type { ColumnDef, PaginationState, SortingState, OnChangeFn } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { DataTable } from '@/components/shared/data-table'
import { MoreHorizontal } from 'lucide-react'
import { GatewayHealthBadge } from './gateway-health-badge'
import type { Gateway } from '@/lib/api/client'

interface GatewayTableProps {
  data: Gateway[]
  totalCount: number
  gatewayType: 'a_leg' | 'b_leg'
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  sorting?: SortingState
  onSortingChange?: OnChangeFn<SortingState>
  isLoading?: boolean
  allGateways?: Gateway[]
  onEdit: (gateway: Gateway) => void
  onToggle: (gateway: Gateway) => void
  onTest: (gateway: Gateway) => void
  onSwitchToggle: (gateway: Gateway, enabled: boolean) => void
}

export function GatewayTable({
  data,
  totalCount,
  gatewayType,
  pagination,
  onPaginationChange,
  sorting,
  onSortingChange,
  isLoading,
  allGateways = [],
  onEdit,
  onToggle,
  onTest,
  onSwitchToggle,
}: GatewayTableProps) {
  const failoverMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const gw of allGateways) {
      map.set(gw.id, gw.name)
    }
    return map
  }, [allGateways])

  const columns = useMemo<ColumnDef<Gateway, unknown>[]>(() => {
    const cols: ColumnDef<Gateway, unknown>[] = [
      {
        accessorKey: 'name',
        header: '名称',
        enableSorting: false,
      },
      {
        id: 'sip_address',
        header: 'SIP 地址',
        enableSorting: false,
        cell: ({ row }) => (
          <span className="font-mono text-sm">
            {row.original.host}:{row.original.port}
          </span>
        ),
      },
      {
        accessorKey: 'type',
        header: '类型',
        enableSorting: false,
        cell: ({ row }) => (
          <Badge variant="outline">
            {row.original.type === 'a_leg' ? 'A 路' : 'B 路'}
          </Badge>
        ),
      },
      {
        accessorKey: 'status',
        header: '健康状态',
        enableSorting: false,
        cell: ({ row }) => <GatewayHealthBadge status={row.original.status} />,
      },
    ]

    if (gatewayType === 'a_leg') {
      cols.push({
        accessorKey: 'weight',
        header: '权重',
        cell: ({ row }) => row.original.weight,
      })
    }

    if (gatewayType === 'b_leg') {
      cols.push(
        {
          accessorKey: 'prefix',
          header: '前缀',
          enableSorting: false,
          cell: ({ row }) => (
            <span className="font-mono text-sm">
              {row.original.prefix || '-'}
            </span>
          ),
        },
        {
          id: 'failover',
          header: '容灾网关',
          enableSorting: false,
          cell: ({ row }) => {
            const fid = row.original.failover_gateway_id
            return fid ? failoverMap.get(fid) ?? fid : '-'
          },
        },
      )
    }

    cols.push(
      {
        id: 'enabled',
        header: '启用状态',
        enableSorting: false,
        cell: ({ row }) => (
          <Switch
            checked={row.original.status !== 'disabled'}
            onCheckedChange={(checked) => onSwitchToggle(row.original, checked)}
          />
        ),
      },
      {
        id: 'concurrency',
        header: '当前并发/最大并发',
        enableSorting: false,
        cell: ({ row }) => (
          <span className="font-mono text-sm">
            {row.original.concurrent_calls} / {row.original.max_concurrent}
          </span>
        ),
      },
      {
        id: 'actions',
        header: '操作',
        enableSorting: false,
        cell: ({ row }) => {
          const gw = row.original
          const isUp = gw.status === 'up'
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onEdit(gw)}>
                  编辑配置
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onTest(gw)}>
                  测试呼叫
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => onToggle(gw)}>
                  {isUp ? '下线' : '上线'}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      },
    )

    return cols
  }, [gatewayType, failoverMap, onEdit, onToggle, onTest, onSwitchToggle])

  return (
    <DataTable
      columns={columns}
      data={data}
      totalCount={totalCount}
      pagination={pagination}
      onPaginationChange={onPaginationChange}
      sorting={sorting}
      onSortingChange={onSortingChange}
      isLoading={isLoading}
    />
  )
}

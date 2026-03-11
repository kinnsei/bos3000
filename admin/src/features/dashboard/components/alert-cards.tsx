import { cn } from '@/lib/utils'
import type { Alert } from '@/lib/api/client'

interface AlertCardsProps {
  alerts?: Alert[]
  bridgeRate?: number
  wastageRate?: number
  isLoading: boolean
}

const BRIDGE_THRESHOLD = 60
const WASTAGE_THRESHOLD = 15

interface AlertCardDef {
  type: Alert['type']
  label: string
  getMessage: (alerts: Alert[], bridgeRate: number, wastageRate: number) => string
  getStatus: (alerts: Alert[], bridgeRate: number, wastageRate: number) => 'ok' | 'warning' | 'critical'
}

const alertDefs: AlertCardDef[] = [
  {
    type: 'bridge_rate_low',
    label: '桥接成功率',
    getMessage: (_a, rate) =>
      rate < BRIDGE_THRESHOLD ? `告警: ${rate.toFixed(1)}% < ${BRIDGE_THRESHOLD}%` : '正常',
    getStatus: (_a, rate) => (rate < BRIDGE_THRESHOLD ? 'critical' : 'ok'),
  },
  {
    type: 'balance_low',
    label: '余额不足客户',
    getMessage: (alerts) => {
      const count = alerts.filter((a) => a.type === 'balance_low').length
      return count > 0 ? `${count} 个客户余额不足` : '正常'
    },
    getStatus: (alerts) =>
      alerts.some((a) => a.type === 'balance_low') ? 'warning' : 'ok',
  },
  {
    type: 'gateway_down',
    label: '网关状态',
    getMessage: (alerts) => {
      const count = alerts.filter((a) => a.type === 'gateway_down').length
      return count > 0 ? `${count} 个网关离线` : '全部在线'
    },
    getStatus: (alerts) =>
      alerts.some((a) => a.type === 'gateway_down') ? 'critical' : 'ok',
  },
  {
    type: 'wastage_high',
    label: '损耗率',
    getMessage: (_a, _b, rate) =>
      rate > WASTAGE_THRESHOLD ? `告警: ${rate.toFixed(1)}% > ${WASTAGE_THRESHOLD}%` : '正常',
    getStatus: (_a, _b, rate) => (rate > WASTAGE_THRESHOLD ? 'critical' : 'ok'),
  },
]

const statusColors = {
  ok: 'bg-green-500',
  warning: 'bg-yellow-500',
  critical: 'bg-red-500',
}

export function AlertCards({ alerts = [], bridgeRate = 100, wastageRate = 0, isLoading }: AlertCardsProps) {
  if (isLoading) return null

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {alertDefs.map((def) => {
        const status = def.getStatus(alerts, bridgeRate, wastageRate)
        const message = def.getMessage(alerts, bridgeRate, wastageRate)
        return (
          <div
            key={def.type}
            className="flex items-center gap-3 rounded-lg border p-3"
          >
            <span
              className={cn(
                'h-2.5 w-2.5 shrink-0 rounded-full',
                statusColors[status],
                status === 'critical' && 'animate-pulse',
              )}
            />
            <div className="min-w-0">
              <p className="text-xs text-muted-foreground">{def.label}</p>
              <p className={cn(
                'text-sm font-medium truncate',
                status === 'critical' && 'text-red-600 dark:text-red-400',
                status === 'warning' && 'text-yellow-600 dark:text-yellow-400',
              )}>
                {message}
              </p>
            </div>
          </div>
        )
      })}
    </div>
  )
}

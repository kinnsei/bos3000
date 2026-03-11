import { Badge } from '@/components/ui/badge'

const statusConfig = {
  up: {
    label: '在线',
    dotClass: 'bg-green-500',
    badgeClass: 'bg-green-500/10 text-green-600',
  },
  down: {
    label: '异常',
    dotClass: 'bg-red-500 animate-pulse',
    badgeClass: 'bg-red-500/10 text-red-600',
  },
  disabled: {
    label: '已禁用',
    dotClass: 'bg-gray-400',
    badgeClass: 'bg-gray-500/10 text-gray-500',
  },
} as const

interface GatewayHealthBadgeProps {
  status: 'up' | 'down' | 'disabled'
}

export function GatewayHealthBadge({ status }: GatewayHealthBadgeProps) {
  const config = statusConfig[status]
  return (
    <Badge className={config.badgeClass}>
      <span className={`mr-1.5 inline-block h-2 w-2 rounded-full ${config.dotClass}`} />
      {config.label}
    </Badge>
  )
}

import { Badge } from '@/components/ui/badge'

const statusConfig = {
  up: {
    label: '在线',
    dotClass: 'bg-green-500',
    badgeClass: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  },
  down: {
    label: '异常',
    dotClass: 'bg-red-500 animate-pulse',
    badgeClass: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  },
  disabled: {
    label: '已禁用',
    dotClass: 'bg-gray-400',
    badgeClass: 'bg-gray-100 text-gray-700 dark:bg-gray-800/50 dark:text-gray-400',
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

import { PhoneCall, TrendingUp, TrendingDown, CheckCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { NumberTicker } from '@/components/ui/number-ticker'
import type { OverviewResponse } from '@/lib/api/client'

interface OverviewStatsProps {
  data?: OverviewResponse
  isLoading: boolean
}

function formatCNY(value: number): string {
  return `¥${value.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`
}

export function OverviewStats({ data, isLoading }: OverviewStatsProps) {
  const stats = [
    {
      title: '实时并发',
      value: data?.concurrent_calls ?? 0,
      icon: PhoneCall,
      format: 'number' as const,
    },
    {
      title: '今日收入',
      value: data?.today_revenue ?? 0,
      icon: TrendingUp,
      format: 'cny' as const,
    },
    {
      title: '今日损耗',
      value: data?.today_wastage ?? 0,
      icon: TrendingDown,
      format: 'cny' as const,
      tint: 'text-red-600 dark:text-red-400',
    },
    {
      title: '桥接成功率',
      value: data?.bridge_success_rate ?? 0,
      icon: CheckCircle,
      format: 'percent' as const,
    },
  ]

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      {stats.map((stat) => (
        <Card key={stat.title}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {stat.title}
            </CardTitle>
            <stat.icon className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              <div className={`text-2xl font-bold ${stat.tint ?? ''}`}>
                {stat.format === 'cny' && '¥'}
                <NumberTicker
                  value={stat.value}
                  decimalPlaces={stat.format === 'cny' ? 2 : stat.format === 'percent' ? 1 : 0}
                />
                {stat.format === 'percent' && '%'}
              </div>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export { formatCNY }

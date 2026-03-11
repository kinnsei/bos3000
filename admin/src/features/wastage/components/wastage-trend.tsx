import { useState } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { CHART_COLORS } from '@/lib/theme/chart-theme'
import { useWastageTrend } from '@/lib/api/hooks'

type Period = 'day' | 'week' | 'month'

export function WastageTrend() {
  const [period, setPeriod] = useState<Period>('week')
  const { data, isLoading } = useWastageTrend(period)

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">损耗率趋势</CardTitle>
        <Tabs value={period} onValueChange={(v) => setPeriod(v as Period)}>
          <TabsList className="h-8">
            <TabsTrigger value="day" className="text-xs px-2">日</TabsTrigger>
            <TabsTrigger value="week" className="text-xs px-2">周</TabsTrigger>
            <TabsTrigger value="month" className="text-xs px-2">月</TabsTrigger>
          </TabsList>
        </Tabs>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[350px]" />
        ) : (
          <div className="h-[350px]">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="date" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} unit="%" />
                <Tooltip
                  contentStyle={{ background: 'var(--color-card)', border: '1px solid var(--color-border)' }}
                  formatter={(value, name) => [
                    `${Number(value).toFixed(1)}%`,
                    name === 'wastage_rate' ? '损耗率' : '桥接成功率',
                  ]}
                  labelFormatter={(label) => `日期: ${label}`}
                />
                <Legend formatter={(value) => (value === 'wastage_rate' ? '损耗率' : '桥接成功率')} />
                <Line
                  type="monotone"
                  dataKey="wastage_rate"
                  stroke={CHART_COLORS.danger}
                  strokeWidth={2}
                  dot={false}
                />
                <Line
                  type="monotone"
                  dataKey="bridge_rate"
                  stroke={CHART_COLORS.primary}
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { CHART_COLORS_ARRAY } from '@/lib/theme/chart-theme'
import { useWastageDistribution } from '@/lib/api/hooks'

const FAILURE_LABELS: Record<string, string> = {
  a_connected_b_failed: 'A 通 B 失败',
  bridge_broken_early: '桥接中断',
  b_no_answer: 'B 路无应答',
  b_busy: 'B 路忙',
  b_rejected: 'B 路拒接',
  other: '其他',
}

export function FailureDistribution() {
  const { data, isLoading } = useWastageDistribution()

  const chartData = (data ?? []).map((item) => ({
    ...item,
    label: FAILURE_LABELS[item.reason] ?? item.reason,
  }))

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">B 路失败原因分布</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[400px]" />
        ) : (
          <div className="h-[400px]">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={chartData}
                  dataKey="count"
                  nameKey="label"
                  cx="50%"
                  cy="45%"
                  outerRadius={120}
                  innerRadius={60}
                  paddingAngle={2}
                  label={({ name, percent }) =>
                    `${name ?? ''} ${((percent ?? 0) * 100).toFixed(0)}%`
                  }
                  labelLine={{ strokeWidth: 1 }}
                >
                  {chartData.map((_, i) => (
                    <Cell key={i} fill={CHART_COLORS_ARRAY[i % CHART_COLORS_ARRAY.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ background: 'var(--color-card)', border: '1px solid var(--color-border)' }}
                  formatter={(value, name) => [`${value} 次`, String(name)]}
                />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

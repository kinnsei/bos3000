import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useWastageRanking } from '@/lib/api/hooks'

const GRADIENT_COLORS = [
  'hsl(0, 75%, 55%)',
  'hsl(5, 72%, 57%)',
  'hsl(10, 70%, 59%)',
  'hsl(15, 68%, 61%)',
  'hsl(20, 65%, 63%)',
  'hsl(25, 62%, 65%)',
  'hsl(30, 60%, 67%)',
  'hsl(32, 58%, 69%)',
  'hsl(34, 55%, 71%)',
  'hsl(35, 52%, 73%)',
]

export function CustomerRanking() {
  const { data, isLoading } = useWastageRanking()

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">客户损耗 TOP 10</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[400px]" />
        ) : (
          <div className="h-[400px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={data} layout="vertical" margin={{ left: 20 }}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis type="number" tick={{ fontSize: 12 }} />
                <YAxis
                  type="category"
                  dataKey="customer_name"
                  tick={{ fontSize: 12 }}
                  width={80}
                />
                <Tooltip
                  contentStyle={{ background: 'var(--color-card)', border: '1px solid var(--color-border)' }}
                  formatter={(value) => [`¥${Number(value).toLocaleString()}`, '损耗金额']}
                />
                <Bar dataKey="wastage_cost" radius={[0, 4, 4, 0]}>
                  {(data ?? []).map((_, i) => (
                    <Cell key={i} fill={GRADIENT_COLORS[i] ?? GRADIENT_COLORS[9]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

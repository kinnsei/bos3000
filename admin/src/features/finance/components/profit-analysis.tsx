import { useState, useMemo } from 'react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { CHART_COLORS } from '@/lib/theme/chart-theme'
import { TrendingUp, DollarSign, Percent } from 'lucide-react'

const TOOLTIP_STYLE = {
  contentStyle: { background: 'var(--color-card)', border: '1px solid var(--color-border)' },
}

interface ProfitRow {
  name: string
  revenue: number
  cost: number
  profit: number
}

const mockCustomerData: Record<string, ProfitRow[]> = {
  today: [
    { name: '星辰网络', revenue: 4520, cost: 2100, profit: 2420 },
    { name: '示例科技', revenue: 3800, cost: 1900, profit: 1900 },
    { name: '通达通信', revenue: 2200, cost: 1100, profit: 1100 },
    { name: '云桥通讯', revenue: 1500, cost: 850, profit: 650 },
    { name: '汇联科技', revenue: 980, cost: 600, profit: 380 },
    { name: '华信达', revenue: 750, cost: 420, profit: 330 },
    { name: '盛联通讯', revenue: 620, cost: 380, profit: 240 },
    { name: '瑞通科技', revenue: 480, cost: 300, profit: 180 },
    { name: '天翼联合', revenue: 350, cost: 230, profit: 120 },
    { name: '飞讯通信', revenue: 220, cost: 150, profit: 70 },
  ],
  week: [
    { name: '星辰网络', revenue: 32500, cost: 15200, profit: 17300 },
    { name: '示例科技', revenue: 28000, cost: 14000, profit: 14000 },
    { name: '通达通信', revenue: 18500, cost: 9200, profit: 9300 },
    { name: '云桥通讯', revenue: 12000, cost: 7000, profit: 5000 },
    { name: '汇联科技', revenue: 8500, cost: 5100, profit: 3400 },
    { name: '华信达', revenue: 6200, cost: 3500, profit: 2700 },
    { name: '盛联通讯', revenue: 4800, cost: 2900, profit: 1900 },
    { name: '瑞通科技', revenue: 3500, cost: 2200, profit: 1300 },
    { name: '天翼联合', revenue: 2800, cost: 1800, profit: 1000 },
    { name: '飞讯通信', revenue: 1500, cost: 1000, profit: 500 },
  ],
  month: [
    { name: '星辰网络', revenue: 125000, cost: 58000, profit: 67000 },
    { name: '示例科技', revenue: 108000, cost: 54000, profit: 54000 },
    { name: '通达通信', revenue: 72000, cost: 36000, profit: 36000 },
    { name: '云桥通讯', revenue: 48000, cost: 28000, profit: 20000 },
    { name: '汇联科技', revenue: 35000, cost: 21000, profit: 14000 },
    { name: '华信达', revenue: 25000, cost: 14000, profit: 11000 },
    { name: '盛联通讯', revenue: 19000, cost: 11500, profit: 7500 },
    { name: '瑞通科技', revenue: 14000, cost: 8800, profit: 5200 },
    { name: '天翼联合', revenue: 11000, cost: 7200, profit: 3800 },
    { name: '飞讯通信', revenue: 6000, cost: 4000, profit: 2000 },
  ],
}

const mockGatewayData: Record<string, ProfitRow[]> = {
  today: [
    { name: '网关A-移动', revenue: 5200, cost: 2800, profit: 2400 },
    { name: '网关B-联通', revenue: 3800, cost: 2100, profit: 1700 },
    { name: '网关C-电信', revenue: 2500, cost: 1400, profit: 1100 },
    { name: '网关D-国际', revenue: 1800, cost: 1200, profit: 600 },
    { name: '网关E-VOIP', revenue: 1120, cost: 700, profit: 420 },
  ],
  week: [
    { name: '网关A-移动', revenue: 38000, cost: 20000, profit: 18000 },
    { name: '网关B-联通', revenue: 28000, cost: 15500, profit: 12500 },
    { name: '网关C-电信', revenue: 19000, cost: 10500, profit: 8500 },
    { name: '网关D-国际', revenue: 13000, cost: 8800, profit: 4200 },
    { name: '网关E-VOIP', revenue: 8300, cost: 5200, profit: 3100 },
  ],
  month: [
    { name: '网关A-移动', revenue: 150000, cost: 78000, profit: 72000 },
    { name: '网关B-联通', revenue: 112000, cost: 62000, profit: 50000 },
    { name: '网关C-电信', revenue: 78000, cost: 43000, profit: 35000 },
    { name: '网关D-国际', revenue: 52000, cost: 35000, profit: 17000 },
    { name: '网关E-VOIP', revenue: 33000, cost: 20600, profit: 12400 },
  ],
}

const PERIOD_OPTIONS = [
  { value: 'today', label: '今日' },
  { value: 'week', label: '本周' },
  { value: 'month', label: '本月' },
]

function formatCurrency(v: number) {
  return `¥${v.toLocaleString()}`
}

export function ProfitAnalysis() {
  const [period, setPeriod] = useState('month')

  const customerData = mockCustomerData[period] ?? []
  const gatewayData = mockGatewayData[period] ?? []

  const summary = useMemo(() => {
    const totalRevenue = customerData.reduce((s, r) => s + r.revenue, 0)
    const totalCost = customerData.reduce((s, r) => s + r.cost, 0)
    const marginRate = totalRevenue > 0 ? ((totalRevenue - totalCost) / totalRevenue * 100) : 0
    return { totalRevenue, totalCost, marginRate }
  }, [customerData])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="grid grid-cols-3 gap-4 flex-1 mr-4">
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <div className="rounded-md bg-green-100 p-2 dark:bg-green-900/30">
                <DollarSign className="h-4 w-4 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">总收入</p>
                <p className="text-lg font-bold font-mono">{formatCurrency(summary.totalRevenue)}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <div className="rounded-md bg-red-100 p-2 dark:bg-red-900/30">
                <TrendingUp className="h-4 w-4 text-red-600 dark:text-red-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">总成本</p>
                <p className="text-lg font-bold font-mono">{formatCurrency(summary.totalCost)}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-3 p-4">
              <div className="rounded-md bg-blue-100 p-2 dark:bg-blue-900/30">
                <Percent className="h-4 w-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">毛利率</p>
                <p className="text-lg font-bold font-mono">{summary.marginRate.toFixed(1)}%</p>
              </div>
            </CardContent>
          </Card>
        </div>
        <Select value={period} onValueChange={setPeriod}>
          <SelectTrigger className="h-8 w-24">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PERIOD_OPTIONS.map((o) => (
              <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">按客户毛利分析</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-[420px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={customerData} layout="vertical" margin={{ left: 10, right: 20 }}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                  <XAxis
                    type="number"
                    tick={{ fontSize: 11 }}
                    tickFormatter={(v) => `¥${(v / 1000).toFixed(0)}k`}
                  />
                  <YAxis type="category" dataKey="name" tick={{ fontSize: 11 }} width={70} />
                  <Tooltip
                    {...TOOLTIP_STYLE}
                    formatter={(value: number, name: string) => [formatCurrency(value), name]}
                  />
                  <Legend />
                  <Bar dataKey="revenue" name="收入" fill={CHART_COLORS.primary} radius={[0, 2, 2, 0]} barSize={10} />
                  <Bar dataKey="cost" name="成本" fill={CHART_COLORS.danger} radius={[0, 2, 2, 0]} barSize={10} />
                  <Bar dataKey="profit" name="毛利" fill={CHART_COLORS.secondary} radius={[0, 2, 2, 0]} barSize={10} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">按网关毛利分析</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-[420px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={gatewayData} layout="vertical" margin={{ left: 10, right: 20 }}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                  <XAxis
                    type="number"
                    tick={{ fontSize: 11 }}
                    tickFormatter={(v) => `¥${(v / 1000).toFixed(0)}k`}
                  />
                  <YAxis type="category" dataKey="name" tick={{ fontSize: 11 }} width={80} />
                  <Tooltip
                    {...TOOLTIP_STYLE}
                    formatter={(value: number, name: string) => [formatCurrency(value), name]}
                  />
                  <Legend />
                  <Bar dataKey="revenue" name="收入" fill={CHART_COLORS.primary} radius={[0, 2, 2, 0]} barSize={14} />
                  <Bar dataKey="cost" name="成本" fill={CHART_COLORS.danger} radius={[0, 2, 2, 0]} barSize={14} />
                  <Bar dataKey="profit" name="毛利" fill={CHART_COLORS.secondary} radius={[0, 2, 2, 0]} barSize={14} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

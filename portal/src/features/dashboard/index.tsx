import { PhoneCall, Clock, Wallet, Users } from 'lucide-react'
import { StatCard } from '@/components/shared/stat-card'
import { useUsageSummary } from '@/lib/api/hooks'

export default function DashboardPage() {
  const { data, isLoading } = useUsageSummary()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">仪表盘</h1>
        <p className="text-muted-foreground">实时业务概览</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="今日呼叫"
          value={data?.today_calls ?? '-'}
          icon={PhoneCall}
          loading={isLoading}
        />
        <StatCard
          title="今日通话时长"
          value={data ? `${Math.floor(data.today_duration / 60)} 分钟` : '-'}
          icon={Clock}
          loading={isLoading}
        />
        <StatCard
          title="今日消费"
          value={data ? `¥${data.today_cost.toFixed(2)}` : '-'}
          icon={Wallet}
          loading={isLoading}
        />
        <StatCard
          title="当前并发"
          value={data ? `${data.concurrent_active} / ${data.concurrent_limit}` : '-'}
          icon={Users}
          loading={isLoading}
        />
      </div>

      {/* TODO: Add trend charts, recent calls table */}
      <div className="rounded-lg border p-8 text-center text-muted-foreground">
        趋势图表和最近通话记录将在后续迭代中实现
      </div>
    </div>
  )
}

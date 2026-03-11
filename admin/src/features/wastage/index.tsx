import { TrendingDown, Percent, AlertCircle } from 'lucide-react'
import { StatCard } from '@/components/shared/stat-card'
import { WastageTrend } from './components/wastage-trend'
import { CustomerRanking } from './components/customer-ranking'
import { FailureDistribution } from './components/failure-distribution'
import { useWastageSummary } from '@/lib/api/hooks'

export default function Wastage() {
  const { data: summary, isLoading } = useWastageSummary()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">损耗分析中心</h1>
        <p className="text-sm text-muted-foreground">平台损耗趋势、客户排名和失败原因分析</p>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard
          title="今日损耗金额"
          value={`¥${(summary?.today_wastage_cost ?? 0).toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`}
          icon={TrendingDown}
          loading={isLoading}
        />
        <StatCard
          title="今日损耗率"
          value={`${(summary?.today_wastage_rate ?? 0).toFixed(1)}%`}
          icon={Percent}
          loading={isLoading}
        />
        <StatCard
          title="最常见失败原因"
          value={summary?.top_failure_reason ?? '-'}
          icon={AlertCircle}
          loading={isLoading}
        />
      </div>

      {/* Trend chart - full width */}
      <WastageTrend />

      {/* Ranking + Distribution side by side */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <CustomerRanking />
        <FailureDistribution />
      </div>
    </div>
  )
}

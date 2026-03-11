import { OverviewStats } from './components/overview-stats'
import { AlertCards } from './components/alert-cards'
import { TrendCharts } from './components/trend-charts'
import { useDashboardOverview, useDashboardTrends } from '@/lib/api/hooks'

export default function Dashboard() {
  const { data: overview, isLoading } = useDashboardOverview()
  const { data: trends, isLoading: trendsLoading } = useDashboardTrends()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">运营概览</h1>
        <p className="text-sm text-muted-foreground">实时监控平台运营数据</p>
      </div>

      <OverviewStats data={overview} isLoading={isLoading} />

      <AlertCards
        alerts={overview?.alerts}
        bridgeRate={overview?.bridge_success_rate}
        wastageRate={overview?.today_wastage && overview?.today_revenue
          ? (overview.today_wastage / (overview.today_revenue + overview.today_wastage)) * 100
          : 0}
        isLoading={isLoading}
      />

      <TrendCharts
        revenueData={trends?.revenue ?? []}
        callData={trends?.calls ?? []}
        isLoading={trendsLoading}
      />
    </div>
  )
}

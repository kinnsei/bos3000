import { useState } from 'react'
import { type PaginationState, type SortingState } from '@tanstack/react-table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StatCard } from '@/components/shared/stat-card'
import { Phone, PhoneCall, Timer, TrendingUp } from 'lucide-react'
import { CdrFiltersBar, type CdrFilters } from './components/cdr-filters'
import { CdrTable } from './components/cdr-table'
import { LiveCalls } from './components/live-calls'
import { useCDRs } from '@/lib/api/hooks'

export default function CDR() {
  const [filters, setFilters] = useState<CdrFilters>({
    startDate: new Date(),
    endDate: new Date(),
    customer: 'all',
    status: 'all',
    gateway: 'all',
    search: '',
  })

  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 20 })
  const [sorting, setSorting] = useState<SortingState>([{ id: 'created_at', desc: true }])

  const { data, isLoading } = useCDRs({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    start_date: filters.startDate?.toISOString().split('T')[0],
    end_date: filters.endDate?.toISOString().split('T')[0],
    ...(filters.status !== 'all' ? { status: filters.status } : {}),
  })

  const cdrs = data?.cdrs ?? []
  const totalCount = data?.total ?? 0

  // Compute stats from current page data
  const completedCount = cdrs.filter((c) => c.status === 'finished' || c.status === 'completed').length
  const bridgeRate = totalCount > 0 ? ((completedCount / Math.max(cdrs.length, 1)) * 100).toFixed(1) : '0.0'
  const avgDuration = cdrs.length > 0
    ? Math.round(cdrs.reduce((s, c) => s + c.duration, 0) / cdrs.length)
    : 0
  const avgMin = Math.floor(avgDuration / 60)
  const avgSec = avgDuration % 60

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">话单管理</h1>
        <p className="text-sm text-muted-foreground">全量话单查询和实时通话监控</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard title="通话总量" value={totalCount} icon={Phone} />
        <StatCard title="接通率" value={`${bridgeRate}%`} icon={PhoneCall} />
        <StatCard
          title="平均通话时长"
          value={`${avgMin}分${String(avgSec).padStart(2, '0')}秒`}
          icon={Timer}
        />
      </div>

      <Tabs defaultValue="records">
        <TabsList>
          <TabsTrigger value="records">通话记录</TabsTrigger>
          <TabsTrigger value="live">
            <TrendingUp className="mr-1 h-3.5 w-3.5" />
            实时通话
          </TabsTrigger>
        </TabsList>

        <TabsContent value="records" className="mt-4">
          <CdrTable
            data={cdrs}
            totalCount={totalCount}
            pagination={pagination}
            onPaginationChange={setPagination}
            sorting={sorting}
            onSortingChange={setSorting}
            isLoading={isLoading}
            toolbar={<CdrFiltersBar filters={filters} onChange={setFilters} />}
          />
        </TabsContent>

        <TabsContent value="live" className="mt-4">
          <LiveCalls />
        </TabsContent>
      </Tabs>
    </div>
  )
}

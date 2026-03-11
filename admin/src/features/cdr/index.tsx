import { useState, useMemo } from 'react'
import { type PaginationState, type SortingState } from '@tanstack/react-table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StatCard } from '@/components/shared/stat-card'
import { Phone, PhoneCall, Timer, TrendingUp } from 'lucide-react'
import { CdrFiltersBar, type CdrFilters } from './components/cdr-filters'
import { CdrTable } from './components/cdr-table'
import { LiveCalls } from './components/live-calls'
import type { CDR } from '@/lib/api/client'

// --- Mock data generator ---

const STATUSES = ['completed', 'failed', 'in-progress']

function generateMockCDRs(count: number): CDR[] {
  const now = Date.now()
  return Array.from({ length: count }, (_, i) => {
    const status = STATUSES[i % 5 === 0 ? 1 : i % 7 === 0 ? 2 : 0]
    const duration = status === 'failed' ? 0 : 15 + Math.floor(Math.random() * 300)
    const startMs = now - (i * 120 + Math.floor(Math.random() * 60)) * 1000
    return {
      id: `cdr-${String(i + 1).padStart(4, '0')}`,
      call_id: `${crypto.randomUUID()}`,
      caller: `1${String(3000000000 + Math.floor(Math.random() * 9000000000)).slice(0, 10)}`,
      callee: `1${String(5000000000 + Math.floor(Math.random() * 9000000000)).slice(0, 10)}`,
      status,
      duration,
      cost: status === 'failed' ? 0 : +(duration * 0.08 + Math.random() * 0.5).toFixed(2),
      started_at: new Date(startMs).toISOString(),
      ended_at: new Date(startMs + duration * 1000).toISOString(),
    }
  })
}

const ALL_MOCK_CDRS = generateMockCDRs(256)

// --- Stats ---

function useMockStats() {
  return useMemo(() => {
    const total = ALL_MOCK_CDRS.length
    const completed = ALL_MOCK_CDRS.filter((c) => c.status === 'completed').length
    const avgDuration = Math.round(
      ALL_MOCK_CDRS.reduce((s, c) => s + c.duration, 0) / total,
    )
    const avgMin = Math.floor(avgDuration / 60)
    const avgSec = avgDuration % 60
    return {
      total,
      bridgeRate: ((completed / total) * 100).toFixed(1),
      avgDuration: `${avgMin}分${String(avgSec).padStart(2, '0')}秒`,
    }
  }, [])
}

// --- Main component ---

export default function CDR() {
  const stats = useMockStats()

  // Filters
  const [filters, setFilters] = useState<CdrFilters>({
    startDate: new Date(),
    endDate: new Date(),
    customer: 'all',
    status: 'all',
    gateway: 'all',
    search: '',
  })

  // Pagination & sorting
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 20 })
  const [sorting, setSorting] = useState<SortingState>([{ id: 'started_at', desc: true }])

  // Filter + sort mock data (client side for mock)
  const { pageData, totalCount } = useMemo(() => {
    let filtered = ALL_MOCK_CDRS

    if (filters.status !== 'all') {
      filtered = filtered.filter((c) => c.status === filters.status)
    }
    if (filters.search) {
      const q = filters.search.toLowerCase()
      filtered = filtered.filter(
        (c) => c.caller.includes(q) || c.callee.includes(q),
      )
    }

    // Sort
    if (sorting.length > 0) {
      const { id, desc } = sorting[0]
      filtered = [...filtered].sort((a, b) => {
        const av = a[id as keyof CDR]
        const bv = b[id as keyof CDR]
        if (typeof av === 'number' && typeof bv === 'number') return desc ? bv - av : av - bv
        return desc
          ? String(bv).localeCompare(String(av))
          : String(av).localeCompare(String(bv))
      })
    }

    const start = pagination.pageIndex * pagination.pageSize
    return {
      pageData: filtered.slice(start, start + pagination.pageSize),
      totalCount: filtered.length,
    }
  }, [filters, pagination, sorting])

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">话单管理</h1>
        <p className="text-muted-foreground">全量话单查询和实时通话监控</p>
      </div>

      {/* Summary stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="今日通话总量"
          value={stats.total}
          icon={Phone}
          trend={{ direction: 'up', value: '+12.5%' }}
          description="较昨日"
        />
        <StatCard
          title="接通率"
          value={`${stats.bridgeRate}%`}
          icon={PhoneCall}
          trend={{ direction: 'up', value: '+2.1%' }}
          description="较昨日"
        />
        <StatCard
          title="平均通话时长"
          value={stats.avgDuration}
          icon={Timer}
          trend={{ direction: 'neutral', value: '持平' }}
          description="较昨日"
        />
      </div>

      {/* Tabs */}
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
            data={pageData}
            totalCount={totalCount}
            pagination={pagination}
            onPaginationChange={setPagination}
            sorting={sorting}
            onSortingChange={setSorting}
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

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Calendar } from '@/components/ui/calendar'
import { CalendarIcon, RotateCcw, Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import { format } from 'date-fns'

export interface CdrFilters {
  startDate: Date | undefined
  endDate: Date | undefined
  customer: string
  status: string
  gateway: string
  search: string
}

const MOCK_CUSTOMERS = ['全部', '客户A', '客户B', '客户C', '客户D', '客户E']
const MOCK_GATEWAYS = ['全部', '网关-北京01', '网关-上海02', '网关-广州03', '网关-深圳04']
const STATUS_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'completed', label: '已完成' },
  { value: 'failed', label: '失败' },
  { value: 'in-progress', label: '进行中' },
]

interface CdrFiltersBarProps {
  filters: CdrFilters
  onChange: (filters: CdrFilters) => void
}

export function CdrFiltersBar({ filters, onChange }: CdrFiltersBarProps) {
  const update = (patch: Partial<CdrFilters>) => onChange({ ...filters, ...patch })

  const reset = () =>
    onChange({
      startDate: new Date(),
      endDate: new Date(),
      customer: 'all',
      status: 'all',
      gateway: 'all',
      search: '',
    })

  return (
    <div className="flex flex-wrap items-end gap-3">
      {/* Date range */}
      <div className="flex items-end gap-2">
        <div className="space-y-1">
          <Label className="text-xs">开始日期</Label>
          <DatePicker
            date={filters.startDate}
            onSelect={(d) => update({ startDate: d })}
          />
        </div>
        <span className="pb-2 text-muted-foreground">-</span>
        <div className="space-y-1">
          <Label className="text-xs">结束日期</Label>
          <DatePicker
            date={filters.endDate}
            onSelect={(d) => update({ endDate: d })}
          />
        </div>
      </div>

      {/* Customer */}
      <div className="space-y-1">
        <Label className="text-xs">客户筛选</Label>
        <Select value={filters.customer} onValueChange={(v) => update({ customer: v })}>
          <SelectTrigger className="w-[130px]">
            <SelectValue placeholder="全部" />
          </SelectTrigger>
          <SelectContent>
            {MOCK_CUSTOMERS.map((c) => (
              <SelectItem key={c} value={c === '全部' ? 'all' : c}>
                {c}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Status */}
      <div className="space-y-1">
        <Label className="text-xs">状态</Label>
        <Select value={filters.status} onValueChange={(v) => update({ status: v })}>
          <SelectTrigger className="w-[110px]">
            <SelectValue placeholder="全部" />
          </SelectTrigger>
          <SelectContent>
            {STATUS_OPTIONS.map((s) => (
              <SelectItem key={s.value} value={s.value}>
                {s.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Gateway */}
      <div className="space-y-1">
        <Label className="text-xs">网关</Label>
        <Select value={filters.gateway} onValueChange={(v) => update({ gateway: v })}>
          <SelectTrigger className="w-[150px]">
            <SelectValue placeholder="全部" />
          </SelectTrigger>
          <SelectContent>
            {MOCK_GATEWAYS.map((g) => (
              <SelectItem key={g} value={g === '全部' ? 'all' : g}>
                {g}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Search */}
      <div className="space-y-1">
        <Label className="text-xs">号码搜索</Label>
        <div className="relative">
          <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="主/被叫号码"
            value={filters.search}
            onChange={(e) => update({ search: e.target.value })}
            className="h-8 w-[160px] pl-7"
          />
        </div>
      </div>

      {/* Reset */}
      <Button variant="outline" size="sm" onClick={reset} className="h-8">
        <RotateCcw className="mr-1 h-3.5 w-3.5" />
        重置
      </Button>
    </div>
  )
}

function DatePicker({
  date,
  onSelect,
}: {
  date: Date | undefined
  onSelect: (d: Date | undefined) => void
}) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className={cn('h-8 w-[130px] justify-start text-left font-normal', !date && 'text-muted-foreground')}
        >
          <CalendarIcon className="mr-1.5 h-3.5 w-3.5" />
          {date ? format(date, 'yyyy-MM-dd') : '选择日期'}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar mode="single" selected={date} onSelect={onSelect} />
      </PopoverContent>
    </Popover>
  )
}

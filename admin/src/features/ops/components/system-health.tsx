import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Database, HardDrive, Globe, Activity } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ResourceCard {
  title: string
  icon: React.ElementType
  status: 'normal' | 'warning' | 'error'
  metrics: { label: string; value: string | number }[]
}

const resources: ResourceCard[] = [
  {
    title: '数据库',
    icon: Database,
    status: 'normal',
    metrics: [
      { label: '活跃连接', value: '12 / 100' },
      { label: '空闲连接', value: 38 },
      { label: '延迟', value: '2.4 ms' },
    ],
  },
  {
    title: 'Redis',
    icon: HardDrive,
    status: 'normal',
    metrics: [
      { label: '状态', value: '已连接' },
      { label: '内存使用', value: '128 MB / 512 MB' },
      { label: '命中率', value: '96.2%' },
    ],
  },
  {
    title: 'API 网关',
    icon: Globe,
    status: 'normal',
    metrics: [
      { label: '请求/分', value: 342 },
      { label: '错误率', value: '0.12%' },
      { label: '平均延迟', value: '45 ms' },
    ],
  },
  {
    title: '回呼引擎',
    icon: Activity,
    status: 'warning',
    metrics: [
      { label: '队列深度', value: 7 },
      { label: '处理中', value: 3 },
      { label: '失败率', value: '1.8%' },
    ],
  },
]

function overallStatus(cards: ResourceCard[]) {
  const hasError = cards.some((c) => c.status === 'error')
  return hasError
}

export function SystemHealth() {
  const isError = overallStatus(resources)

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium text-muted-foreground">系统状态</span>
        <Badge
          className={cn(
            isError
              ? 'bg-red-600 hover:bg-red-600'
              : 'bg-green-600 hover:bg-green-600',
          )}
        >
          {isError ? '系统异常' : '系统正常'}
        </Badge>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {resources.map((res) => {
          const Icon = res.icon
          return (
            <Card key={res.title}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{res.title}</CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="space-y-1.5">
                {res.metrics.map((m) => (
                  <div key={m.label} className="flex justify-between text-sm">
                    <span className="text-muted-foreground">{m.label}</span>
                    <span className="font-medium">{m.value}</span>
                  </div>
                ))}
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}

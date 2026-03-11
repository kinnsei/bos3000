import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { FsStatusCards } from './components/fs-status-cards'
import { SystemHealth } from './components/system-health'
import { AlertTriangle, CheckCircle, Info, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

const eventIcons = {
  info: Info,
  success: CheckCircle,
  warning: AlertTriangle,
  error: XCircle,
} as const

type EventLevel = keyof typeof eventIcons

interface SystemEvent {
  time: string
  level: EventLevel
  message: string
}

const events: SystemEvent[] = [
  { time: '14:32:05', level: 'info', message: 'FreeSWITCH 主节点健康检查通过' },
  { time: '14:31:58', level: 'warning', message: 'FreeSWITCH 备用节点 ESL 连接断开' },
  { time: '14:30:00', level: 'success', message: '数据库连接池回收完成' },
  { time: '14:28:12', level: 'info', message: 'API 网关证书自动续期成功' },
  { time: '14:25:33', level: 'error', message: '回呼任务 #1087 重试失败，已加入死信队列' },
  { time: '14:20:01', level: 'info', message: 'Redis 内存清理 cron 执行完成' },
  { time: '14:15:44', level: 'success', message: '回呼任务 #1086 完成' },
  { time: '14:10:02', level: 'info', message: '系统指标快照已写入' },
  { time: '14:05:00', level: 'info', message: 'FreeSWITCH 主节点健康检查通过' },
  { time: '14:00:00', level: 'info', message: '定时报表生成任务启动' },
]

const levelColors: Record<EventLevel, string> = {
  info: 'text-blue-500',
  success: 'text-green-500',
  warning: 'text-yellow-500',
  error: 'text-red-500',
}

export default function Ops() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">运维监控</h1>
        <p className="text-muted-foreground">FreeSWITCH 状态和系统健康监控</p>
      </div>

      <FsStatusCards />

      <Separator />

      <SystemHealth />

      <Separator />

      <div className="space-y-3">
        <h2 className="text-sm font-medium text-muted-foreground">系统事件</h2>
        <div className="space-y-1">
          {events.map((evt, i) => {
            const Icon = eventIcons[evt.level]
            return (
              <div key={i} className="flex items-center gap-2 text-sm py-1">
                <Icon className={cn('h-3.5 w-3.5 shrink-0', levelColors[evt.level])} />
                <span className="text-muted-foreground font-mono text-xs w-16 shrink-0">{evt.time}</span>
                <span>{evt.message}</span>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

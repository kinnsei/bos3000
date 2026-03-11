import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Server, Clock, Users, Timer } from 'lucide-react'
import { cn } from '@/lib/utils'

interface FsInstance {
  hostname: string
  role: '主节点' | '备用节点'
  eslStatus: 'connected' | 'disconnected'
  lastHealthCheck: string
  activeSessions: number
  uptime: string
}

const instances: FsInstance[] = [
  {
    hostname: 'fs-primary-01.bos3000.local',
    role: '主节点',
    eslStatus: 'connected',
    lastHealthCheck: '2026-03-11 14:32:05',
    activeSessions: 24,
    uptime: '15 天 8 小时',
  },
  {
    hostname: 'fs-standby-02.bos3000.local',
    role: '备用节点',
    eslStatus: 'disconnected',
    lastHealthCheck: '2026-03-11 14:31:58',
    activeSessions: 0,
    uptime: '15 天 8 小时',
  },
]

function StatusDot({ connected }: { connected: boolean }) {
  return (
    <span
      className={cn(
        'inline-block h-2.5 w-2.5 rounded-full',
        connected ? 'bg-green-500' : 'bg-red-500 animate-pulse',
      )}
    />
  )
}

function InstanceCard({ instance }: { instance: FsInstance }) {
  const connected = instance.eslStatus === 'connected'

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div className="flex items-center gap-2">
          <Server className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-medium">{instance.role}</CardTitle>
        </div>
        <div className="flex items-center gap-2">
          <StatusDot connected={connected} />
          <Badge
            variant={connected ? 'default' : 'destructive'}
            className={cn(connected && 'bg-green-600 hover:bg-green-600')}
          >
            {connected ? '已连接' : '已断开'}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-xs text-muted-foreground font-mono">{instance.hostname}</p>
        <div className="grid grid-cols-3 gap-3 text-sm">
          <div className="flex items-center gap-1.5">
            <Users className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">会话</span>
            <span className="font-semibold">{instance.activeSessions}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Timer className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">运行</span>
            <span className="font-semibold">{instance.uptime}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Clock className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">检查</span>
            <span className="font-semibold text-xs">{instance.lastHealthCheck.split(' ')[1]}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function FsStatusCards() {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {instances.map((inst) => (
        <InstanceCard key={inst.hostname} instance={inst} />
      ))}
    </div>
  )
}

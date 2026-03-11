import type { WsStatus } from '@/hooks/use-call-ws'
import { cn } from '@/lib/utils'
import { Wifi, WifiOff, Loader2 } from 'lucide-react'

interface WsStatusBannerProps {
  status: WsStatus
}

export function WsStatusBanner({ status }: WsStatusBannerProps) {
  if (status === 'connected') return null

  return (
    <div
      className={cn(
        'flex items-center justify-center gap-2 px-4 py-1.5 text-xs font-medium',
        status === 'connecting' && 'bg-yellow-500/10 text-yellow-700 dark:text-yellow-400',
        status === 'disconnected' && 'bg-red-500/10 text-red-700 dark:text-red-400',
        status === 'error' && 'bg-red-500/10 text-red-700 dark:text-red-400',
      )}
    >
      {status === 'connecting' && (
        <>
          <Loader2 className="h-3 w-3 animate-spin" />
          <span>正在连接实时通话状态...</span>
        </>
      )}
      {status === 'disconnected' && (
        <>
          <WifiOff className="h-3 w-3" />
          <span>实时连接已断开，正在重连...</span>
        </>
      )}
      {status === 'error' && (
        <>
          <Wifi className="h-3 w-3" />
          <span>实时连接异常，正在重连...</span>
        </>
      )}
    </div>
  )
}

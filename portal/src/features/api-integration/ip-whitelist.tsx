import { useState } from 'react'
import { Plus, Trash2, ShieldAlert } from 'lucide-react'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/shared/confirm-dialog'
import { useIpWhitelist, useAddIp, useRemoveIp } from '@/lib/api/hooks'

export function IpWhitelist() {
  const { data: ips = [] } = useIpWhitelist()
  const addIp = useAddIp()
  const removeIp = useRemoveIp()
  const [newIp, setNewIp] = useState('')

  const handleAdd = async () => {
    const ip = newIp.trim()
    if (!ip) return
    try {
      await addIp.mutateAsync({ ip })
      setNewIp('')
      toast.success(`已添加 IP: ${ip}`)
    } catch {
      toast.error('添加 IP 失败')
    }
  }

  const handleRemove = async (ip: string) => {
    try {
      await removeIp.mutateAsync({ ip })
      toast.success(`已移除 IP: ${ip}`)
    } catch {
      toast.error('移除 IP 失败')
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>IP 白名单</CardTitle>
        <CardDescription>限制允许调用 API 的来源 IP 地址</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2">
          <Input
            placeholder="输入 IP 地址，例如 203.0.113.10"
            value={newIp}
            onChange={(e) => setNewIp(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
            className="max-w-xs"
          />
          <Button onClick={handleAdd} disabled={!newIp.trim() || addIp.isPending}>
            <Plus className="mr-2 h-4 w-4" />
            添加
          </Button>
        </div>

        {ips.length === 0 ? (
          <div className="flex items-center gap-2 rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
            <ShieldAlert className="h-4 w-4 shrink-0" />
            未配置 IP 白名单（所有 IP 可访问）
          </div>
        ) : (
          <div className="space-y-2">
            {ips.map((ip) => (
              <div
                key={ip}
                className="flex items-center justify-between rounded-md border px-4 py-2"
              >
                <code className="font-mono text-sm">{ip}</code>
                <ConfirmDialog
                  trigger={
                    <Button variant="ghost" size="icon" disabled={removeIp.isPending}>
                      <Trash2 className="h-4 w-4 text-muted-foreground" />
                    </Button>
                  }
                  title="移除 IP 地址"
                  description={`确定要从白名单中移除 ${ip} 吗？移除后该 IP 将无法访问 API。`}
                  confirmText="确认移除"
                  variant="danger"
                  onConfirm={() => handleRemove(ip)}
                />
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/shared/confirm-dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  ArrowLeft,
  Copy,
  RefreshCw,
  Plus,
  X,
  Wallet,
  MinusCircle,
} from 'lucide-react'
import { toast } from 'sonner'

// Mock data for demonstration
const mockCustomerDetail = {
  id: '1',
  company: '示例科技有限公司',
  email: 'admin@example.com',
  status: 'active' as const,
  balance: 12500.5,
  credit_limit: 5000,
  max_concurrent: 50,
  daily_limit: 2000,
  created_at: '2025-12-01T08:00:00Z',
  api_key: 'bos_sk_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6',
  ip_whitelist: ['192.168.1.100', '10.0.0.1', '172.16.0.50'],
  recent_calls: [
    { id: 'c1', caller: '13800001111', callee: '02188889999', duration: 120, cost: 2.4, started_at: '2026-03-10T14:30:00Z' },
    { id: 'c2', caller: '13900002222', callee: '01066667777', duration: 45, cost: 0.9, started_at: '2026-03-10T14:15:00Z' },
    { id: 'c3', caller: '13700003333', callee: '02155554444', duration: 300, cost: 6.0, started_at: '2026-03-10T13:50:00Z' },
    { id: 'c4', caller: '13600004444', callee: '07588883333', duration: 60, cost: 1.2, started_at: '2026-03-10T13:20:00Z' },
    { id: 'c5', caller: '13500005555', callee: '02866662222', duration: 180, cost: 3.6, started_at: '2026-03-10T12:45:00Z' },
  ],
}

interface CustomerDetailProps {
  customerId: string
  onBack: () => void
}

export function CustomerDetail({ customerId: _customerId, onBack }: CustomerDetailProps) {
  const customer = mockCustomerDetail
  const [ipInput, setIpInput] = useState('')
  const [ipList, setIpList] = useState(customer.ip_whitelist)

  const maskedKey = `${customer.api_key.slice(0, 8)}...${customer.api_key.slice(-4)}`

  const handleCopyKey = () => {
    navigator.clipboard.writeText(customer.api_key)
    toast.success('API Key 已复制到剪贴板')
  }

  const handleAddIp = () => {
    const trimmed = ipInput.trim()
    if (!trimmed) return
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/
    if (!ipRegex.test(trimmed)) {
      toast.error('IP 地址格式不正确')
      return
    }
    if (ipList.includes(trimmed)) {
      toast.error('该 IP 已存在')
      return
    }
    setIpList([...ipList, trimmed])
    setIpInput('')
    toast.success('IP 已添加')
  }

  const handleRemoveIp = (ip: string) => {
    setIpList(ipList.filter((i) => i !== ip))
    toast.success('IP 已移除')
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h2 className="text-xl font-semibold">{customer.company}</h2>
          <p className="text-sm text-muted-foreground">{customer.email}</p>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Info Card */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">基本信息</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">公司名称</span>
              <span>{customer.company}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">邮箱</span>
              <span>{customer.email}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">状态</span>
              <Badge
                className={
                  customer.status === 'active'
                    ? 'bg-green-500/10 text-green-600'
                    : 'bg-red-500/10 text-red-600'
                }
              >
                {customer.status === 'active' ? '正常' : '已冻结'}
              </Badge>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">并发上限</span>
              <span>{customer.max_concurrent}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">日限额</span>
              <span>{customer.daily_limit}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">创建时间</span>
              <span>{new Date(customer.created_at).toLocaleDateString('zh-CN')}</span>
            </div>
          </CardContent>
        </Card>

        {/* Balance Card */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">余额信息</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm text-muted-foreground">当前余额</p>
              <p className="text-3xl font-bold font-mono">
                ¥{customer.balance.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}
              </p>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">信用额度</span>
              <span className="font-mono">¥{customer.credit_limit.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}</span>
            </div>
            <Separator />
            <div className="flex gap-2">
              <Button size="sm" className="flex-1" onClick={() => toast.info('充值功能待接入')}>
                <Wallet className="mr-1.5 h-3.5 w-3.5" />
                充值
              </Button>
              <Button size="sm" variant="outline" className="flex-1" onClick={() => toast.info('扣款功能待接入')}>
                <MinusCircle className="mr-1.5 h-3.5 w-3.5" />
                扣款
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* API Key Card */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">API Key</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded bg-muted px-3 py-2 text-sm font-mono">
                {maskedKey}
              </code>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline" size="icon" className="h-9 w-9" onClick={handleCopyKey}>
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>复制</TooltipContent>
                </Tooltip>
              </TooltipProvider>
              <ConfirmDialog
                trigger={
                  <Button variant="outline" size="icon" className="h-9 w-9">
                    <RefreshCw className="h-3.5 w-3.5" />
                  </Button>
                }
                title="重新生成 API Key"
                description="重新生成后，旧的 API Key 将立即失效。客户需要更新其集成配置。确定要继续吗？"
                confirmText="重新生成"
                variant="warning"
                onConfirm={() => toast.success('API Key 已重新生成')}
              />
            </div>
          </CardContent>
        </Card>

        {/* IP Whitelist Card */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">IP 白名单</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex gap-2">
              <Input
                placeholder="输入 IP 地址"
                value={ipInput}
                onChange={(e) => setIpInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddIp())}
                className="flex-1"
              />
              <Button variant="outline" size="icon" onClick={handleAddIp}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-1.5">
              {ipList.length === 0 ? (
                <p className="text-sm text-muted-foreground py-2">暂无 IP 白名单</p>
              ) : (
                ipList.map((ip) => (
                  <div
                    key={ip}
                    className="flex items-center justify-between rounded bg-muted px-3 py-1.5 text-sm font-mono"
                  >
                    <span>{ip}</span>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6"
                      onClick={() => handleRemoveIp(ip)}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Calls */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">最近通话</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>主叫</TableHead>
                <TableHead>被叫</TableHead>
                <TableHead>时长(秒)</TableHead>
                <TableHead>费用</TableHead>
                <TableHead>时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {customer.recent_calls.map((call) => (
                <TableRow key={call.id}>
                  <TableCell className="font-mono">{call.caller}</TableCell>
                  <TableCell className="font-mono">{call.callee}</TableCell>
                  <TableCell>{call.duration}</TableCell>
                  <TableCell className="font-mono">¥{call.cost.toFixed(2)}</TableCell>
                  <TableCell>
                    {new Date(call.started_at).toLocaleString('zh-CN', {
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Freeze / Unfreeze */}
      <div className="flex justify-end">
        {customer.status === 'active' ? (
          <ConfirmDialog
            trigger={<Button variant="destructive">冻结账户</Button>}
            title="确认冻结账户"
            description={`冻结后，${customer.company} 将无法发起新的呼叫，且 API 访问将被拒绝。确定要冻结该客户吗？`}
            confirmText="确认冻结"
            variant="danger"
            onConfirm={() => toast.success('账户已冻结')}
          />
        ) : (
          <ConfirmDialog
            trigger={<Button variant="outline">解冻账户</Button>}
            title="确认解冻账户"
            description={`解冻后，${customer.company} 将恢复正常服务。确定要解冻该客户吗？`}
            confirmText="确认解冻"
            variant="warning"
            onConfirm={() => toast.success('账户已解冻')}
          />
        )}
      </div>
    </div>
  )
}

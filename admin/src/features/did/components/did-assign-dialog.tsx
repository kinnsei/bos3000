import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface DidAssignDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  did: { id: string; number: string } | null
  onSubmit: (didId: string, customerId: string, poolType: 'dedicated' | 'shared') => void
}

const mockCustomers = [
  { id: 'c1', name: '示例科技有限公司' },
  { id: 'c2', name: '通达通信' },
  { id: 'c3', name: '汇联科技' },
  { id: 'c4', name: '星辰网络' },
  { id: 'c5', name: '云桥通讯' },
  { id: 'c6', name: '盛达科技' },
  { id: 'c7', name: '明远通讯' },
]

export function DidAssignDialog({ open, onOpenChange, did, onSubmit }: DidAssignDialogProps) {
  const [customerId, setCustomerId] = useState('')
  const [poolType, setPoolType] = useState<'dedicated' | 'shared'>('dedicated')

  useEffect(() => {
    if (open) {
      setCustomerId('')
      setPoolType('dedicated')
    }
  }, [open])

  const handleSubmit = () => {
    if (!did || !customerId) return
    onSubmit(did.id, customerId, poolType)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>分配客户</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label className="text-muted-foreground text-xs">DID号码</Label>
            <p className="font-mono text-sm font-medium">{did?.number}</p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="customer">客户</Label>
            <Select value={customerId} onValueChange={setCustomerId}>
              <SelectTrigger id="customer">
                <SelectValue placeholder="请选择客户" />
              </SelectTrigger>
              <SelectContent>
                {mockCustomers.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="pool-type">号码池类型</Label>
            <Select value={poolType} onValueChange={(v) => setPoolType(v as 'dedicated' | 'shared')}>
              <SelectTrigger id="pool-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="dedicated">专属</SelectItem>
                <SelectItem value="shared">共享</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={!customerId}>
            确认分配
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

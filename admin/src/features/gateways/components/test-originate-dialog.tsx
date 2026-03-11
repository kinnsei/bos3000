import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod/v4'
import { zodResolver } from '@hookform/resolvers/zod/v4'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Phone } from 'lucide-react'
import type { Gateway, TestOriginateResult } from '@/lib/api/client'

const schema = z.object({
  phoneNumber: z.string().min(1, '请输入测试号码'),
})

type FormValues = z.infer<typeof schema>

interface TestOriginateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  gateway: Gateway | null
  onTest: (gatewayId: string, phoneNumber: string) => Promise<TestOriginateResult>
}

export function TestOriginateDialog({
  open,
  onOpenChange,
  gateway,
  onTest,
}: TestOriginateDialogProps) {
  const [result, setResult] = useState<TestOriginateResult | null>(null)

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { phoneNumber: '' },
  })

  const onSubmit = async (data: FormValues) => {
    if (!gateway) return
    setResult(null)
    const res = await onTest(gateway.id, data.phoneNumber)
    setResult(res)
  }

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      reset()
      setResult(null)
    }
    onOpenChange(open)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>测试呼叫</DialogTitle>
          <DialogDescription>
            通过网关 {gateway?.name} 发起测试呼叫
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="phoneNumber">测试号码</Label>
            <div className="relative">
              <Phone className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                id="phoneNumber"
                {...register('phoneNumber')}
                placeholder="输入测试号码"
                className="pl-8"
              />
            </div>
            {errors.phoneNumber && (
              <p className="text-sm text-destructive">{errors.phoneNumber.message}</p>
            )}
          </div>

          {result && (
            <div className="rounded-md border p-3 space-y-1">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">结果：</span>
                <Badge className={result.success ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'}>
                  {result.success ? '成功' : '失败'}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">{result.message}</p>
              {result.duration_ms > 0 && (
                <p className="text-xs text-muted-foreground">
                  耗时：{result.duration_ms}ms
                </p>
              )}
            </div>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              关闭
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? '呼叫中...' : '发起呼叫'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

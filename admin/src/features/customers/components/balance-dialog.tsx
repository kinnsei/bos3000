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

interface BalanceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  mode: 'topup' | 'deduct'
  customer: { id: string; username: string; balance: number } | null
  onSubmit: (data: { amount: number; remark?: string }) => void | Promise<void>
}

const topupSchema = z.object({
  amount: z.number().positive('金额必须大于0'),
  remark: z.string().optional(),
})

const deductSchema = z.object({
  amount: z.number().positive('金额必须大于0'),
  remark: z.string().min(1, '扣款必须填写备注'),
})

export function BalanceDialog({ open, onOpenChange, mode, customer, onSubmit }: BalanceDialogProps) {
  const schema = mode === 'topup' ? topupSchema : deductSchema

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<{ amount: number; remark: string }>({
    resolver: zodResolver(schema),
    defaultValues: { amount: 0, remark: '' },
  })

  const watchAmount = watch('amount')
  const exceedsBalance = mode === 'deduct' && customer && watchAmount > customer.balance

  const onFormSubmit = async (data: { amount: number; remark?: string }) => {
    await onSubmit(data)
    reset()
    onOpenChange(false)
  }

  const handleOpenChange = (val: boolean) => {
    if (!val) reset()
    onOpenChange(val)
  }

  if (!customer) return null

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{mode === 'topup' ? '充值' : '扣款'}</DialogTitle>
          <DialogDescription>
            客户: {customer.username}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onFormSubmit)} className="space-y-4">
          <div className="rounded-md bg-muted p-3 text-sm">
            当前余额: <span className="font-mono font-medium">¥{customer.balance.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}</span>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="amount">{mode === 'topup' ? '充值金额' : '扣款金额'} (¥) *</Label>
            <Input
              id="amount"
              type="number"
              step="0.01"
              min="0.01"
              {...register('amount', { valueAsNumber: true })}
              placeholder="请输入金额"
            />
            {errors.amount && <p className="text-xs text-destructive">{errors.amount.message}</p>}
            {exceedsBalance && (
              <p className="text-xs text-red-600 dark:text-red-400 font-medium">扣款金额超过当前余额，请注意!</p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="remark">
              备注 {mode === 'deduct' ? '*' : ''}
            </Label>
            <Input
              id="remark"
              {...register('remark')}
              placeholder={mode === 'deduct' ? '请填写扣款原因' : '选填'}
            />
            {errors.remark && <p className="text-xs text-destructive">{errors.remark.message}</p>}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              取消
            </Button>
            <Button
              type="submit"
              disabled={isSubmitting}
              variant={mode === 'deduct' ? 'destructive' : 'default'}
            >
              {isSubmitting ? '处理中...' : mode === 'topup' ? '确认充值' : '确认扣款'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

import { useForm } from 'react-hook-form'
import { z } from 'zod/v4'
import { zodResolver } from '@hookform/resolvers/zod/v4'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const createCustomerSchema = z.object({
  company: z.string().min(1, '请输入公司名称'),
  email: z.string().email('请输入有效的邮箱地址'),
  password: z.string().min(8, '密码至少8个字符'),
  phone: z.string().optional(),
  max_concurrent: z.number().int().min(1, '至少为1').default(10),
  daily_limit: z.number().int().min(1, '至少为1').default(1000),
  initial_balance: z.number().min(0, '不能为负数').default(0),
})

export type CreateCustomerData = z.infer<typeof createCustomerSchema>

interface CreateCustomerSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: CreateCustomerData) => void | Promise<void>
}

export function CreateCustomerSheet({ open, onOpenChange, onSubmit }: CreateCustomerSheetProps) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateCustomerData>({
    resolver: zodResolver(createCustomerSchema),
    defaultValues: {
      company: '',
      email: '',
      password: '',
      phone: '',
      max_concurrent: 10,
      daily_limit: 1000,
      initial_balance: 0,
    },
  })

  const onFormSubmit = async (data: CreateCustomerData) => {
    await onSubmit(data)
    reset()
    onOpenChange(false)
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <SheetTitle>新建客户</SheetTitle>
          <SheetDescription>填写客户信息以创建新账户</SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit(onFormSubmit)} className="flex flex-col gap-4 px-4 flex-1">
          <div className="space-y-1.5">
            <Label htmlFor="company">公司名称 *</Label>
            <Input id="company" {...register('company')} placeholder="请输入公司名称" />
            {errors.company && <p className="text-xs text-destructive">{errors.company.message}</p>}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="email">邮箱 *</Label>
            <Input id="email" type="email" {...register('email')} placeholder="请输入邮箱" />
            {errors.email && <p className="text-xs text-destructive">{errors.email.message}</p>}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="password">密码 *</Label>
            <Input id="password" type="password" {...register('password')} placeholder="至少8个字符" />
            {errors.password && <p className="text-xs text-destructive">{errors.password.message}</p>}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="phone">电话</Label>
            <Input id="phone" {...register('phone')} placeholder="请输入电话号码" />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="max_concurrent">并发上限</Label>
              <Input
                id="max_concurrent"
                type="number"
                {...register('max_concurrent', { valueAsNumber: true })}
              />
              {errors.max_concurrent && (
                <p className="text-xs text-destructive">{errors.max_concurrent.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="daily_limit">日呼叫限额</Label>
              <Input
                id="daily_limit"
                type="number"
                {...register('daily_limit', { valueAsNumber: true })}
              />
              {errors.daily_limit && (
                <p className="text-xs text-destructive">{errors.daily_limit.message}</p>
              )}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="initial_balance">初始余额 (¥)</Label>
            <Input
              id="initial_balance"
              type="number"
              step="0.01"
              {...register('initial_balance', { valueAsNumber: true })}
            />
            {errors.initial_balance && (
              <p className="text-xs text-destructive">{errors.initial_balance.message}</p>
            )}
          </div>

          <SheetFooter className="px-0">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? '创建中...' : '创建客户'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

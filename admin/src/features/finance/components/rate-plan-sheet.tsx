import { useEffect } from 'react'
import { useForm, useFieldArray } from 'react-hook-form'
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
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Plus, Trash2 } from 'lucide-react'

const prefixRateSchema = z.object({
  prefix: z.string().regex(/^\d{3}$/, '请输入3位数字前缀'),
  rate: z.number().positive('费率必须大于0'),
})

const ratePlanSchema = z.object({
  name: z.string().min(1, '请输入模板名称'),
  description: z.string().optional(),
  rate_per_minute: z.number().positive('费率必须大于0'),
  b_leg_rates: z.array(prefixRateSchema),
  billing_increment: z.number().int().min(1, '至少为1秒').default(6),
  min_billing_duration: z.number().int().min(0, '不能为负数').default(0),
})

export type RatePlanFormData = z.infer<typeof ratePlanSchema>

export interface RatePlanForEdit {
  id: string
  name: string
  description?: string
  rate_per_minute: number
  b_leg_rates?: { prefix: string; rate: number }[]
  billing_increment: number
  min_billing_duration?: number
}

interface RatePlanSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ratePlan?: RatePlanForEdit
  onSubmit: (data: RatePlanFormData) => void | Promise<void>
}

export function RatePlanSheet({ open, onOpenChange, ratePlan, onSubmit }: RatePlanSheetProps) {
  const mode = ratePlan ? 'edit' : 'create'

  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<RatePlanFormData>({
    resolver: zodResolver(ratePlanSchema),
    defaultValues: {
      name: '',
      description: '',
      rate_per_minute: 0.15,
      b_leg_rates: [],
      billing_increment: 6,
      min_billing_duration: 0,
    },
  })

  const { fields, append, remove } = useFieldArray({
    control,
    name: 'b_leg_rates',
  })

  useEffect(() => {
    if (open) {
      if (ratePlan) {
        reset({
          name: ratePlan.name,
          description: ratePlan.description ?? '',
          rate_per_minute: ratePlan.rate_per_minute,
          b_leg_rates: ratePlan.b_leg_rates ?? [],
          billing_increment: ratePlan.billing_increment,
          min_billing_duration: ratePlan.min_billing_duration ?? 0,
        })
      } else {
        reset({
          name: '',
          description: '',
          rate_per_minute: 0.15,
          b_leg_rates: [],
          billing_increment: 6,
          min_billing_duration: 0,
        })
      }
    }
  }, [open, ratePlan, reset])

  const onFormSubmit = async (data: RatePlanFormData) => {
    await onSubmit(data)
    reset()
    onOpenChange(false)
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-lg flex flex-col">
        <SheetHeader>
          <SheetTitle>{mode === 'create' ? '新建费率模板' : '编辑费率模板'}</SheetTitle>
          <SheetDescription>
            {mode === 'create' ? '创建新的费率模板用于客户计费' : '修改费率模板信息'}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="flex-1 px-4">
          <form id="rate-plan-form" onSubmit={handleSubmit(onFormSubmit)} className="flex flex-col gap-4 pb-4">
            <div className="space-y-1.5">
              <Label htmlFor="rp-name">模板名称 *</Label>
              <Input id="rp-name" {...register('name')} placeholder="请输入模板名称" />
              {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="rp-desc">描述</Label>
              <Input id="rp-desc" {...register('description')} placeholder="可选描述信息" />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="rp-rate">A路费率/分钟 (CNY) *</Label>
              <Input
                id="rp-rate"
                type="number"
                step="0.01"
                {...register('rate_per_minute', { valueAsNumber: true })}
                placeholder="0.15"
              />
              {errors.rate_per_minute && <p className="text-xs text-destructive">{errors.rate_per_minute.message}</p>}
            </div>

            <Separator />

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label>B路费率表</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => append({ prefix: '', rate: 0 })}
                >
                  <Plus className="mr-1 h-3.5 w-3.5" />
                  添加前缀费率
                </Button>
              </div>

              {fields.length === 0 && (
                <p className="text-xs text-muted-foreground">暂无B路费率规则，点击上方按钮添加</p>
              )}

              {fields.map((field, index) => (
                <div key={field.id} className="flex items-start gap-2">
                  <div className="flex-1 space-y-1">
                    <Input
                      {...register(`b_leg_rates.${index}.prefix`)}
                      placeholder="前缀 (3位)"
                      maxLength={3}
                      className="h-8"
                    />
                    {errors.b_leg_rates?.[index]?.prefix && (
                      <p className="text-xs text-destructive">{errors.b_leg_rates[index].prefix.message}</p>
                    )}
                  </div>
                  <div className="flex-1 space-y-1">
                    <Input
                      type="number"
                      step="0.01"
                      {...register(`b_leg_rates.${index}.rate`, { valueAsNumber: true })}
                      placeholder="费率 (CNY/分)"
                      className="h-8"
                    />
                    {errors.b_leg_rates?.[index]?.rate && (
                      <p className="text-xs text-destructive">{errors.b_leg_rates[index].rate.message}</p>
                    )}
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
                    onClick={() => remove(index)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ))}
            </div>

            <Separator />

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="rp-increment">计费增量 (秒)</Label>
                <Input
                  id="rp-increment"
                  type="number"
                  {...register('billing_increment', { valueAsNumber: true })}
                />
                {errors.billing_increment && (
                  <p className="text-xs text-destructive">{errors.billing_increment.message}</p>
                )}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="rp-min">最低计费时长 (秒)</Label>
                <Input
                  id="rp-min"
                  type="number"
                  {...register('min_billing_duration', { valueAsNumber: true })}
                />
                {errors.min_billing_duration && (
                  <p className="text-xs text-destructive">{errors.min_billing_duration.message}</p>
                )}
              </div>
            </div>
          </form>
        </ScrollArea>
        <SheetFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button type="submit" form="rate-plan-form" disabled={isSubmitting}>
            {isSubmitting ? '保存中...' : mode === 'create' ? '创建模板' : '保存修改'}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

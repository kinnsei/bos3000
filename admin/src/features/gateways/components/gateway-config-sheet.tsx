import { useEffect } from 'react'
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
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Gateway } from '@/lib/api/client'

const baseSchema = z.object({
  name: z.string().min(1, '请输入网关名称'),
  host: z.string().min(1, '请输入 SIP 地址'),
  port: z.coerce.number().int().min(1).max(65535),
  max_concurrent: z.coerce.number().int().min(1, '请输入最大并发数'),
  enabled: z.boolean(),
})

const aLegSchema = baseSchema.extend({
  weight: z.coerce.number().int().min(1).max(100),
})

const bLegSchema = baseSchema.extend({
  prefix: z.string().optional(),
  failover_gateway_id: z.string().optional(),
})

type ALegFormValues = z.infer<typeof aLegSchema>
type BLegFormValues = z.infer<typeof bLegSchema>
type FormValues = ALegFormValues | BLegFormValues

interface GatewayConfigSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  gateway?: Gateway
  gatewayType: 'a_leg' | 'b_leg'
  otherGateways: Gateway[]
  onSubmit: (data: FormValues & { type: 'a_leg' | 'b_leg' }) => void | Promise<void>
}

export function GatewayConfigSheet({
  open,
  onOpenChange,
  gateway,
  gatewayType,
  otherGateways,
  onSubmit,
}: GatewayConfigSheetProps) {
  const isEdit = !!gateway
  const schema = gatewayType === 'a_leg' ? aLegSchema : bLegSchema

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      host: '',
      port: 5060,
      max_concurrent: 100,
      enabled: true,
      ...(gatewayType === 'a_leg' ? { weight: 50 } : { prefix: '', failover_gateway_id: '' }),
    },
  })

  useEffect(() => {
    if (open && gateway) {
      reset({
        name: gateway.name,
        host: gateway.host,
        port: gateway.port,
        max_concurrent: gateway.max_concurrent,
        enabled: gateway.status !== 'disabled',
        ...(gatewayType === 'a_leg'
          ? { weight: gateway.weight }
          : { prefix: gateway.prefix, failover_gateway_id: gateway.failover_gateway_id }),
      })
    } else if (open && !gateway) {
      reset({
        name: '',
        host: '',
        port: 5060,
        max_concurrent: 100,
        enabled: true,
        ...(gatewayType === 'a_leg' ? { weight: 50 } : { prefix: '', failover_gateway_id: '' }),
      })
    }
  }, [open, gateway, gatewayType, reset])

  const onFormSubmit = async (data: FormValues) => {
    await onSubmit({ ...data, type: gatewayType })
    onOpenChange(false)
  }

  const enabledValue = watch('enabled')
  const bLegGateways = otherGateways.filter(
    (gw) => gw.type === 'b_leg' && gw.id !== gateway?.id,
  )

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{isEdit ? '编辑网关' : '添加网关'}</SheetTitle>
          <SheetDescription>
            {gatewayType === 'a_leg' ? 'A 路网关配置' : 'B 路网关配置'}
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit(onFormSubmit)} className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="name">名称</Label>
            <Input id="name" {...register('name')} placeholder="例: GW-SH-01" />
            {errors.name && (
              <p className="text-sm text-destructive">{errors.name.message}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="host">SIP 地址</Label>
              <Input id="host" {...register('host')} placeholder="192.168.1.100" />
              {errors.host && (
                <p className="text-sm text-destructive">{errors.host.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="port">端口</Label>
              <Input id="port" type="number" {...register('port')} />
              {errors.port && (
                <p className="text-sm text-destructive">{errors.port.message}</p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="max_concurrent">最大并发数</Label>
            <Input id="max_concurrent" type="number" {...register('max_concurrent')} />
            {errors.max_concurrent && (
              <p className="text-sm text-destructive">{errors.max_concurrent.message}</p>
            )}
          </div>

          {gatewayType === 'a_leg' && (
            <div className="space-y-2">
              <Label htmlFor="weight">权重 (1-100)</Label>
              <Input
                id="weight"
                type="number"
                min={1}
                max={100}
                {...register('weight' as keyof FormValues)}
              />
              {(errors as Record<string, { message?: string }>).weight && (
                <p className="text-sm text-destructive">
                  {(errors as Record<string, { message?: string }>).weight?.message}
                </p>
              )}
            </div>
          )}

          {gatewayType === 'b_leg' && (
            <>
              <div className="space-y-2">
                <Label htmlFor="prefix">前缀</Label>
                <Input
                  id="prefix"
                  {...register('prefix' as keyof FormValues)}
                  placeholder="130,131,132"
                />
                <p className="text-xs text-muted-foreground">多个前缀用逗号分隔</p>
              </div>

              <div className="space-y-2">
                <Label>容灾网关</Label>
                <Select
                  value={(watch as (name: string) => string)('failover_gateway_id') || ''}
                  onValueChange={(val) =>
                    setValue('failover_gateway_id' as keyof FormValues, val === '_none' ? '' : val)
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择容灾网关（可选）" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="_none">无</SelectItem>
                    {bLegGateways.map((gw) => (
                      <SelectItem key={gw.id} value={gw.id}>
                        {gw.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </>
          )}

          <div className="flex items-center justify-between">
            <Label htmlFor="enabled">启用</Label>
            <Switch
              id="enabled"
              checked={enabledValue as boolean}
              onCheckedChange={(checked) => setValue('enabled', checked)}
            />
          </div>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? '保存中...' : '保存'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod/v4'
import { zodResolver } from '@hookform/resolvers/zod'
import { toast } from 'sonner'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useAuth, useUpdateProfile, useMyDIDs } from '@/lib/api/hooks'

const profileSchema = z.object({
  email: z.email('请输入有效的邮箱地址'),
  phone: z.string().min(1, '请输入手机号码'),
})

type ProfileFormData = z.infer<typeof profileSchema>

export function ProfileForm() {
  const { data: user } = useAuth()
  const updateProfile = useUpdateProfile()
  const { data: didPool = [] } = useMyDIDs()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: { email: '', phone: '' },
  })

  useEffect(() => {
    if (user) {
      reset({ email: user.email, phone: user.phone })
    }
  }, [user, reset])

  const onSubmit = async (data: ProfileFormData) => {
    try {
      await updateProfile.mutateAsync({ phone: data.phone })
      toast.success('资料更新成功')
    } catch {
      toast.error('更新失败')
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>基本信息</CardTitle>
          <CardDescription>管理您的账户基本资料</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="company">公司名称</Label>
              <Input
                id="company"
                readOnly
                value={user?.username ?? ''}
                className="bg-muted"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="email">邮箱</Label>
              <Input id="email" type="email" {...register('email')} />
              {errors.email && (
                <p className="text-sm text-destructive">{errors.email.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="phone">手机号码</Label>
              <Input id="phone" {...register('phone')} />
              {errors.phone && (
                <p className="text-sm text-destructive">{errors.phone.message}</p>
              )}
            </div>

            <Button type="submit" disabled={!isDirty || updateProfile.isPending}>
              保存修改
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>DID 号码池</CardTitle>
          <CardDescription>您的账户可用外显号码</CardDescription>
        </CardHeader>
        <CardContent>
          {didPool.length > 0 ? (
            <ScrollArea className="h-40">
              <div className="space-y-1">
                {didPool.map((did) => (
                  <div
                    key={did}
                    className="rounded-md bg-muted px-3 py-2 font-mono text-sm"
                  >
                    {did}
                  </div>
                ))}
              </div>
            </ScrollArea>
          ) : (
            <p className="text-sm text-muted-foreground">暂无可用 DID 号码</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

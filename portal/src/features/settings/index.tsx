export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">账户设置</h1>
        <p className="text-muted-foreground">管理个人信息、密码和IP白名单</p>
      </div>

      {/* TODO: Profile form, password change, IP whitelist management */}
      <div className="rounded-lg border p-8 text-center text-muted-foreground">
        账户设置功能将在后续迭代中实现
      </div>
    </div>
  )
}

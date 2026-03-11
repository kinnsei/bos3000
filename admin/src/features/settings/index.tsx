import { ConfigEditor } from './components/config-editor'

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">系统设置</h1>
        <p className="text-sm text-muted-foreground">system_configs 可视化编辑</p>
      </div>

      <ConfigEditor />
    </div>
  )
}

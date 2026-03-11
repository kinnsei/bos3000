import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'

type ConfigType = 'string' | 'number' | 'boolean' | 'json'

interface ConfigItem {
  key: string
  description: string
  value: string | number | boolean
  type: ConfigType
  lastModified: string
}

const MOCK_CONFIGS: ConfigItem[] = [
  { key: 'call.max_duration_sec', description: '单次通话最大时长（秒）', value: 3600, type: 'number', lastModified: '2026-03-10 14:30' },
  { key: 'call.retry_attempts', description: '呼叫失败重试次数', value: 3, type: 'number', lastModified: '2026-03-08 09:15' },
  { key: 'call.recording_enabled', description: '是否启用通话录音', value: true, type: 'boolean', lastModified: '2026-03-09 11:00' },
  { key: 'call.codec_preference', description: '编解码器优先级', value: 'PCMU,PCMA,G729', type: 'string', lastModified: '2026-02-20 16:45' },
  { key: 'billing.currency', description: '默认结算货币', value: 'CNY', type: 'string', lastModified: '2026-01-15 10:00' },
  { key: 'billing.tax_rate', description: '税率百分比', value: 6, type: 'number', lastModified: '2026-02-28 08:30' },
  { key: 'billing.auto_deduct', description: '是否自动扣费', value: true, type: 'boolean', lastModified: '2026-03-01 12:00' },
  { key: 'billing.rate_table', description: '费率表配置', value: '{"domestic":0.06,"international":0.15}', type: 'json', lastModified: '2026-03-05 17:20' },
  { key: 'compliance.data_retention_days', description: 'CDR 数据保留天数', value: 180, type: 'number', lastModified: '2026-02-10 09:00' },
  { key: 'compliance.gdpr_enabled', description: '是否启用 GDPR 合规', value: false, type: 'boolean', lastModified: '2026-01-20 14:00' },
  { key: 'compliance.audit_log_level', description: '审计日志级别', value: 'info', type: 'string', lastModified: '2026-03-06 10:30' },
  { key: 'system.maintenance_mode', description: '是否开启维护模式', value: false, type: 'boolean', lastModified: '2026-03-10 22:00' },
  { key: 'system.log_level', description: '系统日志级别', value: 'warn', type: 'string', lastModified: '2026-03-07 08:00' },
  { key: 'system.feature_flags', description: '功能开关配置', value: '{"new_dashboard":true,"v2_api":false}', type: 'json', lastModified: '2026-03-09 15:45' },
]

const CATEGORY_LABELS: Record<string, string> = {
  call: '通话设置',
  billing: '计费设置',
  compliance: '合规设置',
  system: '系统设置',
}

function isValidJson(str: string): boolean {
  try { JSON.parse(str); return true } catch { return false }
}

function ConfigRow({ item, onSave }: { item: ConfigItem; onSave: (key: string, value: ConfigItem['value']) => void }) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<string>(
    item.type === 'boolean' ? '' : String(item.value)
  )
  const [checked, setChecked] = useState(item.type === 'boolean' ? (item.value as boolean) : false)

  const handleSave = () => {
    let finalValue: ConfigItem['value']
    if (item.type === 'number') finalValue = Number(draft)
    else if (item.type === 'boolean') finalValue = checked
    else finalValue = draft
    onSave(item.key, finalValue)
    setEditing(false)
  }

  return (
    <div className="flex items-start gap-4 py-3">
      <div className="flex-1 min-w-0 space-y-1">
        <div className="flex items-center gap-2">
          <code className="text-sm font-mono">{item.key}</code>
          <Badge variant="outline" className="text-xs">{item.type}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">{item.description}</p>
        <p className="text-xs text-muted-foreground">最后修改: {item.lastModified}</p>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        {item.type === 'boolean' ? (
          <Switch checked={checked} onCheckedChange={(v) => { setChecked(v); onSave(item.key, v) }} />
        ) : item.type === 'json' ? (
          <div className="flex flex-col items-end gap-1">
            <div className="flex items-center gap-2">
              <textarea
                className="w-64 h-20 rounded-md border border-input bg-background px-3 py-2 text-sm font-mono resize-none"
                value={editing ? draft : String(item.value)}
                onChange={(e) => { setEditing(true); setDraft(e.target.value) }}
                onFocus={() => { if (!editing) { setDraft(String(item.value)); setEditing(true) } }}
              />
              {editing && (
                <Button size="sm" onClick={handleSave} disabled={!isValidJson(draft)}>保存</Button>
              )}
            </div>
            {editing && (
              <span className={`text-xs ${isValidJson(draft) ? 'text-green-600 dark:text-green-400' : 'text-destructive'}`}>
                {isValidJson(draft) ? 'JSON 格式有效' : 'JSON 格式无效'}
              </span>
            )}
          </div>
        ) : (
          <div className="flex items-center gap-2">
            {item.type === 'number' ? (
              <Input
                type="number"
                className="w-32"
                value={editing ? draft : String(item.value)}
                onChange={(e) => { setEditing(true); setDraft(e.target.value) }}
                onFocus={() => { if (!editing) { setDraft(String(item.value)); setEditing(true) } }}
              />
            ) : (
              <Input
                className="w-48"
                value={editing ? draft : String(item.value)}
                onChange={(e) => { setEditing(true); setDraft(e.target.value) }}
                onFocus={() => { if (!editing) { setDraft(String(item.value)); setEditing(true) } }}
              />
            )}
            {editing && <Button size="sm" onClick={handleSave}>保存</Button>}
          </div>
        )}
      </div>
    </div>
  )
}

export function ConfigEditor() {
  const [configs, setConfigs] = useState(MOCK_CONFIGS)

  const grouped = configs.reduce<Record<string, ConfigItem[]>>((acc, item) => {
    const category = item.key.split('.')[0]
    ;(acc[category] ??= []).push(item)
    return acc
  }, {})

  const handleSave = (key: string, value: ConfigItem['value']) => {
    setConfigs((prev) =>
      prev.map((c) => c.key === key ? { ...c, value, lastModified: new Date().toLocaleString('zh-CN') } : c)
    )
    toast.success('配置已保存', { description: key })
  }

  return (
    <div className="space-y-6">
      {Object.entries(grouped).map(([category, items]) => (
        <Card key={category}>
          <CardHeader>
            <CardTitle className="text-lg">{CATEGORY_LABELS[category] ?? category}</CardTitle>
          </CardHeader>
          <CardContent>
            {items.map((item, i) => (
              <div key={item.key}>
                {i > 0 && <Separator />}
                <ConfigRow item={item} onSave={handleSave} />
              </div>
            ))}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

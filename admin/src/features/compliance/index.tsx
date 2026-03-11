import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { BlacklistTable } from './components/blacklist-table'
import { AuditLogTable } from './components/audit-log-table'

export default function Compliance() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">合规审计</h1>
        <p className="text-muted-foreground">全局黑名单管理和审计日志查询</p>
      </div>

      <Tabs defaultValue="blacklist">
        <TabsList>
          <TabsTrigger value="blacklist">黑名单管理</TabsTrigger>
          <TabsTrigger value="audit">审计日志</TabsTrigger>
        </TabsList>
        <TabsContent value="blacklist" className="mt-4">
          <BlacklistTable />
        </TabsContent>
        <TabsContent value="audit" className="mt-4">
          <AuditLogTable />
        </TabsContent>
      </Tabs>
    </div>
  )
}

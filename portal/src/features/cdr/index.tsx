export default function CDRPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">话单查询</h1>
        <p className="text-muted-foreground">查询历史通话记录和话单详情</p>
      </div>

      {/* TODO: CDR search form, data table with filters */}
      <div className="rounded-lg border p-8 text-center text-muted-foreground">
        话单查询表格和筛选功能将在后续迭代中实现
      </div>
    </div>
  )
}

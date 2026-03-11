export default function WastagePage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">损耗分析</h1>
        <p className="text-muted-foreground">查看通话损耗率和失败原因分布</p>
      </div>

      {/* TODO: Wastage summary cards, trend chart, failure distribution pie chart */}
      <div className="rounded-lg border p-8 text-center text-muted-foreground">
        损耗分析图表和统计将在后续迭代中实现
      </div>
    </div>
  )
}

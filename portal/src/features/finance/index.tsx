export default function FinancePage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">财务中心</h1>
        <p className="text-muted-foreground">查看余额、交易记录和账单</p>
      </div>

      {/* TODO: Balance card, transaction history table */}
      <div className="rounded-lg border p-8 text-center text-muted-foreground">
        余额详情和交易记录将在后续迭代中实现
      </div>
    </div>
  )
}

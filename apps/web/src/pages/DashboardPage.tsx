import { useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowRight, Sparkles } from 'lucide-react'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useFetch } from '@/api/useFetch'
import type { InsightsResponse, Ledger, Transaction, TransactionListResponse } from '@/api/types'
import { cn } from '@/lib/utils'

function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function amountSummary(t: Transaction): string {
  const withUnits = t.postings.find((p) => p.units != null)
  if (!withUnits?.units) return ''
  const n = Number(withUnits.units.number)
  return `${n > 0 ? '+' : ''}${withUnits.units.number} ${withUnits.units.currency}`
}

export function DashboardPage() {
  const { ledgerId = '' } = useParams()
  const month = currentMonth()

  const ledger = useFetch<Ledger>(`/ledgers/${ledgerId}`)
  const totals = useFetch<Record<string, string>>(`/ledgers/${ledgerId}/stats/total?month=${month}`)
  const recent = useFetch<TransactionListResponse>(
    `/ledgers/${ledgerId}/transactions?limit=5&order=desc`,
  )
  const insights = useFetch<InsightsResponse>(`/ledgers/${ledgerId}/ai/insights`)

  const metrics = useMemo(() => {
    const t = totals.data ?? {}
    const income = Number(t.Income ?? 0)
    const expense = Number(t.Expenses ?? 0)
    return [
      { label: '总资产', value: t.Assets ?? '0', note: '按本位币折算' },
      { label: `本月收入（${month}）`, value: String(income), note: month },
      { label: `本月支出（${month}）`, value: String(expense), note: month },
      { label: '本月结余', value: String(income + expense), note: '收入 - 支出' },
    ]
  }, [totals.data, month])

  const trendMax = Math.max(
    1,
    ...(recent.data?.items ?? []).map((t) => Math.abs(Number(t.postings[0]?.units?.number ?? 0))),
  )

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">仪表盘</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            账本：{ledger.data?.name ?? '…'} · 本位币 {ledger.data?.operating_currency ?? 'CNY'} ·
            修订 #{ledger.data?.revision ?? 0}
          </p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/import`} className={buttonVariants({ variant: 'outline' })}>
            导入账单
          </Link>
          <Link to={`/ledgers/${ledgerId}/transactions/new`} className={buttonVariants()}>
            记一笔
          </Link>
        </div>
      </div>

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {totals.loading && Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-24 rounded-xl" />)}
        {metrics.map((m) => (
          <Card key={m.label}>
            <CardHeader>
              <CardTitle className="text-sm font-normal text-muted-foreground">{m.label}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xl font-semibold tabular-nums">{m.value}</p>
              <p className="mt-0.5 text-xs text-muted-foreground">{m.note}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="mt-6 grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">近期交易（最近 5 笔）</CardTitle>
            <CardDescription>最近入账的流水，点击可查看详情</CardDescription>
          </CardHeader>
          <CardContent>
            {recent.loading && <Skeleton className="h-32" />}
            {recent.data?.items.map((t) => (
              <Link
                key={t.id}
                to={`/ledgers/${ledgerId}/transactions/${t.id}`}
                className="flex items-center justify-between border-b py-2 text-sm last:border-0 hover:text-primary"
              >
                <span className="truncate">
                  {t.date.slice(5)} {t.payee ?? t.narration ?? '—'}
                </span>
                <span className="ml-3 shrink-0 font-mono tabular-nums">{amountSummary(t)}</span>
              </Link>
            ))}
            {recent.data && recent.data.items.length === 0 && (
              <p className="py-6 text-center text-sm text-muted-foreground">还没有交易，点右上角「记一笔」</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">月度收支（近 5 笔分布）</CardTitle>
            <CardDescription>按金额可视化最近交易</CardDescription>
          </CardHeader>
          <CardContent>
            {recent.loading && <Skeleton className="h-32" />}
            <div className="flex h-32 items-end gap-2">
              {recent.data?.items.map((t) => (
                <div key={t.id} className="flex flex-1 flex-col items-center gap-1">
                  <span className="text-[10px] text-muted-foreground">
                    {t.postings[0]?.units?.number}
                  </span>
                  <div
                    className="w-full rounded-t bg-primary/70"
                    style={{
                      height: `${Math.max(
                        4,
                        (Math.abs(Number(t.postings[0]?.units?.number ?? 0)) / trendMax) * 100,
                      )}%`,
                    }}
                  />
                  <span className="text-[10px] text-muted-foreground">{t.date.slice(5)}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="flex items-center gap-1.5 text-base">
            <Sparkles className="size-4" /> AI 洞察
          </CardTitle>
        </CardHeader>
        <CardContent>
          {insights.loading && <Skeleton className="h-16" />}
          {insights.error && <p className="text-sm text-muted-foreground">AI 洞察暂不可用：{insights.error}</p>}
          {insights.data && insights.data.insights.length === 0 && (
            <p className="text-sm text-muted-foreground">暂未发现异常，账本状态良好。</p>
          )}
          <div className="flex flex-col gap-1.5">
            {insights.data?.insights.map((insight, i) => (
              <p key={i} className="text-sm text-destructive">
                {insight.message}
              </p>
            ))}
          </div>
          {insights.data && insights.data.insights.length > 0 && (
            <Link
              to={`/ledgers/${ledgerId}/ai`}
              className={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'mt-3')}
            >
              查看 AI 洞察 <ArrowRight />
            </Link>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

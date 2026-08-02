import { useParams } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { LedgerNav } from '@/components/LedgerNav'
import { useFetch } from '@/api/useFetch'
import type { StatsPoint } from '@/api/types'

export function StatsPage() {
  const { ledgerId = '' } = useParams()
  const total = useFetch<Record<string, string>>(`/ledgers/${ledgerId}/stats/total`)
  const trend = useFetch<StatsPoint[]>(`/ledgers/${ledgerId}/stats/account-trend?type=month`)

  const max = Math.max(
    1,
    ...(trend.data ?? []).map((p) => Math.abs(Number(p.amount ?? 0))),
  )

  return (
    <div>
      <h1 className="text-xl font-semibold">统计</h1>
      <LedgerNav />

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {total.loading && Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-24" />)}
        {total.data &&
          Object.entries(total.data).map(([type, amount]) => (
            <Card key={type}>
              <CardHeader>
                <CardTitle className="text-sm text-muted-foreground">{type}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-xl font-semibold tabular-nums">{amount}</p>
              </CardContent>
            </Card>
          ))}
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">月度趋势</CardTitle>
          <CardDescription>按 month 汇总的收支趋势（本位币）</CardDescription>
        </CardHeader>
        <CardContent>
          {trend.loading && <Skeleton className="h-40" />}
          {trend.error && <p className="text-sm text-destructive">加载失败：{trend.error}</p>}
          <div className="flex h-40 items-end gap-2">
            {trend.data?.map((p) => (
              <div key={p.date} className="flex flex-1 flex-col items-center gap-1">
                <span className="text-[10px] text-muted-foreground">{p.amount}</span>
                <div
                  className="w-full rounded-t bg-primary/70"
                  style={{ height: `${Math.max(4, (Math.abs(Number(p.amount ?? 0)) / max) * 100)}%` }}
                />
                <span className="text-[10px] text-muted-foreground">{p.date}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

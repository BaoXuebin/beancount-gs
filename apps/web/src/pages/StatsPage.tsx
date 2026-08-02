import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useFetch } from '@/api/useFetch'
import type { StatsPayee, StatsPoint } from '@/api/types'
import { cn } from '@/lib/utils'
import { LoadingHint } from '@/components/LoadingHint'

type Tab = 'trend' | 'share' | 'payee' | 'sankey'

const tabs: { key: Tab; label: string }[] = [
  { key: 'trend', label: '趋势' },
  { key: 'share', label: '占比' },
  { key: 'payee', label: '收款方' },
  { key: 'sankey', label: '桑基' },
]

export function StatsPage() {
  const { ledgerId = '' } = useParams()
  const [tab, setTab] = useState<Tab>('trend')
  const [month, setMonth] = useState('')
  const [account, setAccount] = useState('')

  const params = new URLSearchParams()
  if (month) params.set('month', month)
  if (account) params.set('account', account)
  const qs = params.toString() ? `?${params}` : ''

  const total = useFetch<Record<string, string>>(`/ledgers/${ledgerId}/stats/total${qs}`)
  const trend = useFetch<StatsPoint[]>(`/ledgers/${ledgerId}/stats/account-trend?type=month${qs ? `&${qs}` : ''}`)
  const payees = useFetch<StatsPayee[]>(`/ledgers/${ledgerId}/stats/payee${qs}`)
  const flow = useFetch<{ nodes: { name: string }[]; links: { source: number; target: number; value: string }[] }>(
    `/ledgers/${ledgerId}/stats/account-flow${qs}`,
  )

  const max = Math.max(1, ...(trend.data ?? []).map((p) => Math.abs(Number(p.amount ?? 0))))
  const totalMax = Math.max(
    1,
    ...Object.entries(total.data ?? {}).map(([, v]) => Math.abs(Number(v ?? 0))),
  )

  return (
    <div>
      {(total.loading || trend.loading || payees.loading || flow.loading) && (
        <LoadingHint className="mb-2" />
      )}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">统计</h1>
          <p className="mt-1 text-sm text-muted-foreground">趋势 / 占比 / 收款方 / 桑基</p>
        </div>
        <div className="flex flex-wrap items-end gap-2">
          <div className="grid gap-1.5">
            <Label>月份</Label>
            <Input type="month" value={month} onChange={(e) => setMonth(e.target.value)} className="w-36" />
          </div>
          <div className="grid gap-1.5">
            <Label>账户前缀</Label>
            <Input
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              placeholder="如 Expenses"
              className="w-40"
            />
          </div>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap gap-1 rounded-lg border bg-muted/40 p-1">
        {tabs.map((t) => (
          <button
            key={t.key}
            type="button"
            className={cn(
              'rounded-md px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground',
              tab === t.key && 'bg-background font-medium text-foreground shadow-sm',
            )}
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'trend' && (
        <Card className="mt-4">
          <CardHeader>
            <CardTitle className="text-base">账户趋势</CardTitle>
            <CardDescription>按月份汇总的余额 / 收支趋势（本位币）</CardDescription>
          </CardHeader>
          <CardContent>
            {trend.loading && <Skeleton className="h-48" />}
            {trend.error && <p className="text-sm text-destructive">加载失败：{trend.error}</p>}
            <div className="flex h-48 items-end gap-2">
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
            {trend.data && trend.data.length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground">当前筛选条件下没有数据</p>
            )}
          </CardContent>
        </Card>
      )}

      {tab === 'share' && (
        <Card className="mt-4">
          <CardHeader>
            <CardTitle className="text-base">账户占比</CardTitle>
            <CardDescription>按账户根类型汇总（Assets / Liabilities / Income / Expenses / Equity）</CardDescription>
          </CardHeader>
          <CardContent>
            {total.loading && <Skeleton className="h-48" />}
            {total.error && <p className="text-sm text-destructive">加载失败：{total.error}</p>}
            <div className="flex flex-col gap-3">
              {Object.entries(total.data ?? {}).map(([type, value]) => (
                <div key={type} className="flex items-center gap-3">
                  <span className="w-24 shrink-0 text-sm">{type}</span>
                  <div className="h-6 flex-1 overflow-hidden rounded-md bg-muted">
                    <div
                      className="flex h-full items-center rounded-md bg-primary/70 px-2 text-xs text-primary-foreground"
                      style={{ width: `${(Math.abs(Number(value)) / totalMax) * 100}%` }}
                    >
                      {value}
                    </div>
                  </div>
                </div>
              ))}
            </div>
            {total.data && Object.keys(total.data).length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground">当前筛选条件下没有数据</p>
            )}
          </CardContent>
        </Card>
      )}

      {tab === 'payee' && (
        <Card className="mt-4">
          <CardHeader>
            <CardTitle className="text-base">收款方统计</CardTitle>
            <CardDescription>按金额降序排列</CardDescription>
          </CardHeader>
          <CardContent>
            {payees.loading && <Skeleton className="h-48" />}
            {payees.error && <p className="text-sm text-destructive">加载失败：{payees.error}</p>}
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>收款方</TableHead>
                  <TableHead className="text-right">笔数</TableHead>
                  <TableHead className="text-right">金额</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(payees.data ?? []).map((p) => (
                  <TableRow key={p.payee}>
                    <TableCell>{p.payee}</TableCell>
                    <TableCell className="text-right">{p.count}</TableCell>
                    <TableCell className="text-right font-mono">{p.amount}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {payees.data && payees.data.length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground">当前筛选条件下没有数据</p>
            )}
          </CardContent>
        </Card>
      )}

      {tab === 'sankey' && (
        <Card className="mt-4">
          <CardHeader>
            <CardTitle className="text-base">资金流向（桑基数据）</CardTitle>
            <CardDescription>账户间资金流转，节点与边</CardDescription>
          </CardHeader>
          <CardContent>
            {flow.loading && <Skeleton className="h-48" />}
            {flow.error && <p className="text-sm text-destructive">加载失败：{flow.error}</p>}
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>来源</TableHead>
                  <TableHead>去向</TableHead>
                  <TableHead className="text-right">金额</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(flow.data?.links ?? []).map((link, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-mono">
                      {flow.data?.nodes[link.source]?.name ?? `#${link.source}`}
                    </TableCell>
                    <TableCell className="font-mono">
                      {flow.data?.nodes[link.target]?.name ?? `#${link.target}`}
                    </TableCell>
                    <TableCell className="text-right font-mono">{link.value}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {flow.data && flow.data.links.length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground">当前筛选条件下没有资金流</p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

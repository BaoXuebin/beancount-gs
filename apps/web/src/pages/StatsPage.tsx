import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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

type Tab = 'balance' | 'income' | 'trend' | 'share' | 'payee' | 'sankey'

// Base UI Select 把空字符串值视为未选中（触发器无回显），用哨兵值表示「全部月份」
const ALL_MONTHS = 'all'

const tabs: { key: Tab; label: string }[] = [
  { key: 'balance', label: '资产负债' },
  { key: 'income', label: '收入支出' },
  { key: 'trend', label: '趋势' },
  { key: 'share', label: '占比' },
  { key: 'payee', label: '收款方' },
  { key: 'sankey', label: '桑基' },
]

function roundAmount(n: number): string {
  return String(Math.round(n * 100) / 100)
}

/** 合并两个趋势序列（同日期求和），按日期升序返回 */
function mergePoints(
  a?: StatsPoint[] | null,
  b?: StatsPoint[] | null,
): Array<{ date: string; first: number; second: number }> {
  const byDate = new Map<string, { first: number; second: number }>()
  const collect = (points: StatsPoint[], slot: 'first' | 'second') => {
    for (const p of points) {
      if (!p.date) continue
      const entry = byDate.get(p.date) ?? { first: 0, second: 0 }
      entry[slot] += Number(p.amount ?? 0)
      byDate.set(p.date, entry)
    }
  }
  collect(a ?? [], 'first')
  collect(b ?? [], 'second')
  return Array.from(byDate.entries())
    .sort(([x], [y]) => x.localeCompare(y))
    .map(([date, v]) => ({ date, ...v }))
}

/** 双序列柱状图（first 主色 / second 警示色），悬浮显示精确数值 */
function DualSeriesBars({
  items,
  max,
  firstTitle,
  secondTitle,
}: {
  items: Array<{ date: string; first: number; second: number }>
  max: number
  firstTitle: (v: number) => string
  secondTitle: (v: number) => string
}) {
  return (
    <div className="flex h-48 items-end gap-2">
      {items.map((it) => (
        <div
          key={it.date}
          className="flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-1"
          title={`${it.date}\n${firstTitle(it.first)}\n${secondTitle(it.second)}`}
        >
          <div className="flex h-full w-full items-end justify-center gap-0.5">
            <div
              className="w-1/3 rounded-t bg-primary/70"
              style={{
                height: `${Math.max(it.first !== 0 ? 4 : 0, (Math.abs(it.first) / max) * 100)}%`,
              }}
            />
            <div
              className="w-1/3 rounded-t bg-destructive/60"
              style={{
                height: `${Math.max(it.second !== 0 ? 4 : 0, (Math.abs(it.second) / max) * 100)}%`,
              }}
            />
          </div>
          <span className="max-w-full truncate text-[10px] text-muted-foreground">{it.date}</span>
        </div>
      ))}
    </div>
  )
}

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
  const months = useFetch<string[]>(`/ledgers/${ledgerId}/months`)

  // 月份按年份分组（months.data 倒序，分组保持顺序）
  const monthGroups = useMemo(() => {
    const groups: Array<{ year: string; months: string[] }> = []
    for (const m of months.data ?? []) {
      const year = m.slice(0, 4)
      const last = groups[groups.length - 1]
      if (last && last.year === year) last.months.push(m)
      else groups.push({ year, months: [m] })
    }
    return groups
  }, [months.data])

  const max = Math.max(1, ...(trend.data ?? []).map((p) => Math.abs(Number(p.amount ?? 0))))
  const totalMax = Math.max(
    1,
    ...Object.entries(total.data ?? {}).map(([, v]) => Math.abs(Number(v ?? 0))),
  )

  // —— 资产负债表：不受上方筛选影响，始终统计全部账目（按需加载）——
  const bsActive = tab === 'balance'
  const bsTotal = useFetch<Record<string, string>>(bsActive ? `/ledgers/${ledgerId}/stats/total` : null)
  const bsAssets = useFetch<StatsPoint[]>(
    bsActive ? `/ledgers/${ledgerId}/stats/account-trend?type=month&account=Assets` : null,
  )
  const bsLiabs = useFetch<StatsPoint[]>(
    bsActive ? `/ledgers/${ledgerId}/stats/account-trend?type=month&account=Liabilities` : null,
  )
  // beancount 中负债为负数：展示取绝对值；净资产 = 资产 + 负债账面值
  const bsSummary = useMemo(() => {
    const assets = Math.abs(Number(bsTotal.data?.Assets ?? 0))
    const liabSigned = Number(bsTotal.data?.Liabilities ?? 0)
    return { assets, liabilities: Math.abs(liabSigned), net: assets + liabSigned }
  }, [bsTotal.data])
  const bsLoading = bsTotal.loading || bsAssets.loading || bsLiabs.loading
  const bsError = bsTotal.error ?? bsAssets.error ?? bsLiabs.error
  const bsMonthly = useMemo(
    () => mergePoints(bsAssets.data, bsLiabs.data),
    [bsAssets.data, bsLiabs.data],
  )
  const bsMax = Math.max(1, ...bsMonthly.flatMap((m) => [Math.abs(m.first), Math.abs(m.second)]))

  // —— 收入支出表：跟随月份筛选（账户固定 Income / Expenses，按需加载）——
  const [ieGranularity, setIeGranularity] = useState<'day' | 'month'>('month')
  const ieActive = tab === 'income'
  const ieQs = month ? `?month=${encodeURIComponent(month)}` : ''
  const ieMonthParam = month ? `&month=${encodeURIComponent(month)}` : ''
  const ieTotal = useFetch<Record<string, string>>(
    ieActive ? `/ledgers/${ledgerId}/stats/total${ieQs}` : null,
  )
  const ieExpensePayees = useFetch<StatsPayee[]>(
    ieActive ? `/ledgers/${ledgerId}/stats/payee?account=Expenses${ieMonthParam}` : null,
  )
  const ieIncomePayees = useFetch<StatsPayee[]>(
    ieActive ? `/ledgers/${ledgerId}/stats/payee?account=Income${ieMonthParam}` : null,
  )
  const ieIncomeTrend = useFetch<StatsPoint[]>(
    ieActive
      ? `/ledgers/${ledgerId}/stats/account-trend?type=${ieGranularity}&account=Income${ieMonthParam}`
      : null,
  )
  const ieExpenseTrend = useFetch<StatsPoint[]>(
    ieActive
      ? `/ledgers/${ledgerId}/stats/account-trend?type=${ieGranularity}&account=Expenses${ieMonthParam}`
      : null,
  )
  const currency =
    bsAssets.data?.find((p) => p.operating_currency)?.operating_currency ??
    ieIncomeTrend.data?.find((p) => p.operating_currency)?.operating_currency ??
    ''
  const ieSummary = useMemo(() => {
    const income = Math.abs(Number(ieTotal.data?.Income ?? 0))
    const expenses = Math.abs(Number(ieTotal.data?.Expenses ?? 0))
    return { income, expenses, net: income - expenses }
  }, [ieTotal.data])
  const ieDataLoading = ieTotal.loading || ieExpensePayees.loading || ieIncomePayees.loading
  const ieDataError = ieTotal.error ?? ieExpensePayees.error ?? ieIncomePayees.error
  const ieTrendLoading = ieIncomeTrend.loading || ieExpenseTrend.loading
  const ieTrendError = ieIncomeTrend.error ?? ieExpenseTrend.error
  const ieSeries = useMemo(
    () => mergePoints(ieIncomeTrend.data, ieExpenseTrend.data),
    [ieIncomeTrend.data, ieExpenseTrend.data],
  )
  const ieMax = Math.max(1, ...ieSeries.flatMap((m) => [Math.abs(m.first), Math.abs(m.second)]))

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">统计</h1>
          <p className="mt-1 text-sm text-muted-foreground">资产负债 / 收入支出 / 趋势 / 占比 / 收款方 / 桑基</p>
        </div>
        <div className="flex flex-wrap items-end gap-2">
          <div className="grid gap-1">
            <Label>月份</Label>
            <Select
              value={month || ALL_MONTHS}
              onValueChange={(value) => value != null && setMonth(value === ALL_MONTHS ? '' : value)}
            >
              <SelectTrigger className="w-32">
                {/* 弹层未打开时 Item 未注册，需手动映射哨兵值的回显文案 */}
                <SelectValue>
                  {(value: string) => (value === ALL_MONTHS ? '全部月份' : value)}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_MONTHS}>全部月份</SelectItem>
                {monthGroups.map((g) => (
                  <SelectGroup key={g.year}>
                    <SelectLabel>{g.year} 年</SelectLabel>
                    <SelectItem value={g.year}>{g.year} 全年</SelectItem>
                    {g.months.map((m) => (
                      <SelectItem key={m} value={m}>
                        {m}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1">
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

      {tab === 'balance' && (
        <>
          <div className="mt-4 grid gap-3 sm:grid-cols-3">
            <Card>
              <CardContent className="py-4">
                {bsLoading ? (
                  <Skeleton className="h-12 w-28" />
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">资产总额</p>
                    <p className="mt-1 text-2xl font-semibold tabular-nums">
                      {roundAmount(bsSummary.assets)}
                      {currency && (
                        <span className="ml-1 text-sm font-normal text-muted-foreground">{currency}</span>
                      )}
                    </p>
                  </>
                )}
              </CardContent>
            </Card>
            <Card>
              <CardContent className="py-4">
                {bsLoading ? (
                  <Skeleton className="h-12 w-28" />
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">负债总额</p>
                    <p className="mt-1 text-2xl font-semibold tabular-nums">
                      {roundAmount(bsSummary.liabilities)}
                      {currency && (
                        <span className="ml-1 text-sm font-normal text-muted-foreground">{currency}</span>
                      )}
                    </p>
                  </>
                )}
              </CardContent>
            </Card>
            <Card>
              <CardContent className="py-4">
                {bsLoading ? (
                  <Skeleton className="h-12 w-28" />
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">净资产</p>
                    <p
                      className={cn(
                        'mt-1 text-2xl font-semibold tabular-nums',
                        bsSummary.net < 0 && 'text-destructive',
                      )}
                    >
                      {roundAmount(bsSummary.net)}
                      {currency && (
                        <span className="ml-1 text-sm font-normal text-muted-foreground">{currency}</span>
                      )}
                    </p>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
          <Card className="mt-3">
            <CardHeader>
              <CardTitle className="text-base">资产负债月度变化</CardTitle>
              <CardDescription>各月资产 / 负债分录净额（本位币），负债负值代表负债增加</CardDescription>
            </CardHeader>
            <CardContent>
              {bsLoading ? (
                <Skeleton className="h-48" />
              ) : bsError ? (
                <p className="text-sm text-destructive">加载失败：{bsError}</p>
              ) : bsMonthly.length === 0 ? (
                <p className="py-8 text-center text-sm text-muted-foreground">没有资产负债数据</p>
              ) : (
                <>
                  <div className="mb-3 flex items-center gap-4 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1.5">
                      <span className="size-2.5 rounded-sm bg-primary/70" />资产变化
                    </span>
                    <span className="flex items-center gap-1.5">
                      <span className="size-2.5 rounded-sm bg-destructive/60" />负债变化
                    </span>
                  </div>
                  <DualSeriesBars
                    items={bsMonthly}
                    max={bsMax}
                    firstTitle={(v) => `资产变化 ${roundAmount(v)}`}
                    secondTitle={(v) => `负债变化（账面） ${roundAmount(v)}`}
                  />
                </>
              )}
            </CardContent>
          </Card>
        </>
      )}

      {tab === 'income' && (
        <>
          <div className="mt-4 grid gap-3 sm:grid-cols-3">
            <Card>
              <CardContent className="py-4">
                {ieDataLoading ? (
                  <Skeleton className="h-12 w-28" />
                ) : ieDataError ? (
                  <p className="text-sm text-destructive">加载失败：{ieDataError}</p>
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">收入总额</p>
                    <p className="mt-1 text-2xl font-semibold tabular-nums">
                      {roundAmount(ieSummary.income)}
                      {currency && (
                        <span className="ml-1 text-sm font-normal text-muted-foreground">{currency}</span>
                      )}
                    </p>
                  </>
                )}
              </CardContent>
            </Card>
            <Card>
              <CardContent className="py-4">
                {ieDataLoading ? (
                  <Skeleton className="h-12 w-28" />
                ) : ieDataError ? (
                  <p className="text-sm text-destructive">加载失败：{ieDataError}</p>
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">支出总额</p>
                    <p className="mt-1 text-2xl font-semibold tabular-nums">
                      {roundAmount(ieSummary.expenses)}
                      {currency && (
                        <span className="ml-1 text-sm font-normal text-muted-foreground">{currency}</span>
                      )}
                    </p>
                  </>
                )}
              </CardContent>
            </Card>
            <Card>
              <CardContent className="py-4">
                {ieDataLoading ? (
                  <Skeleton className="h-12 w-28" />
                ) : ieDataError ? (
                  <p className="text-sm text-destructive">加载失败：{ieDataError}</p>
                ) : (
                  <>
                    <p className="text-sm text-muted-foreground">结余（收入 − 支出）</p>
                    <p
                      className={cn(
                        'mt-1 text-2xl font-semibold tabular-nums',
                        ieSummary.net < 0 && 'text-destructive',
                      )}
                    >
                      {roundAmount(ieSummary.net)}
                      {currency && (
                        <span className="ml-1 text-sm font-normal text-muted-foreground">{currency}</span>
                      )}
                    </p>
                  </>
                )}
              </CardContent>
            </Card>
          </div>

          <div className="mt-3 grid gap-3 lg:grid-cols-2">
            {(
              [
                { title: '支出明细', desc: '按收款方汇总，金额降序（前 10 位）', data: ieExpensePayees.data, empty: '当前周期没有支出记录' },
                { title: '收入明细', desc: '按收款方汇总，金额降序（前 10 位）', data: ieIncomePayees.data, empty: '当前周期没有收入记录' },
              ] as const
            ).map((section) => (
              <Card key={section.title}>
                <CardHeader>
                  <CardTitle className="text-base">{section.title}</CardTitle>
                  <CardDescription>{section.desc}</CardDescription>
                </CardHeader>
                <CardContent>
                  {ieDataLoading ? (
                    <div className="flex flex-col gap-2.5">
                      {Array.from({ length: 5 }).map((_, i) => (
                        <Skeleton key={i} className="h-6 w-full" />
                      ))}
                    </div>
                  ) : section.data && section.data.length === 0 ? (
                    <p className="py-6 text-center text-sm text-muted-foreground">{section.empty}</p>
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>收款方</TableHead>
                          <TableHead className="text-right">笔数</TableHead>
                          <TableHead className="text-right">金额</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(section.data ?? []).slice(0, 10).map((p) => (
                          <TableRow key={p.payee}>
                            <TableCell>{p.payee}</TableCell>
                            <TableCell className="text-right">{p.count}</TableCell>
                            <TableCell className="text-right font-mono">{p.amount}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>

          <Card className="mt-3">
            <CardHeader>
              <CardTitle className="text-base">收支趋势统计</CardTitle>
              <CardDescription>
                {month || '全部时间'}内收入与支出的{ieGranularity === 'day' ? '每日' : '每月'}净额（本位币）
              </CardDescription>
            </CardHeader>
            <CardContent>
              {ieTrendLoading ? (
                <Skeleton className="h-48" />
              ) : ieTrendError ? (
                <p className="text-sm text-destructive">加载失败：{ieTrendError}</p>
              ) : ieSeries.length === 0 ? (
                <p className="py-8 text-center text-sm text-muted-foreground">当前周期没有收支数据</p>
              ) : (
                <>
                  <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1.5">
                        <span className="size-2.5 rounded-sm bg-primary/70" />收入
                      </span>
                      <span className="flex items-center gap-1.5">
                        <span className="size-2.5 rounded-sm bg-destructive/60" />支出
                      </span>
                    </div>
                    <div className="flex gap-1 rounded-lg border bg-muted/40 p-1">
                      {(['day', 'month'] as const).map((g) => (
                        <button
                          key={g}
                          type="button"
                          className={cn(
                            'rounded-md px-2.5 py-1 text-xs transition-colors',
                            ieGranularity === g
                              ? 'bg-background font-medium shadow-sm'
                              : 'text-muted-foreground hover:text-foreground',
                          )}
                          onClick={() => setIeGranularity(g)}
                        >
                          {g === 'day' ? '按日' : '按月'}
                        </button>
                      ))}
                    </div>
                  </div>
                  <DualSeriesBars
                    items={ieSeries}
                    max={ieMax}
                    firstTitle={(v) => `收入 ${roundAmount(v)}`}
                    secondTitle={(v) => `支出 ${roundAmount(v)}`}
                  />
                </>
              )}
            </CardContent>
          </Card>
        </>
      )}

      {tab === 'trend' && (
        <Card className="mt-4">
          <CardHeader>
            <CardTitle className="text-base">账户趋势</CardTitle>
            <CardDescription>按月份汇总的余额 / 收支趋势（本位币）</CardDescription>
          </CardHeader>
          <CardContent>
            {trend.loading ? (
              <Skeleton className="h-48" />
            ) : trend.error ? (
              <p className="text-sm text-destructive">加载失败：{trend.error}</p>
            ) : (
              <>
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
              </>
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
            {total.loading ? (
              <Skeleton className="h-48" />
            ) : total.error ? (
              <p className="text-sm text-destructive">加载失败：{total.error}</p>
            ) : (
              <>
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
              </>
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
            {payees.loading ? (
              <div className="flex flex-col gap-2.5">
                {Array.from({ length: 8 }).map((_, i) => (
                  <Skeleton key={i} className="h-6 w-full" />
                ))}
              </div>
            ) : payees.error ? (
              <p className="text-sm text-destructive">加载失败：{payees.error}</p>
            ) : (
              <>
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
              </>
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
            {flow.loading ? (
              <div className="flex flex-col gap-2.5">
                {Array.from({ length: 8 }).map((_, i) => (
                  <Skeleton key={i} className="h-6 w-full" />
                ))}
              </div>
            ) : flow.error ? (
              <p className="text-sm text-destructive">加载失败：{flow.error}</p>
            ) : (
              <>
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
              </>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

import { useCallback, useMemo, useState, type KeyboardEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Eye, Loader2, Pencil, RefreshCw, Sparkles, Trash2, type LucideIcon } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { ApiError, request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { Ledger, Transaction, TransactionListResponse, TransactionTemplate } from '@/api/types'
import { cn } from '@/lib/utils'
import { ACCOUNT_TYPE_META, accountIcon, type AccountType } from '@/lib/accountMeta'
import { TransactionViewDialog } from '@/components/TransactionViewDialog'
import { TransactionEditDialog } from '@/components/TransactionEditDialog'
import { TemplatesDialog } from '@/components/TemplatesDialog'
import { AiRecordDialog } from '@/components/AiRecordDialog'

const PAGE_SIZE = 10000

// Base UI Select 把空字符串值视为未选中（触发器无回显），用哨兵值表示「全部月份」
const ALL_MONTHS = 'all'

// 账户类型筛选（与账户页 typeTabs 一致，标签复用 ACCOUNT_TYPE_META）
const ACCOUNT_TYPE_TABS = [
  { key: '', label: '全部' },
  { key: 'Assets', label: '资产' },
  { key: 'Liabilities', label: '负债' },
  { key: 'Income', label: '收入' },
  { key: 'Expenses', label: '费用' },
  { key: 'Equity', label: '权益' },
]

function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function amountSummary(t: Transaction): string {
  const withUnits = t.postings.find((p) => p.units != null)
  if (!withUnits?.units) return ''
  return `${withUnits.units.number} ${withUnits.units.currency}`
}

/** 取交易的代表性账户：优先支出腿，其次收入腿，最后第一条分录。 */
function representativeAccount(t: Transaction): string {
  const preferred =
    t.postings.find((p) => p.account.startsWith('Expenses:')) ??
    t.postings.find((p) => p.account.startsWith('Income:')) ??
    t.postings[0]
  return preferred?.account ?? ''
}

function TransactionIcon({ t }: { t: Transaction }) {
  const parts = representativeAccount(t).split(':').filter(Boolean)
  const type = (parts[0] ?? '') as AccountType
  const meta = ACCOUNT_TYPE_META[type]
  if (!meta) {
    const Fallback = ACCOUNT_TYPE_META.Expenses.icon
    return <Fallback className="size-4 shrink-0 text-muted-foreground" />
  }
  const Icon: LucideIcon = accountIcon(parts.slice(1).join(':'), type, false)
  return <Icon className={cn('size-4 shrink-0', meta.iconClass)} />
}

export function TransactionsPage() {
  const { ledgerId = '' } = useParams()
  const [q, setQ] = useState('')
  const [month, setMonth] = useState(currentMonth())
  const [account, setAccount] = useState('')
  const [accountType, setAccountType] = useState('')
  const [tag, setTag] = useState('')
  const [filters, setFilters] = useState({
    q: '',
    month: currentMonth(),
    account: '',
    accountType: '',
    tag: '',
  })
  const [conflict, setConflict] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<Transaction | null>(null)
  const [viewing, setViewing] = useState<Transaction | null>(null)
  const [editing, setEditing] = useState<Transaction | null>(null)
  const [templatesOpen, setTemplatesOpen] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [createInitial, setCreateInitial] = useState<TransactionTemplate | null>(null)
  const [aiOpen, setAiOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [reloadKey, setReloadKey] = useState(0)
  const ledger = useFetch<Ledger>(`/ledgers/${ledgerId}`)
  const months = useFetch<string[]>(`/ledgers/${ledgerId}/months`)

  // 实际生效月份：默认当月，但当月无交易时回退到最近有交易的月份（纯派生，避免二次请求）
  const resolvedMonth =
    months.data && filters.month === currentMonth() && !months.data.includes(filters.month)
      ? (months.data[0] ?? filters.month)
      : filters.month

  const query = useCallback(() => {
    const params = new URLSearchParams()
    params.set('limit', String(PAGE_SIZE))
    if (filters.q) params.set('q', filters.q)
    if (resolvedMonth) {
      // 纯 4 位数字视为整年，转换为日期范围查询
      if (/^\d{4}$/.test(resolvedMonth)) {
        params.set('from', `${resolvedMonth}-01-01`)
        params.set('to', `${resolvedMonth}-12-31`)
      } else {
        params.set('month', resolvedMonth)
      }
    }
    if (filters.account) params.set('account', filters.account)
    if (filters.accountType) params.set('account_type', filters.accountType)
    if (filters.tag) params.set('tag', filters.tag)
    return params.toString()
  }, [filters, resolvedMonth])

  // 等月份列表就绪后再请求交易，保证首次请求就带正确筛选
  const url = months.data
    ? `/ledgers/${ledgerId}/transactions?${query()}`
    : null
  // 通过 key 变化触发 useFetch 重新请求，避免本地缓存
  const txns = useFetch<TransactionListResponse>(url, undefined, reloadKey)

  const applyFilters = () => {
    setFilters({
      q: q.trim(),
      month,
      account: account.trim(),
      accountType,
      tag: tag.trim(),
    })
  }

  // 点击类型标签立即生效
  const pickAccountType = (key: string) => {
    setAccountType(key)
    setFilters((f) => ({ ...f, accountType: key }))
  }

  // 回车触发查询；输入法组词回车（isComposing）不触发
  const onFilterEnter = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.nativeEvent.isComposing) applyFilters()
  }

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

  const confirmDelete = async () => {
    if (!deleting) return
    setBusy(true)
    setError(null)
    setConflict(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/transactions/${deleting.id}`, {
        method: 'DELETE',
        revision: rev.revision,
      })
      setDeleting(null)
      setReloadKey((k) => k + 1)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setConflict(`账本已被他人修改（409），当前修订号 ${(err.details as { current_revision?: number })?.current_revision ?? '?'}`)
        setDeleting(null)
        setReloadKey((k) => k + 1)
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">交易</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            共 {txns.data?.total ?? 0} 笔 · 修订 #{ledger.data?.revision ?? 0}
          </p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/import`} className={buttonVariants({ variant: 'outline' })}>
            导入
          </Link>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => setTemplatesOpen(true)}
          >
            模板
          </button>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => setAiOpen(true)}
          >
            <Sparkles /> AI 记录
          </button>
          <button
            type="button"
            className={buttonVariants()}
            onClick={() => {
              setCreateInitial(null)
              setCreateOpen(true)
            }}
          >
            记一笔
          </button>
        </div>
      </div>

      {conflict && (
        <div className="mt-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          <span>⚠ {conflict}</span>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline', size: 'sm' })}
            onClick={() => {
              setConflict(null)
              setReloadKey((k) => k + 1)
            }}
          >
            <RefreshCw /> 刷新
          </button>
        </div>
      )}

      <Card className="mt-4">
        <CardContent className="pt-4">
          <div className="flex flex-wrap items-end gap-2">
            <div className="grid gap-1">
              <Label>搜索</Label>
              <Input
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder="收款方 / 描述 / 金额"
                className="w-44"
                onKeyDown={onFilterEnter}
              />
            </div>
          <div className="grid gap-1">
            <Label>月份</Label>
            <Select
              value={resolvedMonth || ALL_MONTHS}
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
            <Label>账户</Label>
            <Input
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              placeholder="如 Expenses:Food"
              className="w-44"
              onKeyDown={onFilterEnter}
            />
          </div>
          <div className="grid gap-1">
            <Label>标签</Label>
            <Input
              value={tag}
              onChange={(e) => setTag(e.target.value)}
              placeholder="如 Food"
              className="w-32"
              onKeyDown={onFilterEnter}
            />
          </div>
          <div className="flex shrink-0 gap-2">
            <button
              type="button"
              className={buttonVariants()}
              onClick={applyFilters}
              disabled={txns.loading}
            >
              {txns.loading && <Loader2 className="animate-spin" />}
              应用
            </button>
            <button
              type="button"
              className={buttonVariants({ variant: 'ghost' })}
              onClick={() => {
                setQ('')
                setMonth(currentMonth())
                setAccount('')
                setAccountType('')
                setTag('')
                setFilters({ q: '', month: currentMonth(), account: '', accountType: '', tag: '' })
                setReloadKey((k) => k + 1)
              }}
            >
              清除
            </button>
          </div>
          </div>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardContent className="pt-4">
          <div className="mb-3 flex w-fit flex-wrap gap-1 rounded-lg border bg-muted/40 p-1">
            {ACCOUNT_TYPE_TABS.map((t) => (
              <button
                key={t.key || 'all'}
                type="button"
                className={cn(
                  'rounded-md px-3 py-1.5 text-sm transition-colors',
                  accountType === t.key
                    ? 'bg-background font-medium shadow-sm'
                    : 'text-muted-foreground hover:text-foreground',
                )}
                onClick={() => pickAccountType(t.key)}
              >
                {t.label}
              </button>
            ))}
          </div>
          {error && <p className="mb-3 text-sm text-destructive">{error}</p>}
          {txns.error && <p className="text-sm text-destructive">加载失败：{txns.error}</p>}
          <Table className="table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[44px]"><span className="sr-only">类型</span></TableHead>
                <TableHead className="w-[88px]">日期</TableHead>
                <TableHead>收款方 / 描述 / 标签</TableHead>
                <TableHead className="w-[104px] text-right">金额</TableHead>
                <TableHead className="w-[108px] text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {txns.loading && !txns.data
                ? Array.from({ length: 8 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell colSpan={5}>
                        <Skeleton className="h-6 w-full" />
                      </TableCell>
                    </TableRow>
                  ))
                : txns.data?.items.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="align-top pt-2.5">
                    <TransactionIcon t={t} />
                  </TableCell>
                  <TableCell className="align-top whitespace-nowrap">
                    {t.date}
                  </TableCell>
                  <TableCell className="align-top">
                    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 break-words">
                      {t.payee && <span className="font-semibold">{t.payee}</span>}
                      {t.narration && (
                        <span className="text-muted-foreground">{t.narration}</span>
                      )}
                      {t.tags && t.tags.length > 0 && (
                        <span className="flex flex-wrap gap-1">
                          {t.tags.map((tagName) => (
                            <Badge key={tagName} variant="secondary" className="h-4 px-1.5 text-[10px]">
                              #{tagName}
                            </Badge>
                          ))}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="align-top whitespace-nowrap text-right font-mono">
                    {amountSummary(t) || '-'}
                  </TableCell>
                  <TableCell className="align-top whitespace-nowrap text-right">
                    <div className="flex justify-end gap-1">
                      <button
                        type="button"
                        title="查看"
                        className={cn(buttonVariants({ variant: 'outline', size: 'icon-xs' }))}
                        onClick={() => setViewing(t)}
                      >
                        <Eye />
                        <span className="sr-only">查看</span>
                      </button>
                      <button
                        type="button"
                        title="编辑"
                        className={cn(buttonVariants({ variant: 'outline', size: 'icon-xs' }))}
                        onClick={() => setEditing(t)}
                      >
                        <Pencil />
                        <span className="sr-only">编辑</span>
                      </button>
                      <button
                        type="button"
                        title="删除"
                        className={cn(buttonVariants({ variant: 'destructive', size: 'icon-xs' }))}
                        onClick={() => setDeleting(t)}
                      >
                        <Trash2 />
                        <span className="sr-only">删除</span>
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {txns.data && txns.data.items.length === 0 && !txns.loading && (
            <p className="py-8 text-center text-sm text-muted-foreground">没有符合条件的交易</p>
          )}
        </CardContent>
      </Card>

      <Dialog open={deleting != null} onOpenChange={(open) => !open && setDeleting(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>删除交易</DialogTitle>
            <DialogDescription>
              确定删除 {deleting?.date} 的「{deleting?.narration ?? deleting?.payee ?? '交易'}」吗？此操作会写入账本并增加修订号。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setDeleting(null)}
            >
              取消
            </button>
            <button
              type="button"
              className={buttonVariants({ variant: 'destructive' })}
              disabled={busy}
              onClick={confirmDelete}
            >
              {busy ? '删除中…' : '确认删除'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <TransactionViewDialog
        open={viewing != null}
        onOpenChange={(open) => !open && setViewing(null)}
        ledgerId={ledgerId}
        transaction={viewing}
        onEdit={(t) => {
          setViewing(null)
          setEditing(t)
        }}
        onDeleted={() => {
          setViewing(null)
          setReloadKey((k) => k + 1)
        }}
      />
      <TransactionEditDialog
        open={editing != null}
        onOpenChange={(open) => !open && setEditing(null)}
        ledgerId={ledgerId}
        transactionId={editing?.id}
        onSaved={() => {
          setEditing(null)
          setReloadKey((k) => k + 1)
        }}
      />

      <TemplatesDialog
        open={templatesOpen}
        onOpenChange={setTemplatesOpen}
        ledgerId={ledgerId}
        onUse={(template) => {
          setCreateInitial(template)
          setCreateOpen(true)
        }}
      />

      <TransactionEditDialog
        open={createOpen}
        onOpenChange={(open) => !open && setCreateOpen(false)}
        ledgerId={ledgerId}
        initial={createInitial}
        onSaved={() => {
          setCreateOpen(false)
          setCreateInitial(null)
          setReloadKey((k) => k + 1)
        }}
      />

      <AiRecordDialog
        open={aiOpen}
        onOpenChange={setAiOpen}
        ledgerId={ledgerId}
        onCreated={() => setReloadKey((k) => k + 1)}
      />
    </div>
  )
}
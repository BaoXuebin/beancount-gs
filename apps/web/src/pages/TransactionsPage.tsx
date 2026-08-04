import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Eye, Pencil, RefreshCw, Trash2 } from 'lucide-react'
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
  SelectItem,
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
import type { Ledger, Transaction, TransactionListResponse } from '@/api/types'
import { cn } from '@/lib/utils'
import { LoadingHint } from '@/components/LoadingHint'
import { TransactionViewDialog } from '@/components/TransactionViewDialog'
import { TransactionEditDialog } from '@/components/TransactionEditDialog'

const PAGE_SIZE = 50

function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function amountSummary(t: Transaction): string {
  const withUnits = t.postings.find((p) => p.units != null)
  if (!withUnits?.units) return ''
  return `${withUnits.units.number} ${withUnits.units.currency}`
}

export function TransactionsPage() {
  const { ledgerId = '' } = useParams()
  const [q, setQ] = useState('')
  const [month, setMonth] = useState(currentMonth())
  const [account, setAccount] = useState('')
  const [tag, setTag] = useState('')
  const [filters, setFilters] = useState({
    q: '',
    month: currentMonth(),
    account: '',
    tag: '',
  })
  const [offset, setOffset] = useState(0)
  const [conflict, setConflict] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<Transaction | null>(null)
  const [viewing, setViewing] = useState<Transaction | null>(null)
  const [editing, setEditing] = useState<Transaction | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const query = useCallback(() => {
    const params = new URLSearchParams()
    params.set('limit', String(PAGE_SIZE))
    params.set('offset', String(offset))
    if (filters.q) params.set('q', filters.q)
    if (filters.month) params.set('month', filters.month)
    if (filters.account) params.set('account', filters.account)
    if (filters.tag) params.set('tag', filters.tag)
    return params.toString()
  }, [filters, offset])

  const [reloadKey, setReloadKey] = useState(0)
  const url = `/ledgers/${ledgerId}/transactions?${query()}`
  // 通过 key 变化触发 useFetch 重新请求，避免本地缓存
  const txns = useFetch<TransactionListResponse>(url, undefined, reloadKey)
  const ledger = useFetch<Ledger>(`/ledgers/${ledgerId}`)
  const months = useFetch<string[]>(`/ledgers/${ledgerId}/months`)
  const [monthsAdjusted, setMonthsAdjusted] = useState(false)

  // 默认当月；若账本没有当月交易，则回退到最近一个有交易的月份
  useEffect(() => {
    if (monthsAdjusted || !months.data || months.data.length === 0) return
    const current = currentMonth()
    if (filters.month === current && !months.data.includes(current)) {
      const fallback = months.data[0]
      setMonth(fallback)
      setFilters((f) => ({ ...f, month: fallback }))
    }
    setMonthsAdjusted(true)
  }, [months.data, monthsAdjusted, filters.month])

  const applyFilters = () => {
    setFilters({ q: q.trim(), month, account: account.trim(), tag: tag.trim() })
    setOffset(0)
  }

  const loadMore = () => {
    setOffset((o) => o + PAGE_SIZE)
  }

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
      {txns.loading && <LoadingHint className="mb-2" />}
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
          <Link to={`/ledgers/${ledgerId}/templates`} className={buttonVariants({ variant: 'outline' })}>
            模板
          </Link>
          <Link to={`/ledgers/${ledgerId}/transactions/new`} className={buttonVariants()}>
            记一笔
          </Link>
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
        <CardContent className="flex flex-wrap items-end gap-2 pt-4">
          <div className="grid gap-1">
            <Label>搜索</Label>
            <Input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="收款方 / 描述 / 金额"
              className="w-44"
              onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
            />
          </div>
          <div className="grid gap-1">
            <Label>月份</Label>
            <Select value={month} onValueChange={(value) => value != null && setMonth(value)}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部月份</SelectItem>
                {(months.data ?? []).map((m) => (
                  <SelectItem key={m} value={m}>
                    {m}
                  </SelectItem>
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
              onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
            />
          </div>
          <div className="grid gap-1">
            <Label>标签</Label>
            <Input
              value={tag}
              onChange={(e) => setTag(e.target.value)}
              placeholder="如 Food"
              className="w-32"
              onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
            />
          </div>
          <div className="flex shrink-0 gap-2">
            <button type="button" className={buttonVariants()} onClick={applyFilters}>
              应用
            </button>
            <button
              type="button"
              className={buttonVariants({ variant: 'ghost' })}
              onClick={() => {
                setQ('')
                setMonth(currentMonth())
                setAccount('')
                setTag('')
                setFilters({ q: '', month: currentMonth(), account: '', tag: '' })
                setOffset(0)
                setReloadKey((k) => k + 1)
              }}
            >
              清除
            </button>
          </div>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardContent className="pt-6">
          {error && <p className="mb-3 text-sm text-destructive">{error}</p>}
          {txns.loading && Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="my-2 h-12" />)}
          {txns.error && <p className="text-sm text-destructive">加载失败：{txns.error}</p>}
          <Table className="table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[24%]">收款方 / 日期</TableHead>
                <TableHead className="w-[26%]">描述 / 标签</TableHead>
                <TableHead className="w-[28%]">账户</TableHead>
                <TableHead className="w-[104px] text-right">金额</TableHead>
                <TableHead className="w-[108px] text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {txns.data?.items.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="align-top whitespace-normal">
                    <div className="break-words font-medium">{t.payee ?? '-'}</div>
                    <div className="mt-0.5 text-xs text-muted-foreground">{t.date}</div>
                  </TableCell>
                  <TableCell className="align-top whitespace-normal">
                    <div className="break-words">{t.narration ?? '-'}</div>
                    {t.tags && t.tags.length > 0 && (
                      <div className="mt-1.5 flex flex-wrap gap-1">
                        {t.tags.map((tagName) => (
                          <Badge key={tagName} variant="secondary" className="h-4 px-1.5 text-[10px]">
                            #{tagName}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </TableCell>
                  <TableCell className="align-top whitespace-normal">
                    <div className="space-y-0.5">
                      {t.postings.map((p, i) => (
                        <div key={i} className="break-words font-mono text-xs">
                          {p.account}
                        </div>
                      ))}
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
          {txns.data && offset + txns.data.items.length < txns.data.total && (
            <div className="mt-4">
              <button type="button" className={buttonVariants({ variant: 'outline' })} onClick={loadMore}>
                加载更多
              </button>
            </div>
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
    </div>
  )
}
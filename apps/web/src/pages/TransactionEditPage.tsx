import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { Plus, Sparkles, Trash2 } from 'lucide-react'
import { buttonVariants } from '@/components/ui/button'
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
import { ApiError, request } from '@/api/client'
import type { AiRecordResult, Amount, Posting, Transaction } from '@/api/types'
import { cn } from '@/lib/utils'

interface PostingRow {
  account: string
  number: string
  currency: string
  cost_number: string
  cost_currency: string
  price_number: string
  price_currency: string
}

function blankPosting(): PostingRow {
  return { account: '', number: '', currency: 'CNY', cost_number: '', cost_currency: '', price_number: '', price_currency: '' }
}

function today(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}

function postingToRow(p: Posting): PostingRow {
  return {
    account: p.account,
    number: p.units?.number ?? '',
    currency: p.units?.currency ?? 'CNY',
    cost_number: p.cost?.number ?? '',
    cost_currency: p.cost?.currency ?? '',
    price_number: p.price?.number ?? '',
    price_currency: p.price?.currency ?? '',
  }
}

function rowToPosting(r: PostingRow): Posting {
  const posting: Posting = { account: r.account.trim() }
  if (r.number) posting.units = { number: r.number, currency: r.currency || 'CNY' } as Amount
  if (r.cost_number && r.cost_currency) {
    posting.cost = { number: r.cost_number, currency: r.cost_currency }
  }
  if (r.price_number && r.price_currency) {
    posting.price = { number: r.price_number, currency: r.price_currency } as Amount
  }
  return posting
}

export function TransactionEditPage() {
  const { ledgerId = '', transactionId } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const isEdit = transactionId != null
  const draft = (location.state as { draft?: AiRecordResult['draft'] } | null)?.draft

  const [date, setDate] = useState(today())
  const [payee, setPayee] = useState('')
  const [narration, setNarration] = useState('')
  const [tags, setTags] = useState('')
  const [divideDates, setDivideDates] = useState('')
  const [postings, setPostings] = useState<PostingRow[]>([blankPosting(), blankPosting()])
  const [aiText, setAiText] = useState('')
  const [aiBusy, setAiBusy] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [conflict, setConflict] = useState<string | null>(null)
  const [loading, setLoading] = useState(isEdit)

  useEffect(() => {
    if (!isEdit) return
    let cancelled = false
    request<Transaction>(`/ledgers/${ledgerId}/transactions/${transactionId}`)
      .then((t) => {
        if (cancelled) return
        setDate(t.date)
        setPayee(t.payee ?? '')
        setNarration(t.narration ?? '')
        setTags((t.tags ?? []).join(', '))
        setPostings(t.postings.length ? t.postings.map(postingToRow) : [blankPosting(), blankPosting()])
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [isEdit, ledgerId, transactionId])

  useEffect(() => {
    if (!draft || isEdit) return
    if (draft.date) setDate(draft.date)
    if (draft.payee) setPayee(draft.payee)
    if (draft.narration) setNarration(draft.narration)
    if (draft.tags?.length) setTags(draft.tags.join(', '))
    if (draft.postings?.length) setPostings(draft.postings.map(postingToRow))
  }, [draft, isEdit])

  const updatePosting = (index: number, patch: Partial<PostingRow>) => {
    setPostings((prev) => prev.map((p, i) => (i === index ? { ...p, ...patch } : p)))
  }

  const balances = useMemo(() => {
    const sumByCurrency = new Map<string, number>()
    for (const p of postings) {
      if (!p.number || !p.currency) continue
      const n = Number(p.number)
      if (Number.isNaN(n)) continue
      sumByCurrency.set(p.currency, (sumByCurrency.get(p.currency) ?? 0) + n)
    }
    return [...sumByCurrency.entries()].map(([currency, sum]) => ({
      currency,
      sum,
      balanced: Math.abs(sum) < 0.001,
    }))
  }, [postings])

  const generateDraft = async () => {
    if (!aiText.trim()) return
    setAiBusy(true)
    setError(null)
    try {
      const result = await request<AiRecordResult>(`/ledgers/${ledgerId}/ai/record`, {
        method: 'POST',
        body: JSON.stringify({ text: aiText }),
      })
      const draft = result.draft
      if (draft.date) setDate(draft.date)
      if (draft.payee) setPayee(draft.payee)
      if (draft.narration) setNarration(draft.narration)
      if (draft.tags) setTags(draft.tags.join(', '))
      if (draft.postings?.length) setPostings(draft.postings.map(postingToRow))
      setAiText('')
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setAiBusy(false)
    }
  }

  const save = async () => {
    if (postings.length < 2) {
      setError('至少需要两条分录')
      return
    }
    const body = {
      date,
      payee: payee || undefined,
      narration: narration || undefined,
      tags: tags
        .split(/[,，\s]+/)
        .map((s) => s.replace(/^#/, ''))
        .filter(Boolean),
      postings: postings.map(rowToPosting).filter((p) => p.account),
      ...(divideDates.trim()
        ? { divide_dates: divideDates.split(/[,，\s]+/).filter(Boolean) }
        : {}),
    }
    setBusy(true)
    setError(null)
    setConflict(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      if (isEdit) {
        await request(`/ledgers/${ledgerId}/transactions/${transactionId}`, {
          method: 'PUT',
          body: JSON.stringify(body),
          revision: rev.revision,
        })
      } else {
        await request(`/ledgers/${ledgerId}/transactions`, {
          method: 'POST',
          body: JSON.stringify(body),
          revision: rev.revision,
        })
      }
      navigate(`/ledgers/${ledgerId}/transactions`)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setConflict(
          `账本已被他人修改（409），当前修订号 ${(err.details as { current_revision?: number })?.current_revision ?? '?'}，请刷新后重试`,
        )
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return <Skeleton className="h-96" />
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{isEdit ? '编辑交易' : '记一笔'}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            字段遵循 beancount 术语：narration / postings / units / cost
          </p>
        </div>
        <Link to={`/ledgers/${ledgerId}/transactions`} className={buttonVariants({ variant: 'outline' })}>
          返回交易
        </Link>
      </div>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="flex items-center gap-1.5 text-base">
            <Sparkles className="size-4" /> AI 生成分录（可选）
          </CardTitle>
          <CardDescription>输入自然语言，如「昨天星巴克咖啡 38 元」，AI 生成草稿后仍可修改</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap items-center gap-2">
          <Input
            value={aiText}
            onChange={(e) => setAiText(e.target.value)}
            placeholder="如：昨天星巴克咖啡 38 元"
            className="min-w-64 flex-1"
            onKeyDown={(e) => e.key === 'Enter' && generateDraft()}
          />
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            disabled={aiBusy || !aiText.trim()}
            onClick={generateDraft}
          >
            {aiBusy ? '生成中…' : '生成草稿'}
          </button>
        </CardContent>
      </Card>

      {conflict && (
        <div className="mt-4 rounded-lg border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          ⚠ {conflict}
        </div>
      )}
      {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

      <Card className="mt-4">
        <CardContent className="grid gap-4 pt-6 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <Label>日期</Label>
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </div>
          <div className="grid gap-1.5">
            <Label>收款方 Payee</Label>
            <Input value={payee} onChange={(e) => setPayee(e.target.value)} placeholder="盒马鲜生" />
          </div>
          <div className="grid gap-1.5">
            <Label>描述 Narration</Label>
            <Input value={narration} onChange={(e) => setNarration(e.target.value)} placeholder="日常采购" />
          </div>
          <div className="grid gap-1.5">
            <Label>标签 Tags</Label>
            <Input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="#Food #Family" />
          </div>
          <div className="grid gap-1.5 sm:col-span-2">
            <Label>分期日期（可选，逗号分隔）</Label>
            <Input
              value={divideDates}
              onChange={(e) => setDivideDates(e.target.value)}
              placeholder="2026-08-01, 2026-09-01, 2026-10-01"
            />
          </div>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">分录 Postings</CardTitle>
          <CardDescription>金额使用 decimal 字符串精确计算；成本 / 汇率用于外币分录</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>账户</TableHead>
                <TableHead className="w-32">金额</TableHead>
                <TableHead className="w-24">币种</TableHead>
                <TableHead className="w-28">成本</TableHead>
                <TableHead className="w-24">汇率</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {postings.map((p, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Input
                      value={p.account}
                      onChange={(e) => updatePosting(i, { account: e.target.value })}
                      placeholder="Expenses:Food"
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      value={p.number}
                      onChange={(e) => updatePosting(i, { number: e.target.value })}
                      placeholder="-120.00"
                      className="font-mono text-right"
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      value={p.currency}
                      onChange={(e) => updatePosting(i, { currency: e.target.value })}
                      placeholder="CNY"
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Input
                        value={p.cost_number}
                        onChange={(e) => updatePosting(i, { cost_number: e.target.value })}
                        placeholder="成本"
                        className="font-mono"
                      />
                      <Input
                        value={p.cost_currency}
                        onChange={(e) => updatePosting(i, { cost_currency: e.target.value })}
                        placeholder="币种"
                        className="w-20"
                      />
                    </div>
                  </TableCell>
                  <TableCell>
                    <Input
                      value={p.price_number}
                      onChange={(e) => updatePosting(i, { price_number: e.target.value })}
                      placeholder="汇率"
                      className="font-mono"
                    />
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      className="text-muted-foreground hover:text-destructive"
                      onClick={() => setPostings((prev) => prev.filter((_, j) => j !== i))}
                      title="删除分录"
                    >
                      <Trash2 className="size-4" />
                    </button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
            <button
              type="button"
              className={buttonVariants({ variant: 'outline', size: 'sm' })}
              onClick={() => setPostings((prev) => [...prev, blankPosting()])}
            >
              <Plus /> 添加分录
            </button>
            <div className="flex flex-wrap gap-2 text-sm">
              {balances.length === 0 && <span className="text-muted-foreground">填写金额后自动校验借贷平衡</span>}
              {balances.map((b) => (
                <span
                  key={b.currency}
                  className={cn(b.balanced ? 'text-emerald-600' : 'text-destructive')}
                >
                  {b.currency} 差额 {b.sum.toFixed(2)} {b.balanced ? '✓ 平衡' : '✗ 不平衡'}
                </span>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="mt-4 flex justify-end gap-2">
        <Link to={`/ledgers/${ledgerId}/transactions`} className={buttonVariants({ variant: 'outline' })}>
          取消
        </Link>
        <button type="button" className={buttonVariants()} disabled={busy} onClick={save}>
          {busy ? '保存中…' : '保存'}
        </button>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { Plus, Sparkles, Trash2 } from 'lucide-react'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ApiError, request } from '@/api/client'
import type { AiRecordBatchResult, Transaction } from '@/api/types'

interface DraftPosting {
  account: string
  number: string
  currency: string
}

interface DraftRow {
  date: string
  payee: string
  narration: string
  tags: string
  postings: DraftPosting[]
}

function blankPosting(): DraftPosting {
  return { account: '', number: '', currency: 'CNY' }
}

function toDraftRow(t: Transaction): DraftRow {
  return {
    date: t.date,
    payee: t.payee ?? '',
    narration: t.narration ?? '',
    tags: (t.tags ?? []).join(', '),
    postings: t.postings.map((p) => ({
      account: p.account,
      number: p.units?.number ?? '',
      currency: p.units?.currency ?? 'CNY',
    })),
  }
}

function parseTags(s: string): string[] {
  return s
    .split(/[,，\s]+/)
    .map((x) => x.replace(/^#/, ''))
    .filter(Boolean)
}

interface AiRecordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ledgerId: string
  onCreated?: () => void
}

export function AiRecordDialog({ open, onOpenChange, ledgerId, onCreated }: AiRecordDialogProps) {
  const [text, setText] = useState('')
  const [notes, setNotes] = useState<string | null>(null)
  const [drafts, setDrafts] = useState<DraftRow[]>([])
  const [busy, setBusy] = useState(false)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<{ created: number; failed: string[] } | null>(null)

  const generate = async () => {
    if (!text.trim() || busy) return
    setBusy(true)
    setError(null)
    setResult(null)
    setDrafts([])
    setNotes(null)
    try {
      const res = await request<AiRecordBatchResult>(`/ledgers/${ledgerId}/ai/record/batch`, {
        method: 'POST',
        body: JSON.stringify({ text }),
      })
      setDrafts((res.drafts ?? []).map(toDraftRow))
      setNotes(res.notes ?? null)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'AI_NOT_CONFIGURED') {
        setError('AI 未配置：请先在「设置 → AI」中配置后再使用 AI 记录')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setBusy(false)
    }
  }

  const updateDraft = (index: number, patch: Partial<DraftRow>) => {
    setDrafts((prev) => prev.map((d, i) => (i === index ? { ...d, ...patch } : d)))
  }

  const updatePosting = (di: number, pi: number, patch: Partial<DraftPosting>) => {
    setDrafts((prev) =>
      prev.map((d, i) =>
        i === di
          ? { ...d, postings: d.postings.map((p, j) => (j === pi ? { ...p, ...patch } : p)) }
          : d,
      ),
    )
  }

  const createAll = async () => {
    if (drafts.length === 0 || creating) return
    setCreating(true)
    setError(null)
    setResult(null)
    let created = 0
    const failed: string[] = []
    for (const d of drafts) {
      const postings = d.postings
        .filter((p) => p.account.trim())
        .map((p) => ({
          account: p.account.trim(),
          ...(p.number ? { units: { number: p.number, currency: p.currency || 'CNY' } } : {}),
        }))
      try {
        const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
        await request(`/ledgers/${ledgerId}/transactions`, {
          method: 'POST',
          body: JSON.stringify({
            date: d.date,
            payee: d.payee || undefined,
            narration: d.narration || undefined,
            tags: parseTags(d.tags),
            postings,
          }),
          revision: rev.revision,
        })
        created += 1
      } catch (err) {
        failed.push(`${d.date} ${d.narration || d.payee || '交易'}：${err instanceof Error ? err.message : String(err)}`)
      }
    }
    setResult({ created, failed })
    if (created > 0) onCreated?.()
    // 创建完成后清空草稿，避免重复点击造成重复创建
    setDrafts([])
    setCreating(false)
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onOpenChange(false)}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>AI 记录</DialogTitle>
          <DialogDescription>
            一次描述多笔交易（每行一笔），AI 生成草稿后逐条确认；支持一次创建多笔
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-1.5">
          <Label>自然语言描述</Label>
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            rows={3}
            placeholder={'例如：\n昨天星巴克咖啡 38 元\n前天打车去机场 120 元\n收到 8 月工资 18200 元'}
            className="w-full resize-y rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            className={buttonVariants()}
            disabled={busy || !text.trim()}
            onClick={generate}
          >
            <Sparkles /> {busy ? '生成中…' : 'AI 生成草稿'}
          </button>
          {drafts.length > 0 && <span className="text-sm text-muted-foreground">共 {drafts.length} 笔待确认</span>}
        </div>

        {error && <p className="text-sm text-destructive">{error}</p>}
        {notes && <p className="text-sm text-muted-foreground">AI 提示：{notes}</p>}

        {drafts.length > 0 && (
          <div className="grid gap-3">
            {drafts.map((d, di) => (
              <Card key={di} className="border-primary/30">
                <CardHeader className="flex flex-row items-start justify-between gap-2">
                  <CardTitle className="text-sm">草稿 {di + 1}</CardTitle>
                  <button
                    type="button"
                    className="text-muted-foreground hover:text-destructive"
                    title="删除该笔"
                    onClick={() => setDrafts((prev) => prev.filter((_, j) => j !== di))}
                  >
                    <Trash2 className="size-4" />
                  </button>
                </CardHeader>
                <CardContent className="grid gap-3">
                  <div className="grid gap-1.5 sm:grid-cols-2">
                    <div className="grid gap-1">
                      <Label>日期</Label>
                      <Input type="date" value={d.date} onChange={(e) => updateDraft(di, { date: e.target.value })} />
                    </div>
                    <div className="grid gap-1">
                      <Label>收款方</Label>
                      <Input value={d.payee} onChange={(e) => updateDraft(di, { payee: e.target.value })} placeholder="星巴克" />
                    </div>
                    <div className="grid gap-1">
                      <Label>描述</Label>
                      <Input value={d.narration} onChange={(e) => updateDraft(di, { narration: e.target.value })} placeholder="咖啡" />
                    </div>
                    <div className="grid gap-1">
                      <Label>标签</Label>
                      <Input value={d.tags} onChange={(e) => updateDraft(di, { tags: e.target.value })} placeholder="#Food" />
                    </div>
                  </div>

                  <div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>账户</TableHead>
                          <TableHead className="w-32">金额</TableHead>
                          <TableHead className="w-24">币种</TableHead>
                          <TableHead className="w-10" />
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {d.postings.map((p, pi) => (
                          <TableRow key={pi}>
                            <TableCell>
                              <Input
                                value={p.account}
                                onChange={(e) => updatePosting(di, pi, { account: e.target.value })}
                                placeholder="Expenses:Food"
                                className="font-mono"
                              />
                            </TableCell>
                            <TableCell>
                              <Input
                                value={p.number}
                                onChange={(e) => updatePosting(di, pi, { number: e.target.value })}
                                placeholder="-120.00"
                                className="font-mono text-right"
                              />
                            </TableCell>
                            <TableCell>
                              <Input
                                value={p.currency}
                                onChange={(e) => updatePosting(di, pi, { currency: e.target.value })}
                                placeholder="CNY"
                              />
                            </TableCell>
                            <TableCell>
                              <button
                                type="button"
                                className="text-muted-foreground hover:text-destructive"
                                title="删除分录"
                                onClick={() =>
                                  setDrafts((prev) =>
                                    prev.map((dd, i) =>
                                      i === di
                                        ? { ...dd, postings: dd.postings.filter((_, j) => j !== pi) }
                                        : dd,
                                    ),
                                  )
                                }
                              >
                                <Trash2 className="size-4" />
                              </button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                    <button
                      type="button"
                      className={buttonVariants({ variant: 'outline', size: 'sm' })}
                      onClick={() =>
                        setDrafts((prev) =>
                          prev.map((dd, i) => (i === di ? { ...dd, postings: [...dd.postings, blankPosting()] } : dd)),
                        )
                      }
                    >
                      <Plus /> 添加分录
                    </button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        {result && (
          <div className="rounded-lg border border-emerald-600/30 bg-emerald-600/5 px-4 py-3 text-sm">
            <p className="text-emerald-600">已创建 {result.created} 笔交易</p>
            {result.failed.length > 0 && (
              <ul className="mt-2 list-inside list-disc space-y-1 text-destructive">
                {result.failed.map((f, i) => <li key={i}>{f}</li>)}
              </ul>
            )}
          </div>
        )}

        <DialogFooter>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => onOpenChange(false)}
          >
            关闭
          </button>
          <button
            type="button"
            className={buttonVariants()}
            disabled={drafts.length === 0 || creating}
            onClick={createAll}
          >
            {creating ? '创建中…' : `创建 ${drafts.length} 笔`}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

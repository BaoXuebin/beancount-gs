import { useEffect, useRef, useState } from 'react'
import { ArrowUp, Plus, Sparkles, Square, Trash2 } from 'lucide-react'
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
import type { AiRecordBatchResult, Posting, Transaction, TransactionCreate } from '@/api/types'
import { cn } from '@/lib/utils'

interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
}

interface DraftPosting {
  account: string
  number: string
  currency: string
}

interface DraftItem {
  key: string
  checked: boolean
  date: string
  payee: string
  narration: string
  tags: string
  postings: DraftPosting[]
}

function blankPosting(): DraftPosting {
  return { account: '', number: '', currency: 'CNY' }
}

function toDraftItem(t: Transaction): Omit<DraftItem, 'key' | 'checked'> {
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

function postingSig(p: Posting | DraftPosting): string {
  const u = (p as Posting).units
  if (u) {
    return p.account + '|' + u.number + '|' + u.currency
  }
  const d = p as DraftPosting
  return d.account + '|' + d.number + '|' + d.currency
}

function signatureOf(t: Transaction | Omit<DraftItem, 'key' | 'checked'>): string {
  const postings = (t.postings ?? []).map(postingSig).join(';')
  return [t.date, t.payee ?? '', t.narration ?? '', postings].join('::')
}

// 弱匹配键：忽略金额 / 币种，AI 只改金额或币种时仍能对上同一草稿，保留勾选状态。
function weakKeyOf(t: Transaction | Omit<DraftItem, 'key' | 'checked'>): string {
  const accounts = (t.postings ?? []).map((p) => p.account).join(';')
  return [t.date, t.payee ?? '', t.narration ?? '', accounts].join('::')
}

function mergeDrafts(prev: DraftItem[], txns: Transaction[]): DraftItem[] {
  const prevBySig = new Map<string, DraftItem>()
  const prevByWeak = new Map<string, DraftItem>()
  for (const d of prev) {
    prevBySig.set(signatureOf(d), d)
    prevByWeak.set(weakKeyOf(d), d)
  }
  const out: DraftItem[] = []
  for (const t of txns) {
    const sig = signatureOf(t)
    const weak = weakKeyOf(t)
    const old = prevBySig.get(sig) ?? prevByWeak.get(weak)
    const base = toDraftItem(t)
    out.push({
      ...base,
      key: old?.key ?? (crypto.randomUUID ? crypto.randomUUID() : String(Math.random())),
      checked: old ? old.checked : true,
    })
  }
  return out
}

function toTransaction(d: DraftItem): TransactionCreate {
  return {
    date: d.date,
    flag: '*',
    payee: d.payee || undefined,
    narration: d.narration || undefined,
    tags: d.tags
      .split(/[,，\s]+/)
      .map((x) => x.replace(/^#/, ''))
      .filter(Boolean),
    postings: d.postings
      .filter((p) => p.account.trim())
      .map((p) => ({
        account: p.account.trim(),
        ...(p.number ? { units: { number: p.number, currency: p.currency || 'CNY' } } : {}),
      })),
  }
}

interface AiRecordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ledgerId: string
  onCreated?: () => void
}

export function AiRecordDialog({ open, onOpenChange, ledgerId, onCreated }: AiRecordDialogProps) {
  const [messages, setMessages] = useState<ChatMsg[]>([])
  const [drafts, setDrafts] = useState<DraftItem[]>([])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<{ created: number; failed: string[] } | null>(null)

  const chatUrl = '/ledgers/' + ledgerId + '/ai/record/chat'
  const abortRef = useRef<AbortController | null>(null)

  // 关闭弹窗时中断在途请求
  useEffect(() => {
    if (!open) abortRef.current?.abort()
  }, [open])

  const send = async () => {
    const text = input.trim()
    if (!text || busy) return
    const nextMessages = [...messages, { role: 'user' as const, content: text }]
    setMessages(nextMessages)
    setInput('')
    setBusy(true)
    setError(null)
    setResult(null)
    const controller = new AbortController()
    abortRef.current = controller
    try {
      const res = await request<AiRecordBatchResult>(chatUrl, {
        method: 'POST',
        body: JSON.stringify({
          messages: nextMessages,
          drafts: drafts.map(toTransaction),
        }),
        signal: controller.signal,
      })
      setMessages((prev) => [...prev, { role: 'assistant', content: res.notes ?? '已更新草稿' }])
      setDrafts((prev) => mergeDrafts(prev, res.drafts ?? []))
    } catch (err) {
      if (controller.signal.aborted) {
        // 用户取消：还原消息与输入
        setMessages(messages)
        setInput(text)
        return
      }
      if (err instanceof ApiError && err.code === 'AI_NOT_CONFIGURED') {
        setError('AI 未配置：请先在「设置 → AI」中配置后再使用 AI 记录')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setBusy(false)
      abortRef.current = null
    }
  }

  const updateDraft = (key: string, patch: Partial<DraftItem>) => {
    setDrafts((prev) => prev.map((d) => (d.key === key ? { ...d, ...patch } : d)))
  }

  const updatePosting = (key: string, pi: number, patch: Partial<DraftPosting>) => {
    setDrafts((prev) =>
      prev.map((d) =>
        d.key === key
          ? { ...d, postings: d.postings.map((p, j) => (j === pi ? { ...p, ...patch } : p)) }
          : d,
      ),
    )
  }

  const toggleAll = () => {
    const allChecked = drafts.length > 0 && drafts.every((d) => d.checked)
    setDrafts((prev) => prev.map((d) => ({ ...d, checked: !allChecked })))
  }

  const selectedCount = drafts.filter((d) => d.checked).length

  const createSelected = async () => {
    const selected = drafts.filter((d) => d.checked)
    if (selected.length === 0 || creating) return
    setCreating(true)
    setError(null)
    setResult(null)
    let created = 0
    const failed: string[] = []
    for (const d of selected) {
      try {
        const rev = await request<{ revision: number }>('/ledgers/' + ledgerId + '/revision')
        await request('/ledgers/' + ledgerId + '/transactions', {
          method: 'POST',
          body: JSON.stringify(toTransaction(d)),
          revision: rev.revision,
        })
        created += 1
      } catch (err) {
        failed.push(
          d.date + ' ' + (d.narration || d.payee || '交易') + '：' + (err instanceof Error ? err.message : String(err)),
        )
      }
    }
    setResult({ created, failed })
    if (created > 0) {
      onCreated?.()
      setDrafts((prev) => prev.filter((d) => !d.checked))
    }
    setCreating(false)
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onOpenChange(false)}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>AI 记录</DialogTitle>
          <DialogDescription>
            对话生成交易草稿：左侧聊天，右侧预览；勾选要创建的草稿，可多轮对话继续调整
          </DialogDescription>
        </DialogHeader>

        <div className="flex min-h-[55vh] flex-col gap-4 sm:flex-row">
          {/* 左：对话列表 */}
          <div className="flex w-full shrink-0 flex-col rounded-lg border sm:w-72">
            <div className="max-h-72 flex-1 space-y-3 overflow-y-auto p-3 sm:max-h-[50vh]">
              {messages.length === 0 && (
                <p className="text-xs leading-5 text-muted-foreground">
                  输入自然语言描述，可一次描述多笔（每行一笔）。
                  之后可以继续对话调整：如「把咖啡改成 50 元」「再加一笔打车 120 元」。
                </p>
              )}
              {messages.map((m, i) => (
                <div
                  key={i}
                  className={cn(
                    'max-w-[95%] whitespace-pre-wrap rounded-lg px-3 py-2 text-sm',
                    m.role === 'user' ? 'ml-auto bg-primary/10' : 'bg-muted',
                  )}
                >
                  {m.content}
                </div>
              ))}
              {busy && <div className="text-xs text-muted-foreground">AI 思考中…</div>}
            </div>
            <div className="border-t p-2">
              {/* 聊天式输入框：textarea 在上，底部工具行右侧为圆形发送按钮 */}
              <div className="rounded-lg border border-input bg-transparent transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 dark:bg-input/30">
                <textarea
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                      e.preventDefault()
                      send()
                    }
                  }}
                  rows={2}
                  placeholder="输入消息，Enter 发送，Shift+Enter 换行"
                  className="w-full resize-none bg-transparent px-2.5 pt-2 pb-1 text-sm outline-none placeholder:text-muted-foreground"
                />
                <div className="flex items-center justify-end px-1.5 pb-1.5">
                  {busy ? (
                    <button
                      type="button"
                      className={cn(buttonVariants({ variant: 'destructive', size: 'icon-sm' }), 'rounded-full')}
                      onClick={() => abortRef.current?.abort()}
                      title="取消"
                    >
                      <Square className="size-3" fill="currentColor" />
                    </button>
                  ) : (
                    <button
                      type="button"
                      className={cn(buttonVariants({ size: 'icon-sm' }), 'rounded-full')}
                      disabled={!input.trim()}
                      onClick={send}
                      title="发送"
                    >
                      <ArrowUp className="size-4" />
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* 右：交易草稿预览 */}
          <div className="min-w-0 flex-1">
            <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Sparkles className="size-4 text-primary" />
                <span className="text-sm font-medium">交易草稿预览</span>
                {drafts.length > 0 && (
                  <span className="text-xs text-muted-foreground">
                    已勾选 {selectedCount} / {drafts.length}
                  </span>
                )}
              </div>
              {drafts.length > 0 && (
                <div className="flex gap-2">
                  <button
                    type="button"
                    className={buttonVariants({ variant: 'outline', size: 'sm' })}
                    onClick={toggleAll}
                  >
                    全选 / 全不选
                  </button>
                  <button
                    type="button"
                    className={buttonVariants({ size: 'sm' })}
                    disabled={selectedCount === 0 || creating}
                    onClick={createSelected}
                  >
                    {creating ? '创建中…' : '创建选中的 ' + selectedCount + ' 笔'}
                  </button>
                </div>
              )}
            </div>

            {error && <p className="mb-2 text-sm text-destructive">{error}</p>}
            {result && (
              <div className="mb-2 rounded-lg border border-emerald-600/30 bg-emerald-600/5 px-4 py-3 text-sm">
                <p className="text-emerald-600">已创建 {result.created} 笔交易</p>
                {result.failed.length > 0 && (
                  <ul className="mt-2 list-inside list-disc space-y-1 text-destructive">
                    {result.failed.map((f, i) => (
                      <li key={i}>{f}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {drafts.length === 0 ? (
              <div className="rounded-lg border border-dashed p-10 text-center text-sm text-muted-foreground">
                {messages.length > 0 ? '当前对话暂无草稿' : '在左侧描述交易，AI 生成的草稿会出现在这里'}
              </div>
            ) : (
              <div className="max-h-[50vh] space-y-3 overflow-y-auto pr-1">
                {drafts.map((d, di) => (
                  <Card key={d.key} className="border-primary/30">
                    <CardHeader className="flex flex-row items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          className="size-4 accent-primary"
                          checked={d.checked}
                          onChange={(e) => updateDraft(d.key, { checked: e.target.checked })}
                        />
                        <CardTitle className="text-sm">草稿 {di + 1}</CardTitle>
                      </div>
                      <button
                        type="button"
                        className="text-muted-foreground hover:text-destructive"
                        title="删除该笔"
                        onClick={() => setDrafts((prev) => prev.filter((x) => x.key !== d.key))}
                      >
                        <Trash2 className="size-4" />
                      </button>
                    </CardHeader>
                    <CardContent className="grid gap-3">
                      <div className="grid gap-1.5 sm:grid-cols-2">
                        <div className="grid gap-1">
                          <Label>日期</Label>
                          <Input
                            type="date"
                            value={d.date}
                            onChange={(e) => updateDraft(d.key, { date: e.target.value })}
                          />
                        </div>
                        <div className="grid gap-1">
                          <Label>收款方</Label>
                          <Input
                            value={d.payee}
                            onChange={(e) => updateDraft(d.key, { payee: e.target.value })}
                            placeholder="星巴克"
                          />
                        </div>
                        <div className="grid gap-1">
                          <Label>描述</Label>
                          <Input
                            value={d.narration}
                            onChange={(e) => updateDraft(d.key, { narration: e.target.value })}
                            placeholder="咖啡"
                          />
                        </div>
                        <div className="grid gap-1">
                          <Label>标签</Label>
                          <Input
                            value={d.tags}
                            onChange={(e) => updateDraft(d.key, { tags: e.target.value })}
                            placeholder="#Food"
                          />
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
                                    onChange={(e) => updatePosting(d.key, pi, { account: e.target.value })}
                                    placeholder="Expenses:Food"
                                    className="font-mono"
                                  />
                                </TableCell>
                                <TableCell>
                                  <Input
                                    value={p.number}
                                    onChange={(e) => updatePosting(d.key, pi, { number: e.target.value })}
                                    placeholder="-120.00"
                                    className="font-mono text-right"
                                  />
                                </TableCell>
                                <TableCell>
                                  <Input
                                    value={p.currency}
                                    onChange={(e) => updatePosting(d.key, pi, { currency: e.target.value })}
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
                                        prev.map((dd) =>
                                          dd.key === d.key
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
                              prev.map((dd) =>
                                dd.key === d.key ? { ...dd, postings: [...dd.postings, blankPosting()] } : dd,
                              ),
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
          </div>
        </div>

        <DialogFooter>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => onOpenChange(false)}
          >
            关闭
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

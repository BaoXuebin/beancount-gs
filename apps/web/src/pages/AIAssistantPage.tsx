import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Sparkles } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
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
import type { AiRecordResult, InsightsResponse } from '@/api/types'

type Mode = 'record' | 'summarize' | 'insights'

interface ChatMessage {
  role: 'user' | 'assistant'
  text: string
  draft?: AiRecordResult
  summary?: string
}

const examples: Record<Mode, string[]> = {
  record: ['昨天星巴克咖啡 38 元', '周五打车去机场 120 元', '收到 8 月工资 18200 元'],
  summarize: ['2026-08', '2026-07'],
  insights: [],
}

export function AIAssistantPage() {
  const { ledgerId = '' } = useParams()
  const navigate = useNavigate()
  const [mode, setMode] = useState<Mode>('record')
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [confirming, setConfirming] = useState(false)

  const push = (msg: ChatMessage) => setMessages((prev) => [...prev, msg])

  const send = async (text: string) => {
    if (!text.trim() || busy) return
    setInput('')
    setError(null)
    push({ role: 'user', text })
    setBusy(true)
    try {
      if (mode === 'record') {
        const result = await request<AiRecordResult>(`/ledgers/${ledgerId}/ai/record`, {
          method: 'POST',
          body: JSON.stringify({ text }),
        })
        push({ role: 'assistant', text: result.notes ?? '已生成待确认交易草稿：', draft: result })
      } else if (mode === 'summarize') {
        const month = text.trim()
        const data = await request<{ summary: string }>(
          `/ledgers/${ledgerId}/ai/summarize${month ? `?month=${encodeURIComponent(month)}` : ''}`,
          { method: 'POST' },
        )
        push({ role: 'assistant', text: `总结（${month || '全部'}）：`, summary: data.summary })
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const loadInsights = async () => {
    if (busy) return
    setError(null)
    setBusy(true)
    try {
      const data = await request<InsightsResponse>(`/ledgers/${ledgerId}/ai/insights`)
      const text = data.insights.length
        ? data.insights.map((i) => `⚠ ${i.message}`).join('\n')
        : '暂未发现异常，账本状态良好。'
      push({ role: 'assistant', text })
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const confirmDraft = async (draft: AiRecordResult['draft']) => {
    setConfirming(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/transactions`, {
        method: 'POST',
        body: JSON.stringify(draft),
        revision: rev.revision,
      })
      navigate(`/ledgers/${ledgerId}/transactions`)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setError('账本已被他人修改（409），请重新生成草稿')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setConfirming(false)
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold">
            <Sparkles className="size-5" /> AI 记账助手
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            自然语言记账 / 总结 / 洞察 · AI 只生成待确认内容，写入账本需你确认
          </p>
        </div>
        <Link to={`/ledgers/${ledgerId}/settings/ai`} className={buttonVariants({ variant: 'outline' })}>
          AI 设置
        </Link>
      </div>

      <div className="mt-4 flex flex-wrap gap-1 rounded-lg border bg-muted/40 p-1">
        {(
          [
            { key: 'record', label: '记一笔' },
            { key: 'summarize', label: '总结' },
            { key: 'insights', label: '洞察' },
          ] as { key: Mode; label: string }[]
        ).map((m) => (
          <button
            key={m.key}
            type="button"
            className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
              mode === m.key
                ? 'bg-background font-medium shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => {
              setMode(m.key)
              setError(null)
            }}
          >
            {m.label}
          </button>
        ))}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {examples[mode].map((ex) => (
          <button
            key={ex}
            type="button"
            className="rounded-full border px-3 py-1 text-xs text-muted-foreground hover:border-primary hover:text-foreground"
            onClick={() => {
              setInput(ex)
              if (mode === 'insights') loadInsights()
            }}
          >
            {ex}
          </button>
        ))}
        {mode === 'insights' && (
          <button
            type="button"
            className="rounded-full border px-3 py-1 text-xs text-muted-foreground hover:border-primary hover:text-foreground"
            onClick={loadInsights}
          >
            帮我找重复扣款 / 异常
          </button>
        )}
      </div>

      {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

      <div className="mt-4 flex flex-col gap-3">
        {messages.map((msg, i) => {
          const draft = msg.draft
          return (
            <div key={i} className={msg.role === 'user' ? 'self-end' : 'w-full'}>
              <Card className={msg.role === 'user' ? 'border-primary/40' : ''}>
                <CardHeader>
                  <CardTitle className="text-sm">{msg.role === 'user' ? '用户' : 'AI'}</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-3">
                  <p className="whitespace-pre-line text-sm">{msg.text}</p>
                  {msg.summary && <p className="whitespace-pre-line text-sm">{msg.summary}</p>}
                  {draft && (
                    <>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>日期</TableHead>
                            <TableHead>收款方</TableHead>
                            <TableHead>描述</TableHead>
                            <TableHead className="text-right">金额</TableHead>
                            <TableHead>账户</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {draft.draft.postings.map((p, j) => (
                            <TableRow key={j}>
                              {j === 0 && (
                                <TableCell rowSpan={draft.draft.postings.length}>
                                  {draft.draft.date}
                                </TableCell>
                              )}
                              {j === 0 && (
                                <TableCell rowSpan={draft.draft.postings.length}>
                                  {draft.draft.payee ?? '—'}
                                </TableCell>
                              )}
                              {j === 0 && (
                                <TableCell rowSpan={draft.draft.postings.length}>
                                  {draft.draft.narration ?? '—'}
                                </TableCell>
                              )}
                              <TableCell className="text-right font-mono">
                                {p.units?.number ?? '—'} {p.units?.currency ?? ''}
                              </TableCell>
                              <TableCell className="max-w-[220px] truncate font-mono">
                                {p.account}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                      {draft.draft.account_suggestions?.length ? (
                        <div className="flex flex-wrap gap-1">
                          {draft.draft.account_suggestions.map((s, j) => (
                            <Badge key={j} variant="outline" className="font-mono">
                              {s.account}（AI {Math.round((s.confidence ?? 0) * 100)}%）
                            </Badge>
                          ))}
                        </div>
                      ) : null}
                      <div className="flex gap-2">
                        <Link
                          to={`/ledgers/${ledgerId}/transactions/new`}
                          state={{ draft: draft.draft }}
                          className={buttonVariants({ variant: 'outline', size: 'sm' })}
                        >
                          修改
                        </Link>
                        <button
                          type="button"
                          className={buttonVariants({ size: 'sm' })}
                          disabled={confirming}
                          onClick={() => confirmDraft(draft.draft)}
                        >
                          {confirming ? '入账中…' : '确认入账'}
                        </button>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
          )
        })}
        {busy && <Skeleton className="h-24" />}
        {messages.length === 0 && !busy && (
          <p className="py-10 text-center text-sm text-muted-foreground">
            {mode === 'record' && '输入自然语言，例如「昨天星巴克咖啡 38 元」'}
            {mode === 'summarize' && '输入月份，例如 2026-08，生成账本总结'}
            {mode === 'insights' && '点击上方示例按钮，检查重复扣款 / 大额支出等异常'}
          </p>
        )}
      </div>

      <div className="mt-4 flex gap-2">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder={
            mode === 'record'
              ? '输入自然语言记账…'
              : mode === 'summarize'
                ? '输入月份，如 2026-08…'
                : '洞察为自动分析，无需输入'
          }
          disabled={mode === 'insights' || busy}
          onKeyDown={(e) => e.key === 'Enter' && send(input)}
        />
        <button
          type="button"
          className={buttonVariants()}
          disabled={busy || !input.trim() || mode === 'insights'}
          onClick={() => send(input)}
        >
          {busy ? '处理中…' : '发送'}
        </button>
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        写操作需要 editor 及以上角色并携带修订号；所有 AI 调用记录进入审计日志
      </p>
    </div>
  )
}

import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Archive, Plus, Sparkles, Trash2 } from 'lucide-react'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { ApiError, request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type {
  Account,
  AccountOpenBatchResult,
  AiAccountsResult,
} from '@/api/types'
import { AccountTree } from '@/components/AccountTree'
import { ACCOUNT_TYPE_META, formatNumber } from '@/lib/accountMeta'
import type { AccountType } from '@/lib/accountMeta'
import { AccountDetailDialog } from '@/components/AccountDetailDialog'
import { CurrenciesDialog } from '@/components/CurrenciesDialog'
import { LoadingHint } from '@/components/LoadingHint'
import { cn } from '@/lib/utils'

const typeTabs = [
  { key: 'Assets', label: '资产' },
  { key: 'Liabilities', label: '负债' },
  { key: 'Income', label: '收入' },
  { key: 'Expenses', label: '费用' },
  { key: 'Equity', label: '权益' },
]

interface SuggestionRow {
  account: string
  currency: string
}

function parseAccountText(text: string): SuggestionRow[] {
  return text
    .split(/[\n,，、;；]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .map((account) => ({ account, currency: '' }))
}

const TAB_KEY = 'bgs:accounts:tab'

export function AccountsPage() {
  const { ledgerId = '' } = useParams()
  const [showClosed, setShowClosed] = useState(false)
  const accounts = useFetch<Account[]>(`/ledgers/${ledgerId}/accounts?status=${showClosed ? 'closed' : 'open'}`)
  const totals = useFetch<Record<string, string>>(`/ledgers/${ledgerId}/stats/total`)
  const [tab, setTab] = useState<string>(() => {
    const saved = localStorage.getItem(TAB_KEY)
    return typeTabs.some((t) => t.key === saved) ? (saved as string) : 'Assets'
  })
  const [viewing, setViewing] = useState<Account | null>(null)
  const [currenciesOpen, setCurrenciesOpen] = useState(false)

  useEffect(() => {
    localStorage.setItem(TAB_KEY, tab)
  }, [tab])

  // 单个开户
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState({
    account: '',
    opened_on: new Date().toISOString().slice(0, 10),
    currency: 'CNY',
    booking: 'none',
  })

  // AI 批量开户
  const [aiOpen, setAiOpen] = useState(false)
  const [aiText, setAiText] = useState('')
  const [aiBusy, setAiBusy] = useState(false)
  const [aiError, setAiError] = useState<string | null>(null)
  const [aiNotes, setAiNotes] = useState<string | null>(null)
  const [suggestions, setSuggestions] = useState<SuggestionRow[]>([])
  const [createBusy, setCreateBusy] = useState(false)
  const [result, setResult] = useState<AccountOpenBatchResult | null>(null)

  const allAccounts = accounts.data ?? []
  const byType = allAccounts.filter((a) => a.type === tab)
  const counts = typeTabs.reduce<Record<string, number>>((acc, t) => {
    acc[t.key] = allAccounts.filter((a) => a.type === t.key && a.status === 'open').length
    return acc
  }, {})

  const createAccount = async () => {
    if (!form.account.trim()) {
      setError('请填写账户名称')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request<Account>(`/ledgers/${ledgerId}/accounts`, {
        method: 'POST',
        body: JSON.stringify({
          account: form.account.trim(),
          opened_on: form.opened_on,
          currency: form.currency,
          booking: form.booking === 'none' ? undefined : form.booking,
        }),
        revision: rev.revision,
      })
      setOpen(false)
      setForm({
        account: '',
        opened_on: new Date().toISOString().slice(0, 10),
        currency: 'CNY',
        booking: 'none',
      })
      accounts.refetch()
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setError('账本已被他人修改（409），请刷新后重试')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setBusy(false)
    }
  }

  const generateAccounts = async () => {
    if (!aiText.trim()) return
    setAiBusy(true)
    setAiError(null)
    setResult(null)
    setAiNotes(null)
    try {
      const res = await request<AiAccountsResult>(`/ledgers/${ledgerId}/ai/accounts`, {
        method: 'POST',
        body: JSON.stringify({ text: aiText }),
      })
      setSuggestions((res.accounts ?? []).map((a) => ({ account: a.account, currency: a.currency ?? '' })))
      setAiNotes(res.notes ?? null)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'AI_NOT_CONFIGURED') {
        setAiError('AI 未配置；已改为从文本解析，每行一个账户名（如 Assets:Bank:招商银行）')
        setSuggestions(parseAccountText(aiText))
      } else {
        setAiError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setAiBusy(false)
    }
  }

  const parseLines = () => {
    setAiError(null)
    setResult(null)
    setAiNotes(null)
    setSuggestions(parseAccountText(aiText))
  }

  const updateSuggestion = (index: number, patch: Partial<SuggestionRow>) => {
    setSuggestions((prev) => prev.map((s, i) => (i === index ? { ...s, ...patch } : s)))
  }

  const createBatch = async () => {
    const rows = suggestions
      .map((s) => ({ account: s.account.trim(), currency: s.currency.trim() }))
      .filter((s) => s.account)
    if (rows.length === 0) {
      setAiError('请至少填写一个账户')
      return
    }
    setCreateBusy(true)
    setAiError(null)
    setResult(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      const res = await request<AccountOpenBatchResult>(`/ledgers/${ledgerId}/accounts/batch`, {
        method: 'POST',
        body: JSON.stringify({
          accounts: rows.map((r) => ({
            account: r.account,
            opened_on: new Date().toISOString().slice(0, 10),
            ...(r.currency ? { currency: r.currency } : {}),
          })),
        }),
        revision: rev.revision,
      })
      setResult(res)
      setSuggestions([])
      setAiText('')
      accounts.refetch()
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setAiError('账本已被他人修改（409），请刷新后重试')
      } else {
        setAiError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setCreateBusy(false)
    }
  }

  const validSuggestionCount = suggestions.filter((s) => s.account.trim()).length

  const startAddChild = (parentPath: string) => {
    setError(null)
    setForm((f) => ({ ...f, account: parentPath ? `${parentPath}:` : '' }))
    setOpen(true)
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">账户</h1>
          <p className="mt-1 text-sm text-muted-foreground">按类型展示，层级结构体现账户归属关系</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => setCurrenciesOpen(true)}
          >
            币种与汇率
          </button>
          <button
            type="button"
            title={showClosed ? '隐藏已关闭账户' : '显示已关闭账户'}
            className={cn(buttonVariants({ variant: showClosed ? 'default' : 'outline' }))}
            onClick={() => setShowClosed((v) => !v)}
          >
            <Archive /> 已关闭
          </button>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => {
              setAiError(null)
              setResult(null)
              setAiText('')
              setSuggestions([])
              setAiOpen(true)
            }}
          >
            <Sparkles /> AI 批量开户
          </button>
          <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
            <Plus /> 开户
          </button>
        </div>
      </div>

      {accounts.loading && accounts.data == null && (
        <div className="mt-3">
          <LoadingHint />
        </div>
      )}

      <div className="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-5">
        {typeTabs.map((t) => {
          const meta = ACCOUNT_TYPE_META[t.key as AccountType]
          const raw = totals.data?.[t.key]
          const value = raw != null && raw !== '' ? Number(raw) : null
          return (
            <div key={t.key} className="rounded-lg border bg-background px-3 py-2">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <meta.icon className={"size-3.5 " + meta.iconClass} />
                总{meta.label}
              </div>
              <div className={"mt-0.5 truncate font-mono text-sm font-semibold tabular-nums " + meta.chipClass}>
                {totals.loading && value == null ? '…' : value != null ? formatNumber(value) : '—'}
              </div>
            </div>
          )
        })}
      </div>

      <div className="mt-4 flex flex-wrap gap-1 rounded-lg border bg-muted/40 p-1">
        {typeTabs.map((t) => (
          <button
            key={t.key}
            type="button"
            className={cn(
              'flex-1 rounded-md px-3 py-1.5 text-sm transition-colors sm:flex-none sm:px-4',
              tab === t.key
                ? 'bg-background font-medium shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
            onClick={() => setTab(t.key)}
          >
            {t.label}
            <span className="ml-1 text-xs text-muted-foreground">
              {counts[t.key] ?? 0}
            </span>
          </button>
        ))}
      </div>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">
            {typeTabs.find((t) => t.key === tab)?.label}账户
          </CardTitle>
          <CardDescription>
            按「{tab}」前缀的层级关系展开，支持搜索与快捷操作
          </CardDescription>
        </CardHeader>
        <CardContent>
          {accounts.loading && accounts.data == null ? (
            <div className="flex flex-col gap-2">
              {Array.from({ length: 8 }).map((_, i) => (
                <Skeleton key={i} className="h-7 rounded-md" style={{ marginLeft: (i % 4) * 18 }} />
              ))}
            </div>
          ) : accounts.error ? (
            <p className="text-sm text-destructive">加载失败：{accounts.error}</p>
          ) : (
            <AccountTree
              accounts={byType}
              onSelect={(acc) => setViewing(acc)}
              onAddChild={startAddChild}
            />
          )}
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={(o) => !o && setOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>开户</DialogTitle>
            <DialogDescription>新账户写入 account/*.bean 的 open 指令</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>账户名称</Label>
              <Input
                value={form.account}
                onChange={(e) => setForm((f) => ({ ...f, account: e.target.value }))}
                placeholder="Assets:Bank:招商银行"
              />
              <p className="text-xs text-muted-foreground">
                使用冒号层级，如 Assets:Bank:招商银行，将自动归入对应类型树
              </p>
            </div>
            <div className="grid gap-1.5">
              <Label>开户日期</Label>
              <Input
                type="date"
                value={form.opened_on}
                onChange={(e) => setForm((f) => ({ ...f, opened_on: e.target.value }))}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>币种</Label>
              <Input
                value={form.currency}
                onChange={(e) => setForm((f) => ({ ...f, currency: e.target.value }))}
                placeholder="CNY（多币种用逗号分隔）"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>成本计价</Label>
              <Select
                value={form.booking}
                onValueChange={(value) => value && setForm((f) => ({ ...f, booking: value }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">无（默认）</SelectItem>
                  <SelectItem value="fifo">FIFO（外币自动启用）</SelectItem>
                  <SelectItem value="average">平均成本</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setOpen(false)}
            >
              取消
            </button>
            <button
              type="button"
              className={buttonVariants()}
              disabled={busy}
              onClick={createAccount}
            >
              {busy ? '创建中…' : '创建账户'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={aiOpen} onOpenChange={(o) => !o && setAiOpen(false)}>
        <DialogContent className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>AI 批量开户</DialogTitle>
            <DialogDescription>
              用自然语言描述账户，AI 生成 beancount 账户列表；确认后一次批量写入 open 指令
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-1.5">
            <Label>账户描述</Label>
            <textarea
              value={aiText}
              onChange={(e) => setAiText(e.target.value)}
              rows={3}
              placeholder={'例如：我需要这些账户：招商银行储蓄卡、美团外卖、水电费、工资收入\n也可以直接输入账户名，每行一个'}
              className="w-full resize-y rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
            />
          </div>

          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className={buttonVariants()}
              disabled={aiBusy || !aiText.trim()}
              onClick={generateAccounts}
            >
              <Sparkles /> {aiBusy ? '生成中…' : 'AI 生成账户'}
            </button>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              disabled={!aiText.trim()}
              onClick={parseLines}
            >
              从文本解析
            </button>
          </div>

          {aiError && <p className="text-sm text-destructive">{aiError}</p>}
          {aiNotes && <p className="text-sm text-muted-foreground">AI 提示：{aiNotes}</p>}

          {suggestions.length > 0 && (
            <div className="grid gap-2">
              <Label>待创建账户（可编辑）</Label>
              <div className="grid grid-cols-[1fr_auto_auto] items-center gap-2">
                {suggestions.map((s, i) => (
                  <div key={i} className="col-span-3 grid grid-cols-[1fr_6rem_auto] items-center gap-2">
                    <Input
                      value={s.account}
                      onChange={(e) => updateSuggestion(i, { account: e.target.value })}
                      placeholder="Assets:Bank:招商银行"
                      className="font-mono"
                    />
                    <Input
                      value={s.currency}
                      onChange={(e) => updateSuggestion(i, { currency: e.target.value })}
                      placeholder="CNY"
                      className="font-mono"
                    />
                    <button
                      type="button"
                      className="text-muted-foreground hover:text-destructive"
                      title="移除"
                      onClick={() => setSuggestions((prev) => prev.filter((_, j) => j !== i))}
                    >
                      <Trash2 className="size-4" />
                    </button>
                  </div>
                ))}
              </div>
              <div>
                <button
                  type="button"
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                  onClick={() => setSuggestions((prev) => [...prev, { account: '', currency: '' }])}
                >
                  <Plus /> 添加一行
                </button>
              </div>
            </div>
          )}

          {result && (
            <div className="rounded-lg border border-emerald-600/30 bg-emerald-600/5 px-4 py-3 text-sm">
              <p className="text-emerald-600">
                已创建 {result.created.length} 个账户
                {result.skipped && result.skipped.length > 0
                  ? `，跳过 ${result.skipped.length} 个（${result.skipped.map((s) => s.account).join('、')}）`
                  : ''}
              </p>
            </div>
          )}

          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setAiOpen(false)}
            >
              关闭
            </button>
            <button
              type="button"
              className={buttonVariants()}
              disabled={createBusy || validSuggestionCount === 0}
              onClick={createBatch}
            >
              {createBusy ? '创建中…' : `批量创建（${validSuggestionCount}）`}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AccountDetailDialog
        open={viewing != null}
        onOpenChange={(o) => !o && setViewing(null)}
        ledgerId={ledgerId}
        account={viewing?.account ?? null}
        onChanged={() => accounts.refetch()}
      />

      <CurrenciesDialog
        open={currenciesOpen}
        onOpenChange={setCurrenciesOpen}
        ledgerId={ledgerId}
      />
    </div>
  )
}
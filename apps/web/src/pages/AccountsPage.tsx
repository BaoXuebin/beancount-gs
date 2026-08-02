import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError, request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { Account } from '@/api/types'

const typeLabels: Record<string, string> = {
  Assets: '资产',
  Liabilities: '负债',
  Income: '收入',
  Expenses: '费用',
  Equity: '权益',
}

function displayName(account: string): string {
  return account.split(':').pop() ?? account
}

function amountText(a: Account): string {
  if (a.market_number) return `${a.market_number} ${a.market_currency ?? ''}`.trim()
  if (a.positions?.length) {
    return a.positions.map((p) => `${p.number} ${p.currency}`).join(' · ')
  }
  return '0'
}

export function AccountsPage() {
  const { ledgerId = '' } = useParams()
  const accounts = useFetch<Account[]>(`/ledgers/${ledgerId}/accounts?status=open`)

  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState({
    account: '',
    opened_on: new Date().toISOString().slice(0, 10),
    currency: 'CNY',
    booking: 'none',
  })

  const groups = ['Assets', 'Liabilities', 'Income', 'Expenses', 'Equity']
    .map((type) => ({
      type,
      items: (accounts.data ?? []).filter((a) => a.type === type),
    }))
    .filter((g) => g.items.length > 0)

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

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">账户</h1>
          <p className="mt-1 text-sm text-muted-foreground">资产 / 负债 / 收入 / 费用 / 权益</p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/account-types`} className={buttonVariants({ variant: 'outline' })}>
            账户类型
          </Link>
          <Link to={`/ledgers/${ledgerId}/currencies`} className={buttonVariants({ variant: 'outline' })}>
            币种与汇率
          </Link>
          <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
            <Plus /> 开户
          </button>
        </div>
      </div>

      {accounts.loading && (
        <div className="mt-6 grid gap-4 lg:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-32 rounded-xl" />
          ))}
        </div>
      )}
      {accounts.error && <p className="mt-6 text-sm text-destructive">加载失败：{accounts.error}</p>}

      <div className="mt-6 grid gap-4 lg:grid-cols-2">
        {groups.map((group) => (
          <Card key={group.type}>
            <CardHeader>
              <CardTitle className="text-base">
                {typeLabels[group.type] ?? group.type}{' '}
                <span className="text-sm font-normal text-muted-foreground">({group.type})</span>
              </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col">
              {group.items.map((a) => (
                <Link
                  key={a.account}
                  to={`/ledgers/${ledgerId}/accounts/${encodeURIComponent(a.account)}`}
                  className="flex items-center justify-between border-b py-2 text-sm last:border-0 hover:text-primary"
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="truncate font-mono">{displayName(a.account)}</span>
                    <Badge variant="outline" className="hidden font-mono text-[10px] sm:inline-flex">
                      {a.account}
                    </Badge>
                  </span>
                  <span className="ml-3 shrink-0 font-mono tabular-nums">{amountText(a)}</span>
                </Link>
              ))}
            </CardContent>
          </Card>
        ))}
      </div>
      {!accounts.loading && !accounts.error && groups.length === 0 && (
        <Card className="mt-6">
          <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
            <p className="text-sm text-muted-foreground">还没有账户，先开第一个账户</p>
            <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
              <Plus /> 开户
            </button>
          </CardContent>
        </Card>
      )}

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
    </div>
  )
}

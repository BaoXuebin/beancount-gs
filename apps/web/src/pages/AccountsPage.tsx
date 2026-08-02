import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Plus } from 'lucide-react'
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
import type { Account } from '@/api/types'
import { AccountTree } from '@/components/AccountTree'
import { LoadingHint } from '@/components/LoadingHint'
import { cn } from '@/lib/utils'

const typeTabs = [
  { key: 'Assets', label: '资产' },
  { key: 'Liabilities', label: '负债' },
  { key: 'Income', label: '收入' },
  { key: 'Expenses', label: '费用' },
  { key: 'Equity', label: '权益' },
]

export function AccountsPage() {
  const { ledgerId = '' } = useParams()
  const accounts = useFetch<Account[]>(`/ledgers/${ledgerId}/accounts?status=open`)
  const [tab, setTab] = useState('Assets')

  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState({
    account: '',
    opened_on: new Date().toISOString().slice(0, 10),
    currency: 'CNY',
    booking: 'none',
  })

  const byType = (accounts.data ?? []).filter((a) => a.type === tab)
  const counts = typeTabs.reduce<Record<string, number>>((acc, t) => {
    acc[t.key] = (accounts.data ?? []).filter((a) => a.type === t.key).length
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

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">账户</h1>
          <p className="mt-1 text-sm text-muted-foreground">按类型展示，层级结构体现账户归属关系</p>
        </div>
        <div className="flex gap-2">
          <Link to="account-types" className={buttonVariants({ variant: 'outline' })}>
            账户类型
          </Link>
          <Link to="currencies" className={buttonVariants({ variant: 'outline' })}>
            币种与汇率
          </Link>
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
            按「{tab}」前缀的层级关系展开，点击账户名查看详情
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
            <AccountTree accounts={byType} ledgerId={ledgerId} />
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
    </div>
  )
}

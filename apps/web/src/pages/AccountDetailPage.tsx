import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
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
import type { Account } from '@/api/types'
import { LoadingHint } from '@/components/LoadingHint'

export function AccountDetailPage() {
  const { ledgerId = '', account = '' } = useParams()
  const decoded = decodeURIComponent(account)
  const data = useFetch<Account>(`/ledgers/${ledgerId}/accounts/${encodeURIComponent(decoded)}`)
  const [closeOpen, setCloseOpen] = useState(false)
  const [balanceOpen, setBalanceOpen] = useState(false)
  const [closedOn, setClosedOn] = useState(new Date().toISOString().slice(0, 10))
  const [balanceDate, setBalanceDate] = useState(new Date().toISOString().slice(0, 10))
  const [balanceNumber, setBalanceNumber] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const withRevision = async () => {
    const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
    return rev.revision
  }

  const closeAccount = async () => {
    setBusy(true)
    setError(null)
    try {
      const revision = await withRevision()
      await request(`/ledgers/${ledgerId}/accounts/${encodeURIComponent(decoded)}`, {
        method: 'POST',
        body: JSON.stringify({ closed_on: closedOn }),
        revision,
      })
      setCloseOpen(false)
      setNotice('账户已关闭')
      data.refetch()
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

  const reconcileBalance = async () => {
    setBusy(true)
    setError(null)
    try {
      const revision = await withRevision()
      await request(`/ledgers/${ledgerId}/accounts/${encodeURIComponent(decoded)}/balance`, {
        method: 'POST',
        body: JSON.stringify({ date: balanceDate, number: balanceNumber }),
        revision,
      })
      setBalanceOpen(false)
      setBalanceNumber('')
      setNotice('期初对账已写入 pad + balance 指令')
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

  if (data.loading) {
    return (
      <div>
        <LoadingHint className="mb-3" />
        <Skeleton className="h-96" />
      </div>
    )
  }
  if (data.error || !data.data) {
    return <p className="text-sm text-destructive">加载失败：{data.error}</p>
  }

  const a = data.data
  const metrics = [
    { label: '当前市值', value: a.market_number ? `${a.market_number} ${a.market_currency ?? ''}` : '—' },
    { label: '成本 / 盈亏', value: '后端暂未返回（契约已定义）' },
    { label: '开户日', value: a.opened_on ?? '—' },
    { label: '状态', value: a.status === 'open' ? '在用' : '已关闭' },
  ]

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">账户详情</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {decoded} · {a.currency ?? 'CNY'}
          </p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/accounts`} className={buttonVariants({ variant: 'outline' })}>
            返回账户
          </Link>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => {
              setError(null)
              setBalanceOpen(true)
            }}
          >
            期初对账
          </button>
          {a.status === 'open' && (
            <button
              type="button"
              className={buttonVariants({ variant: 'destructive' })}
              onClick={() => {
                setError(null)
                setCloseOpen(true)
              }}
            >
              关闭账户
            </button>
          )}
        </div>
      </div>

      {notice && <p className="mt-4 text-sm text-emerald-600">{notice}</p>}
      {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {metrics.map((m) => (
          <Card key={m.label}>
            <CardHeader>
              <CardTitle className="text-sm font-normal text-muted-foreground">{m.label}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm font-medium">{m.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">持仓</CardTitle>
          <CardDescription>多币种持仓（FIFO）与余额</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>数量</TableHead>
                <TableHead>币种</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(a.positions ?? []).map((p, i) => (
                <TableRow key={i}>
                  <TableCell className="font-mono">{p.number}</TableCell>
                  <TableCell>{p.currency}</TableCell>
                </TableRow>
              ))}
              {(!a.positions || a.positions.length === 0) && (
                <TableRow>
                  <TableCell colSpan={2} className="text-center text-muted-foreground">
                    暂无持仓
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          {a.market_number && (
            <div className="mt-3 flex items-center gap-2 text-sm">
              市值
              <Badge variant="outline" className="font-mono">
                {a.market_number} {a.market_currency}
              </Badge>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={closeOpen} onOpenChange={(o) => !o && setCloseOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>关闭账户</DialogTitle>
            <DialogDescription>写入 closed 指令，账户将不再出现在在用列表</DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5">
            <Label>关闭日期</Label>
            <Input type="date" value={closedOn} onChange={(e) => setClosedOn(e.target.value)} />
          </div>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setCloseOpen(false)}
            >
              取消
            </button>
            <button
              type="button"
              className={buttonVariants({ variant: 'destructive' })}
              disabled={busy}
              onClick={closeAccount}
            >
              {busy ? '处理中…' : '确认关闭'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={balanceOpen} onOpenChange={(o) => !o && setBalanceOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>期初对账</DialogTitle>
            <DialogDescription>写入 pad + balance 指令，用于核对账户余额</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>日期</Label>
              <Input type="date" value={balanceDate} onChange={(e) => setBalanceDate(e.target.value)} />
            </div>
            <div className="grid gap-1.5">
              <Label>余额</Label>
              <Input
                value={balanceNumber}
                onChange={(e) => setBalanceNumber(e.target.value)}
                placeholder="12345.67"
                className="font-mono"
              />
            </div>
          </div>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setBalanceOpen(false)}
            >
              取消
            </button>
            <button
              type="button"
              className={buttonVariants()}
              disabled={busy || !balanceNumber}
              onClick={reconcileBalance}
            >
              {busy ? '处理中…' : '写入对账'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

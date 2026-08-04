import { useEffect, useState } from 'react'
import { buttonVariants } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError, request } from '@/api/client'
import type { AccountDetail } from '@/api/types'
import { AccountDetailBody } from '@/components/AccountDetailBody'
import { LoadingHint } from '@/components/LoadingHint'

function today(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}

interface AccountDetailDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ledgerId: string
  account: string | null
  onChanged?: () => void
}

type Mode = 'detail' | 'close' | 'balance' | 'reopen'

export function AccountDetailDialog({
  open,
  onOpenChange,
  ledgerId,
  account,
  onChanged,
}: AccountDetailDialogProps) {
  const [data, setData] = useState<AccountDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [mode, setMode] = useState<Mode>('detail')
  const [closedOn, setClosedOn] = useState(today())
  const [balanceDate, setBalanceDate] = useState(today())
  const [balanceNumber, setBalanceNumber] = useState('')
  const [reopenedOn, setReopenedOn] = useState(today())
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    if (!open || !account) return
    let cancelled = false
    setLoading(true)
    setError(null)
    setNotice(null)
    setMode('detail')
    setData(null)
    request<AccountDetail>(`/ledgers/${ledgerId}/accounts/${encodeURIComponent(account)}`)
      .then((d) => {
        if (!cancelled) setData(d)
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
  }, [open, ledgerId, account, reloadKey])

  const close = () => onOpenChange(false)

  const closeAccount = async () => {
    if (!account) return
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/accounts/${encodeURIComponent(account)}`, {
        method: 'POST',
        body: JSON.stringify({ closed_on: closedOn }),
        revision: rev.revision,
      })
      setMode('detail')
      setNotice('账户已关闭')
      onChanged?.()
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
    if (!account) return
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/accounts/${encodeURIComponent(account)}/balance`, {
        method: 'POST',
        body: JSON.stringify({ date: balanceDate, number: balanceNumber }),
        revision: rev.revision,
      })
      setMode('detail')
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

  const reopenAccount = async () => {
    if (!account) return
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/accounts/${encodeURIComponent(account)}`, {
        method: 'PUT',
        body: JSON.stringify({ opened_on: reopenedOn }),
        revision: rev.revision,
      })
      setMode('detail')
      setNotice('账户已重新开户')
      setReloadKey((k) => k + 1)
      onChanged?.()
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
    <Dialog open={open} onOpenChange={(o) => !o && close()}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>账户详情</DialogTitle>
          <DialogDescription>
            {account ? `${account} · ${data?.currency ?? 'CNY'}` : ' '}
          </DialogDescription>
        </DialogHeader>

        {notice && <p className="text-sm text-emerald-600">{notice}</p>}
        {error && <p className="text-sm text-destructive">{error}</p>}

        {loading ? (
          <div>
            <LoadingHint className="mb-3" />
            <Skeleton className="h-40" />
          </div>
        ) : mode === 'close' ? (
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>关闭日期</Label>
              <Input type="date" value={closedOn} onChange={(e) => setClosedOn(e.target.value)} />
            </div>
            <p className="text-sm text-muted-foreground">
              写入 closed 指令，账户将不再出现在在用列表
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                className={buttonVariants({ variant: 'outline' })}
                disabled={busy}
                onClick={() => setMode('detail')}
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
            </div>
          </div>
        ) : mode === 'reopen' ? (
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>重新开户日期</Label>
              <Input type="date" value={reopenedOn} onChange={(e) => setReopenedOn(e.target.value)} />
            </div>
            <p className="text-sm text-muted-foreground">
              写入一条新的 open 指令，账户将恢复为「在用」并出现在列表中
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                className={buttonVariants({ variant: 'outline' })}
                disabled={busy}
                onClick={() => setMode('detail')}
              >
                取消
              </button>
              <button
                type="button"
                className={buttonVariants()}
                disabled={busy || !reopenedOn}
                onClick={reopenAccount}
              >
                {busy ? '处理中…' : '确认重新开户'}
              </button>
            </div>
          </div>
        ) : mode === 'balance' ? (
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
            <p className="text-sm text-muted-foreground">
              写入 pad + balance 指令，用于核对账户余额
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                className={buttonVariants({ variant: 'outline' })}
                disabled={busy}
                onClick={() => setMode('detail')}
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
            </div>
          </div>
        ) : data ? (
          <div className="grid gap-4">
            <AccountDetailBody a={data} />
          </div>
        ) : null}

        <div className="-mx-4 -mb-4 mt-auto flex flex-col-reverse gap-2 rounded-b-xl border-t bg-muted/50 p-4 sm:flex-row sm:justify-end">
          <button type="button" className={buttonVariants({ variant: 'outline' })} onClick={close}>
            关闭
          </button>
          {mode === 'detail' && data?.status === 'open' && (
            <>
              <button
                type="button"
                className={buttonVariants({ variant: 'destructive' })}
                onClick={() => {
                  setError(null)
                  setNotice(null)
                  setMode('close')
                }}
              >
                关闭账户
              </button>
              <button
                type="button"
                className={buttonVariants()}
                onClick={() => {
                  setError(null)
                  setNotice(null)
                  setMode('balance')
                }}
              >
                期初对账
              </button>
            </>
          )}
          {mode === 'detail' && data?.status === 'closed' && (
            <button
              type="button"
              className={buttonVariants()}
              onClick={() => {
                setError(null)
                setNotice(null)
                setMode('reopen')
              }}
            >
              重新开户
            </button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError, request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { Transaction } from '@/api/types'
import { cn } from '@/lib/utils'
import { LoadingHint } from '@/components/LoadingHint'
import { TransactionDetailBody } from '@/components/TransactionDetailBody'

export function TransactionDetailPage() {
  const { ledgerId = '', transactionId = '' } = useParams()
  const navigate = useNavigate()
  const txn = useFetch<Transaction>(`/ledgers/${ledgerId}/transactions/${transactionId}`)
  const raw = useFetch<string>(`/ledgers/${ledgerId}/transactions/${transactionId}/raw`)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const remove = async () => {
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/transactions/${transactionId}`, {
        method: 'DELETE',
        revision: rev.revision,
      })
      navigate(`/ledgers/${ledgerId}/transactions`)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setError(`账本已被他人修改（409），当前修订号 ${(err.details as { current_revision?: number })?.current_revision ?? '?'}`)
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
      setConfirmOpen(false)
    } finally {
      setBusy(false)
    }
  }

  if (txn.loading) {
    return (
      <div>
        <LoadingHint className="mb-3" />
        <Skeleton className="h-96" />
      </div>
    )
  }

  const t = txn.data
  if (!t || txn.error) {
    return <p className="text-sm text-destructive">加载失败：{txn.error}</p>
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">交易详情</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t.date} · {t.tags?.map((tag) => `#${tag}`).join(' ') || '无标签'}
          </p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/transactions`} className={buttonVariants({ variant: 'outline' })}>
            返回
          </Link>
          <Link
            to={`/ledgers/${ledgerId}/transactions/${transactionId}/edit`}
            className={buttonVariants({ variant: 'outline' })}
          >
            编辑
          </Link>
          <button
            type="button"
            className={buttonVariants({ variant: 'destructive' })}
            onClick={() => setConfirmOpen(true)}
          >
            删除
          </button>
        </div>
      </div>

      {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

      <div className="mt-4 grid gap-4">
        <TransactionDetailBody t={t} raw={raw.data} rawLoading={raw.loading} rawError={raw.error} />
      </div>

      <Dialog open={confirmOpen} onOpenChange={(open) => !open && setConfirmOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>删除交易</DialogTitle>
            <DialogDescription>此操作会写入账本并增加修订号，确认删除？</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setConfirmOpen(false)}
            >
              取消
            </button>
            <button
              type="button"
              className={cn(buttonVariants({ variant: 'destructive' }))}
              disabled={busy}
              onClick={remove}
            >
              {busy ? '删除中…' : '确认删除'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

import { useEffect, useState } from 'react'
import { Pencil, Trash2 } from 'lucide-react'
import { buttonVariants } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError, request } from '@/api/client'
import type { Transaction } from '@/api/types'
import { TransactionDetailBody } from '@/components/TransactionDetailBody'
import { LoadingHint } from '@/components/LoadingHint'

interface TransactionViewDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ledgerId: string
  transaction: Transaction | null
  onEdit?: (t: Transaction) => void
  onDeleted?: () => void
}

export function TransactionViewDialog({
  open,
  onOpenChange,
  ledgerId,
  transaction,
  onEdit,
  onDeleted,
}: TransactionViewDialogProps) {
  const [raw, setRaw] = useState<string | null>(null)
  const [rawLoading, setRawLoading] = useState(false)
  const [rawError, setRawError] = useState<string | null>(null)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open || !transaction) return
    let cancelled = false
    setRawLoading(true)
    setRawError(null)
    setRaw(null)
    setConfirmingDelete(false)
    setError(null)
    request<string>(`/ledgers/${ledgerId}/transactions/${transaction.id}/raw`)
      .then((text) => {
        if (!cancelled) setRaw(text)
      })
      .catch((err: unknown) => {
        if (!cancelled) setRawError(err instanceof Error ? err.message : String(err))
      })
      .finally(() => {
        if (!cancelled) setRawLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, ledgerId, transaction])

  const close = () => onOpenChange(false)

  const remove = async () => {
    if (!transaction) return
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/transactions/${transaction.id}`, {
        method: 'DELETE',
        revision: rev.revision,
      })
      onDeleted?.()
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setError(`账本已被他人修改（409），当前修订号 ${(err.details as { current_revision?: number })?.current_revision ?? '?'}`)
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
      setConfirmingDelete(false)
    } finally {
      setBusy(false)
    }
  }

  const t = transaction

  return (
    <Dialog open={open} onOpenChange={(o) => !o && close()}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>交易详情</DialogTitle>
          <DialogDescription>
            {t ? `${t.date} · ${t.tags?.map((tag) => `#${tag}`).join(' ') || '无标签'}` : ' '}
          </DialogDescription>
        </DialogHeader>

        {t ? (
          <div className="grid gap-4">
            {error && <p className="text-sm text-destructive">{error}</p>}
            <TransactionDetailBody t={t} raw={raw} rawLoading={rawLoading} rawError={rawError} />
          </div>
        ) : (
          <div>
            <LoadingHint className="mb-3" />
            <Skeleton className="h-40" />
          </div>
        )}

        <div className="-mx-4 -mb-4 mt-auto flex flex-col-reverse gap-2 rounded-b-xl border-t bg-muted/50 p-4 sm:flex-row sm:justify-end">
          {confirmingDelete ? (
            <>
              <span className="flex items-center text-sm text-destructive">
                确认删除该交易？
              </span>
              <button
                type="button"
                className={buttonVariants({ variant: 'outline' })}
                disabled={busy}
                onClick={() => setConfirmingDelete(false)}
              >
                取消
              </button>
              <button
                type="button"
                className={buttonVariants({ variant: 'destructive' })}
                disabled={busy || !t}
                onClick={remove}
              >
                {busy ? '删除中…' : '确认删除'}
              </button>
            </>
          ) : (
            <>
              <button
                type="button"
                className={buttonVariants({ variant: 'outline' })}
                onClick={close}
              >
                关闭
              </button>
              <button
                type="button"
                className={buttonVariants({ variant: 'destructive' })}
                disabled={!t}
                onClick={() => {
                  setError(null)
                  setConfirmingDelete(true)
                }}
              >
                <Trash2 /> 删除
              </button>
              <button
                type="button"
                className={buttonVariants()}
                disabled={!t}
                onClick={() => t && onEdit?.(t)}
              >
                <Pencil /> 编辑
              </button>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

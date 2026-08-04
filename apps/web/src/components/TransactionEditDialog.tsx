import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { TransactionEditForm } from '@/components/TransactionEditForm'
import type { AiRecordResult, TransactionTemplate } from '@/api/types'

interface TransactionEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ledgerId: string
  transactionId?: string | null
  draft?: AiRecordResult['draft'] | null
  initial?: TransactionTemplate | null
  onSaved?: () => void
}

export function TransactionEditDialog({
  open,
  onOpenChange,
  ledgerId,
  transactionId,
  draft,
  initial,
  onSaved,
}: TransactionEditDialogProps) {
  const isEdit = transactionId != null
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? '编辑交易' : '记一笔'}</DialogTitle>
          <DialogDescription>
            字段遵循 beancount 术语：narration / postings / units / cost
          </DialogDescription>
        </DialogHeader>
        <TransactionEditForm
          ledgerId={ledgerId}
          transactionId={transactionId}
          draft={draft}
          initial={initial}
          onCancel={() => onOpenChange(false)}
          onSaved={() => {
            onOpenChange(false)
            onSaved?.()
          }}
        />
      </DialogContent>
    </Dialog>
  )
}

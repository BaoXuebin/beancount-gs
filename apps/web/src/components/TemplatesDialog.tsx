import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { TemplatesBody } from '@/components/TemplatesBody'
import type { TransactionTemplate } from '@/api/types'

interface TemplatesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ledgerId: string
  onUse?: (template: TransactionTemplate) => void
}

export function TemplatesDialog({ open, onOpenChange, ledgerId, onUse }: TemplatesDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onOpenChange(false)}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>交易模板</DialogTitle>
          <DialogDescription>选择模板后自动填充分录，可再调整</DialogDescription>
        </DialogHeader>
        <TemplatesBody
          ledgerId={ledgerId}
          onUse={(t) => {
            onOpenChange(false)
            onUse?.(t)
          }}
        />
      </DialogContent>
    </Dialog>
  )
}

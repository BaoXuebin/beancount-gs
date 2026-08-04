import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { CurrenciesBody } from '@/components/CurrenciesBody'

interface CurrenciesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  ledgerId: string
}

export function CurrenciesDialog({ open, onOpenChange, ledgerId }: CurrenciesDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onOpenChange(false)}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>币种与汇率</DialogTitle>
          <DialogDescription>本位币 CNY · 汇率写入 price/prices.bean</DialogDescription>
        </DialogHeader>
        <CurrenciesBody ledgerId={ledgerId} />
      </DialogContent>
    </Dialog>
  )
}

import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

export function LoadingHint({
  label = '加载中…',
  className,
}: {
  label?: string
  className?: string
}) {
  return (
    <div
      className={cn(
        'flex items-center gap-1.5 text-xs text-muted-foreground',
        className,
      )}
    >
      <Loader2 className="size-3 animate-spin" /> {label}
    </div>
  )
}

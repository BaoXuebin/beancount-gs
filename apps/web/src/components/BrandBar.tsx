import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { UserMenu } from '@/components/UserMenu'

export function BrandBar({
  title,
  backTo,
  backLabel = '返回',
}: {
  title?: string
  backTo?: string
  backLabel?: string
}) {
  return (
    <header className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2.5">
        <div className="flex min-w-0 items-center gap-2">
          <Link to="/workspaces" className="shrink-0 font-medium">
            beancount-gs
          </Link>
          {backTo && (
            <Link
              to={backTo}
              className="flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <ArrowLeft className="size-3.5" /> {backLabel}
            </Link>
          )}
          {title && (
            <span className="truncate border-l pl-2.5 text-sm text-muted-foreground">{title}</span>
          )}
        </div>
        <UserMenu align="end" />
      </div>
    </header>
  )
}

import { Link } from 'react-router-dom'
import { UserMenu } from '@/components/UserMenu'

export function BrandBar({ title }: { title?: string }) {
  return (
    <header className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2.5">
        <div className="flex min-w-0 items-center gap-3">
          <Link to="/workspaces" className="shrink-0 font-medium">
            beancount-gs
          </Link>
          {title && (
            <span className="truncate border-l pl-3 text-sm text-muted-foreground">{title}</span>
          )}
        </div>
        <UserMenu align="end" />
      </div>
    </header>
  )
}

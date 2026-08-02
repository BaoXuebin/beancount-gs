import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { UserMenu } from '@/components/UserMenu'
import { Breadcrumb, type Crumb } from '@/components/Breadcrumb'

export function BrandBar({ crumbs = [] }: { crumbs?: Crumb[] }) {
  return (
    <header className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2.5">
        <div className="flex min-w-0 items-center gap-1.5">
          <Link to="/workspaces" className="shrink-0 font-medium">
            beancount-gs
          </Link>
          {crumbs.length > 0 && (
            <>
              <ChevronRight className="size-3.5 shrink-0 text-muted-foreground/50" />
              <Breadcrumb items={crumbs} />
            </>
          )}
        </div>
        <UserMenu align="end" />
      </div>
    </header>
  )
}

import { Fragment } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface Crumb {
  label: string
  to?: string
}

export function Breadcrumb({ items }: { items: Crumb[] }) {
  return (
    <nav className="flex min-w-0 items-center gap-1 text-sm">
      {items.map((item, i) => {
        const last = i === items.length - 1
        return (
          <Fragment key={i}>
            {i > 0 && <ChevronRight className="size-3.5 shrink-0 text-muted-foreground/50" />}
            {item.to && !last ? (
              <Link
                to={item.to}
                className="shrink-0 truncate text-muted-foreground transition-colors hover:text-foreground"
              >
                {item.label}
              </Link>
            ) : (
              <span className={cn('truncate', last && 'font-medium text-foreground')}>
                {item.label}
              </span>
            )}
          </Fragment>
        )
      })}
    </nav>
  )
}

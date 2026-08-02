import { Fragment } from 'react'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

const steps = ['登录', '工作区', '账本', '功能页']

export function FlowSteps({ current }: { current: number }) {
  return (
    <div className="mt-3 flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
      {steps.map((step, i) => (
        <Fragment key={step}>
          {i > 0 && <ChevronRight className="size-3 text-muted-foreground/40" />}
          <span
            className={cn(
              'transition-colors',
              i === current && 'font-medium text-foreground',
              i > current && 'opacity-60',
            )}
          >
            {step}
          </span>
        </Fragment>
      ))}
    </div>
  )
}

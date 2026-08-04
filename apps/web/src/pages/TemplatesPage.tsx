import { Link, useNavigate, useParams } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import { TemplatesBody } from '@/components/TemplatesBody'

export function TemplatesPage() {
  const { ledgerId = '' } = useParams()
  const navigate = useNavigate()
  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">交易模板</h1>
          <p className="mt-1 text-sm text-muted-foreground">常用交易一键复用，记一笔时填充分录</p>
        </div>
        <Link to={`/ledgers/${ledgerId}/transactions`} className={buttonVariants({ variant: 'outline' })}>
          返回交易
        </Link>
      </div>
      <div className="mt-6">
        <TemplatesBody
          ledgerId={ledgerId}
          onUse={() => navigate(`/ledgers/${ledgerId}/transactions/new`)}
        />
      </div>
    </div>
  )
}

import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { Link } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import { TransactionEditForm } from '@/components/TransactionEditForm'
import type { AiRecordResult } from '@/api/types'

export function TransactionEditPage() {
  const { ledgerId = '', transactionId } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const isEdit = transactionId != null
  const draft = (location.state as { draft?: AiRecordResult['draft'] } | null)?.draft

  const back = () => navigate(`/ledgers/${ledgerId}/transactions`)

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{isEdit ? '编辑交易' : '记一笔'}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            字段遵循 beancount 术语：narration / postings / units / cost
          </p>
        </div>
        <Link to={`/ledgers/${ledgerId}/transactions`} className={buttonVariants({ variant: 'outline' })}>
          返回交易
        </Link>
      </div>
      <div className="mt-4">
        <TransactionEditForm
          ledgerId={ledgerId}
          transactionId={transactionId}
          draft={draft}
          onCancel={back}
          onSaved={back}
        />
      </div>
    </div>
  )
}

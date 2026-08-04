import { Link, useParams } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import { CurrenciesBody } from '@/components/CurrenciesBody'

export function CurrenciesPage() {
  const { ledgerId = '' } = useParams()
  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">币种与汇率</h1>
          <p className="mt-1 text-sm text-muted-foreground">本位币 CNY · 汇率写入 price/prices.bean</p>
        </div>
        <Link to={`/ledgers/${ledgerId}/accounts`} className={buttonVariants({ variant: 'outline' })}>
          返回账户
        </Link>
      </div>
      <div className="mt-6">
        <CurrenciesBody ledgerId={ledgerId} />
      </div>
    </div>
  )
}

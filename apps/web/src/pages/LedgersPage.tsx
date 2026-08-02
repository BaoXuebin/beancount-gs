import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useFetch } from '@/api/useFetch'
import type { Ledger } from '@/api/types'

export function LedgersPage() {
  const { data, error, loading } = useFetch<Ledger[]>('/ledgers')

  return (
    <div>
      <h1 className="text-xl font-semibold">账本</h1>
      <p className="mt-1 text-sm text-muted-foreground">账本归属工作区，多人协作编辑，修订号控制并发</p>
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {loading &&
          Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-32 rounded-xl" />)}
        {error && <p className="text-sm text-destructive">加载失败：{error}</p>}
        {data?.map((ledger) => (
          <Link key={ledger.id} to={`/ledgers/${ledger.id}/transactions`}>
            <Card className="h-full transition-colors hover:border-primary">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">{ledger.name}</CardTitle>
                  <Badge variant="outline">{ledger.operating_currency}</Badge>
                </div>
                <CardDescription>
                  修订 #{ledger.revision} · 成员 {ledger.member_count ?? 0}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <span className="text-sm text-primary">打开账本 →</span>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  )
}

import { Link, useParams } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useFetch } from '@/api/useFetch'
import type { Currency } from '@/api/types'
import { NotImplemented } from '@/components/NotImplemented'

export function CurrenciesPage() {
  const { ledgerId = '' } = useParams()
  const currencies = useFetch<Currency[]>(`/ledgers/${ledgerId}/currencies`)
  const notImplemented = currencies.errorStatus === 404

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

      {currencies.loading && <Skeleton className="mt-6 h-40" />}
      {notImplemented && <NotImplemented feature="币种与汇率" />}
      {!currencies.loading && !notImplemented && currencies.error && (
        <p className="mt-6 text-sm text-destructive">加载失败：{currencies.error}</p>
      )}
      {currencies.data && (
        <Card className="mt-6">
          <CardHeader>
            <CardTitle className="text-base">币种列表</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>币种</TableHead>
                  <TableHead>名称</TableHead>
                  <TableHead className="text-right">汇率（本位币）</TableHead>
                  <TableHead>更新日期</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {currencies.data.map((c) => (
                  <TableRow key={c.currency}>
                    <TableCell className="font-mono">{c.currency}</TableCell>
                    <TableCell>{c.name}</TableCell>
                    <TableCell className="text-right font-mono">{c.price ?? '—'}</TableCell>
                    <TableCell>{c.price_date ?? '—'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

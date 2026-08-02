import { useParams } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useFetch } from '@/api/useFetch'
import type { Ledger, Transaction, TransactionListResponse } from '@/api/types'

function postingsSummary(t: Transaction): string {
  return t.postings.map((p) => p.account).join(' / ')
}

function amountSummary(t: Transaction): string {
  const withUnits = t.postings.find((p) => p.units != null)
  if (!withUnits?.units) return ''
  return `${withUnits.units.number} ${withUnits.units.currency}`
}

export function TransactionsPage() {
  const { ledgerId = '' } = useParams()
  const ledger = useFetch<Ledger>(`/ledgers/${ledgerId}`)
  const txns = useFetch<TransactionListResponse>(`/ledgers/${ledgerId}/transactions`)

  return (
    <div>
      <h1 className="text-xl font-semibold">{ledger.data?.name ?? '交易'}</h1>
      <p className="mt-1 text-sm text-muted-foreground">
        共 {txns.data?.total ?? 0} 笔交易 · 修订 #{ledger.data?.revision ?? 0}
      </p>
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">交易列表</CardTitle>
          <CardDescription>字段遵循 beancount 术语：narration / postings / units / cost</CardDescription>
        </CardHeader>
        <CardContent>
          {txns.error && <p className="text-sm text-destructive">加载失败：{txns.error}</p>}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>日期</TableHead>
                <TableHead>收款方</TableHead>
                <TableHead>描述</TableHead>
                <TableHead>账户</TableHead>
                <TableHead className="text-right">金额</TableHead>
                <TableHead>标签</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {txns.data?.items.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="whitespace-nowrap">{t.date}</TableCell>
                  <TableCell>{t.payee ?? '-'}</TableCell>
                  <TableCell>{t.narration ?? '-'}</TableCell>
                  <TableCell className="max-w-[220px] truncate">{postingsSummary(t)}</TableCell>
                  <TableCell className="text-right font-mono">{amountSummary(t)}</TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      {t.tags?.map((tag) => (
                        <Badge key={tag} variant="secondary">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

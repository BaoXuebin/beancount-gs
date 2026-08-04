import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { Transaction } from '@/api/types'

interface TransactionDetailBodyProps {
  t: Transaction
  raw: string | null
  rawLoading: boolean
  rawError: string | null
}

export function TransactionDetailBody({ t, raw, rawLoading, rawError }: TransactionDetailBodyProps) {
  return (
    <>
      <Card>
        <CardContent className="grid gap-4 pt-6 sm:grid-cols-2">
          <div>
            <p className="text-xs text-muted-foreground">日期</p>
            <p className="mt-0.5 text-sm">{t.date}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">收款方 / 描述</p>
            <p className="mt-0.5 text-sm">
              {t.payee ?? '—'} · {t.narration ?? '—'}
            </p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">标志</p>
            <p className="mt-0.5 text-sm font-mono">{t.flag ?? '*'}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">交易 ID</p>
            <p className="mt-0.5 break-words font-mono text-sm">{t.id}</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">分录（{t.postings.length} 条）</CardTitle>
        </CardHeader>
        <CardContent>
          <Table className="table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead>账户</TableHead>
                <TableHead className="w-24 text-right">金额</TableHead>
                <TableHead className="w-20">币种</TableHead>
                <TableHead className="w-44">成本 / 汇率</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {t.postings.map((p, i) => (
                <TableRow key={i}>
                  <TableCell className="align-top whitespace-normal">
                    <div className="break-words font-mono">{p.account}</div>
                  </TableCell>
                  <TableCell className="align-top text-right font-mono">{p.units?.number ?? '—'}</TableCell>
                  <TableCell className="align-top">{p.units?.currency ?? '—'}</TableCell>
                  <TableCell className="align-top font-mono">
                    <div className="break-words">
                      {p.cost ? `${p.cost.number} ${p.cost.currency}` : '—'}
                      {p.price ? ` @ ${p.price.number} ${p.price.currency}` : ''}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">原始文本（只读）</CardTitle>
          <CardDescription>可切换到「源文件」直接编辑原始 bean 文本</CardDescription>
        </CardHeader>
        <CardContent>
          {rawLoading && <Skeleton className="h-24" />}
          <pre className="max-h-64 overflow-auto rounded-lg bg-muted p-4 text-xs leading-relaxed">
            {raw ?? rawError ?? '—'}
          </pre>
        </CardContent>
      </Card>
    </>
  )
}
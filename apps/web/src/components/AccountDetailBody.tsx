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
import type { Account } from '@/api/types'

interface AccountDetailBodyProps {
  a: Account
}

export function AccountDetailBody({ a }: AccountDetailBodyProps) {
  const metrics = [
    { label: '当前市值', value: a.market_number ? `${a.market_number} ${a.market_currency ?? ''}` : '—' },
    { label: '成本 / 盈亏', value: '后端暂未返回（契约已定义）' },
    { label: '开户日', value: a.opened_on ?? '—' },
    { label: '状态', value: a.status === 'open' ? '在用' : '已关闭' },
  ]

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {metrics.map((m) => (
          <Card key={m.label}>
            <CardHeader>
              <CardTitle className="text-sm font-normal text-muted-foreground">{m.label}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm font-medium">{m.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">持仓</CardTitle>
          <CardDescription>多币种持仓（FIFO）与余额</CardDescription>
        </CardHeader>
        <CardContent>
          <Table className="table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead>数量</TableHead>
                <TableHead>币种</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(a.positions ?? []).map((p, i) => (
                <TableRow key={i}>
                  <TableCell className="font-mono">{p.number}</TableCell>
                  <TableCell>{p.currency}</TableCell>
                </TableRow>
              ))}
              {(!a.positions || a.positions.length === 0) && (
                <TableRow>
                  <TableCell colSpan={2} className="text-center text-muted-foreground">
                    暂无持仓
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          {a.market_number && (
            <div className="mt-3 flex items-center gap-2 text-sm">
              市值
              <Badge variant="outline" className="font-mono">
                {a.market_number} {a.market_currency}
              </Badge>
            </div>
          )}
        </CardContent>
      </Card>
    </>
  )
}
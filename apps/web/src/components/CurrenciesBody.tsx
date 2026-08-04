import { useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
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
import { request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { Currency } from '@/api/types'
import { NotImplemented } from '@/components/NotImplemented'
import { cn } from '@/lib/utils'

export function CurrenciesBody({ ledgerId }: { ledgerId: string }) {
  const currencies = useFetch<Currency[]>(`/ledgers/${ledgerId}/currencies`)
  const notImplemented = currencies.errorStatus === 404
  const [syncing, setSyncing] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  const sync = async () => {
    setSyncing(true)
    setMessage(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      const list = await request<Currency[]>(`/ledgers/${ledgerId}/currencies/sync`, {
        method: 'POST',
        revision: rev.revision,
      })
      setMessage(`已同步 ${list.length} 个币种的最新汇率`)
      currencies.refetch()
    } catch (err) {
      setMessage(err instanceof Error ? err.message : String(err))
    } finally {
      setSyncing(false)
    }
  }

  if (currencies.loading && currencies.data == null) {
    return <Skeleton className="h-40" />
  }
  if (notImplemented) {
    return <NotImplemented feature="币种与汇率" />
  }
  if (currencies.error) {
    return <p className="text-sm text-destructive">加载失败：{currencies.error}</p>
  }

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-muted-foreground">汇率相对本位币 {currencies.data?.find((c) => c.is_operating)?.currency ?? 'CNY'}，写入 price/prices.bean</p>
        <button
          type="button"
          className={cn(buttonVariants({ variant: 'outline', size: 'sm' }))}
          disabled={syncing}
          onClick={sync}
        >
          <RefreshCw className={cn('size-3.5', syncing && 'animate-spin')} />
          {syncing ? '同步中…' : '同步汇率'}
        </button>
      </div>
      {message && (
        <p className={cn('text-sm', message.startsWith('汇率同步完成') ? 'text-emerald-600' : 'text-destructive')}>
          {message}
        </p>
      )}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">币种列表</CardTitle>
          <CardDescription>名称 / 符号来自 .beancount-gs/currency.json</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>币种</TableHead>
                <TableHead>名称 / 符号</TableHead>
                <TableHead className="text-right">汇率（本位币）</TableHead>
                <TableHead>更新日期</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {currencies.data?.map((c) => (
                <TableRow key={c.currency}>
                  <TableCell className="font-mono">
                    <span className="flex items-center gap-1.5">
                      {c.currency}
                      {c.is_operating && (
                        <Badge variant="secondary" className="text-[10px]">
                          本位币
                        </Badge>
                      )}
                    </span>
                  </TableCell>
                  <TableCell>
                    {c.name || '—'}
                    {c.symbol ? <span className="ml-1.5 text-muted-foreground">({c.symbol})</span> : null}
                  </TableCell>
                  <TableCell className="text-right font-mono">{c.price ?? '—'}</TableCell>
                  <TableCell>{c.price_date ?? '—'}</TableCell>
                </TableRow>
              ))}
              {currencies.data && currencies.data.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="py-6 text-center text-muted-foreground">
                    暂无币种
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}

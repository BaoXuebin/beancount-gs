import { useParams } from 'react-router-dom'
import { Link } from 'react-router-dom'
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
import type { AuditLog } from '@/api/types'
import { NotImplemented } from '@/components/NotImplemented'

export function AuditPage() {
  const { ledgerId = '' } = useParams()
  const logs = useFetch<AuditLog[]>(`/audit-logs?ledger_id=${ledgerId}`)
  const notImplemented = logs.errorStatus === 404

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">审计日志</h1>
          <p className="mt-1 text-sm text-muted-foreground">账本操作记录 · 仅 owner 可见</p>
        </div>
        <Link to={`/ledgers/${ledgerId}/settings`} className={buttonVariants({ variant: 'outline' })}>
          返回设置
        </Link>
      </div>

      {logs.loading && <Skeleton className="mt-6 h-40" />}
      {notImplemented && <NotImplemented feature="审计日志" />}
      {!logs.loading && !notImplemented && logs.error && (
        <p className="mt-6 text-sm text-destructive">加载失败：{logs.error}</p>
      )}
      {logs.data && (
        <Card className="mt-6">
          <CardHeader>
            <CardTitle className="text-base">操作记录</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>时间</TableHead>
                  <TableHead>用户</TableHead>
                  <TableHead>动作</TableHead>
                  <TableHead>对象</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.data.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell className="whitespace-nowrap font-mono text-xs">
                      {log.created_at}
                    </TableCell>
                    <TableCell>{log.actor}</TableCell>
                    <TableCell>{log.action}</TableCell>
                    <TableCell className="max-w-[200px] truncate font-mono text-xs">
                      {log.object ?? '—'}
                    </TableCell>
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

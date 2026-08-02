import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ApiError, request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { AccountTypeMapping } from '@/api/types'

export function AccountTypesPage() {
  const { ledgerId = '' } = useParams()
  const types = useFetch<AccountTypeMapping[]>(`/ledgers/${ledgerId}/account-types`)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [prefix, setPrefix] = useState('')
  const [name, setName] = useState('')

  const create = async () => {
    if (!prefix.trim() || !name.trim()) {
      setError('前缀和名称不能为空')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/account-types`, {
        method: 'POST',
        body: JSON.stringify({ prefix: prefix.trim(), name: name.trim() }),
        revision: rev.revision,
      })
      setOpen(false)
      setPrefix('')
      setName('')
      types.refetch()
    } catch (err) {
      if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
        setError('账本已被他人修改（409），请刷新后重试')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">账户类型</h1>
          <p className="mt-1 text-sm text-muted-foreground">账户前缀 → 中文名称映射，用于统计展示</p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/accounts`} className={buttonVariants({ variant: 'outline' })}>
            返回账户
          </Link>
          <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
            <Plus /> 新增类型
          </button>
        </div>
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">类型映射</CardTitle>
          <CardDescription>未匹配前缀默认使用账户最后一个节点作为名称</CardDescription>
        </CardHeader>
        <CardContent>
          {types.loading && <Skeleton className="h-32" />}
          {types.error && <p className="text-sm text-destructive">加载失败：{types.error}</p>}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>前缀</TableHead>
                <TableHead>名称</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(types.data ?? []).map((t) => (
                <TableRow key={t.prefix}>
                  <TableCell className="font-mono">{t.prefix}</TableCell>
                  <TableCell>{t.name}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={(o) => !o && setOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新增账户类型</DialogTitle>
            <DialogDescription>例如前缀 Assets → 名称 资产</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>前缀</Label>
              <Input value={prefix} onChange={(e) => setPrefix(e.target.value)} placeholder="Assets" />
            </div>
            <div className="grid gap-1.5">
              <Label>名称</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="资产" />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setOpen(false)}
            >
              取消
            </button>
            <button type="button" className={buttonVariants()} disabled={busy} onClick={create}>
              {busy ? '保存中…' : '保存'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

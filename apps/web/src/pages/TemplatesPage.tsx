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
import { ApiError, request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { TransactionTemplate } from '@/api/types'
import { NotImplemented } from '@/components/NotImplemented'

export function TemplatesPage() {
  const { ledgerId = '' } = useParams()
  const templates = useFetch<TransactionTemplate[]>(`/ledgers/${ledgerId}/templates`)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState({ name: '', payee: '', narration: '', tags: '' })

  const notImplemented = templates.errorStatus === 404

  const create = async () => {
    setBusy(true)
    setError(null)
    try {
      await request(`/ledgers/${ledgerId}/templates`, {
        method: 'POST',
        body: JSON.stringify({
          name: form.name,
          payee: form.payee || undefined,
          narration: form.narration,
          tags: form.tags
            .split(/[,，\s]+/)
            .map((s) => s.replace(/^#/, ''))
            .filter(Boolean),
          postings: [],
        }),
      })
      setOpen(false)
      templates.refetch()
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('模板接口尚未实现（OpenAPI 已定义），无法保存')
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
          <h1 className="text-xl font-semibold">交易模板</h1>
          <p className="mt-1 text-sm text-muted-foreground">常用交易一键复用，记一笔时填充分录</p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/transactions`} className={buttonVariants({ variant: 'outline' })}>
            返回交易
          </Link>
          <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
            <Plus /> 新建模板
          </button>
        </div>
      </div>

      {templates.loading && <Skeleton className="mt-6 h-40" />}
      {notImplemented && <NotImplemented feature="交易模板" />}
      {!templates.loading && !notImplemented && templates.error && (
        <p className="mt-6 text-sm text-destructive">加载失败：{templates.error}</p>
      )}

      {templates.data && (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {templates.data.map((t) => (
            <Card key={t.id}>
              <CardHeader>
                <CardTitle className="text-base">{t.name}</CardTitle>
                <CardDescription className="truncate font-mono text-xs">
                  {t.postings.map((p) => p.account).join(' → ') || t.narration}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Link
                  to={`/ledgers/${ledgerId}/transactions/new`}
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                >
                  使用
                </Link>
              </CardContent>
            </Card>
          ))}
          {templates.data.length === 0 && (
            <p className="col-span-full py-8 text-center text-sm text-muted-foreground">还没有模板</p>
          )}
        </div>
      )}

      <Dialog open={open} onOpenChange={(o) => !o && setOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新建模板</DialogTitle>
            <DialogDescription>保存常用交易结构，下次一键填充</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>模板名称</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="如「工资入账」"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>收款方 Payee</Label>
              <Input
                value={form.payee}
                onChange={(e) => setForm((f) => ({ ...f, payee: e.target.value }))}
                placeholder="如「公司」"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>描述 Narration</Label>
              <Input
                value={form.narration}
                onChange={(e) => setForm((f) => ({ ...f, narration: e.target.value }))}
                placeholder="如「工资」"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>标签</Label>
              <Input
                value={form.tags}
                onChange={(e) => setForm((f) => ({ ...f, tags: e.target.value }))}
                placeholder="#Salary（可选）"
              />
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
              {busy ? '保存中…' : '保存模板'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

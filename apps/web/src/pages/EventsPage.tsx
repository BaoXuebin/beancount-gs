import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
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
import type { Event } from '@/api/types'
import { NotImplemented } from '@/components/NotImplemented'
import { LoadingHint } from '@/components/LoadingHint'

export function EventsPage() {
  const { ledgerId = '' } = useParams()
  const events = useFetch<Event[]>(`/ledgers/${ledgerId}/events`)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [form, setForm] = useState({
    date: new Date().toISOString().slice(0, 10),
    type: '生活',
    description: '',
  })

  const notImplemented = events.errorStatus === 404

  const create = async () => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/events`, {
        method: 'POST',
        body: JSON.stringify(form),
        revision: rev.revision,
      })
      setOpen(false)
      setNotice('事件已添加')
      events.refetch()
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('事件接口尚未实现（OpenAPI 已定义），无法保存')
      } else if (err instanceof ApiError && err.code === 'LEDGER_STALE') {
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
      {events.loading && <LoadingHint className="mb-2" />}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">事件</h1>
          <p className="mt-1 text-sm text-muted-foreground">账本时间线 · 按日期倒序</p>
        </div>
        <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
          <Plus /> 添加事件
        </button>
      </div>

      {notice && <p className="mt-4 text-sm text-emerald-600">{notice}</p>}
      {events.loading && <Skeleton className="mt-6 h-40" />}
      {notImplemented && <NotImplemented feature="事件" />}
      {!events.loading && !notImplemented && events.error && (
        <p className="mt-6 text-sm text-destructive">加载失败：{events.error}</p>
      )}
      {events.data && (
        <Card className="mt-6">
          <CardHeader>
            <CardTitle className="text-base">时间线</CardTitle>
            <CardDescription>账本生命周期中的重要节点</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col">
            {events.data.map((e, i) => (
              <div
                key={i}
                className="flex items-center justify-between border-b py-2.5 text-sm last:border-0"
              >
                <span className="font-mono">{e.date}</span>
                <span className="flex-1 px-3">{e.description}</span>
                <Badge variant="outline">{e.type}</Badge>
              </div>
            ))}
            {events.data.length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground">还没有事件</p>
            )}
          </CardContent>
        </Card>
      )}

      <Dialog open={open} onOpenChange={(o) => !o && setOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加事件</DialogTitle>
            <DialogDescription>事件写入 event/events.bean 的 event 指令</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>日期</Label>
              <Input
                type="date"
                value={form.date}
                onChange={(e) => setForm((f) => ({ ...f, date: e.target.value }))}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>类型</Label>
              <Input
                value={form.type}
                onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))}
                placeholder="工作 / 生活 / 投资"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>描述</Label>
              <Input
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                placeholder="如「入职新公司」"
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
              {busy ? '保存中…' : '保存'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { Ledger, LedgerCreate, Team } from '@/api/types'

export function LedgersPage() {
  const ledgers = useFetch<Ledger[]>('/ledgers')
  const teams = useFetch<Team[]>('/teams')

  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState<Partial<LedgerCreate>>({ operating_currency: 'CNY' })

  const openDialog = () => {
    setError(null)
    setForm((prev) => ({
      ...prev,
      team_id: prev.team_id ?? teams.data?.[0]?.id ?? '',
    }))
    setOpen(true)
  }

  const create = async () => {
    if (!form.team_id) {
      setError('请选择工作区')
      return
    }
    if (!form.name?.trim()) {
      setError('请填写账本名称')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await request<Ledger>('/ledgers', {
        method: 'POST',
        body: JSON.stringify({
          team_id: form.team_id,
          name: form.name.trim(),
          operating_currency: form.operating_currency?.trim() || 'CNY',
          ...(form.start_date ? { start_date: form.start_date } : {}),
        }),
      })
      setOpen(false)
      setForm({ operating_currency: 'CNY' })
      ledgers.refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const isEmpty =
    !ledgers.loading && !ledgers.error && ledgers.data != null && ledgers.data.length === 0

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">账本</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            账本归属工作区，多人协作编辑，修订号控制并发
          </p>
        </div>
        <Button onClick={openDialog}>
          <Plus /> 新建账本
        </Button>
      </div>

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {ledgers.loading &&
          Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-32 rounded-xl" />)}
        {ledgers.error && <p className="text-sm text-destructive">加载失败：{ledgers.error}</p>}
        {ledgers.data?.map((ledger) => (
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

      {isEmpty && (
        <Card className="mt-6">
          <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
            <p className="text-sm text-muted-foreground">还没有账本，先创建第一个账本开始记账</p>
            <Button onClick={openDialog}>
              <Plus /> 创建账本
            </Button>
          </CardContent>
        </Card>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新建账本</DialogTitle>
            <DialogDescription>账本创建在你拥有 editor 及以上权限的工作区</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>工作区</Label>
              <Select
                value={form.team_id ?? null}
                onValueChange={(value) => value && setForm((prev) => ({ ...prev, team_id: value }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择工作区" />
                </SelectTrigger>
                <SelectContent>
                  {teams.data?.map((team) => (
                    <SelectItem key={team.id} value={team.id}>
                      {team.name}（{team.role}）
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {teams.error && (
                <p className="text-xs text-destructive">工作区加载失败：{teams.error}</p>
              )}
            </div>
            <div className="grid gap-1.5">
              <Label>账本名称</Label>
              <Input
                value={form.name ?? ''}
                onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
                placeholder="如：家庭账本"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>记账币种</Label>
              <Input
                value={form.operating_currency ?? ''}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, operating_currency: e.target.value }))
                }
                placeholder="CNY"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>起始日期（可选）</Label>
              <Input
                type="date"
                value={form.start_date ?? ''}
                onChange={(e) => setForm((prev) => ({ ...prev, start_date: e.target.value }))}
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              取消
            </Button>
            <Button onClick={create} disabled={busy}>
              {busy ? '创建中…' : '创建'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

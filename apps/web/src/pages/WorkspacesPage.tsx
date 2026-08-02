import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight, Plus } from 'lucide-react'
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
import { request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import type { Team } from '@/api/types'
import { BrandBar } from '@/components/BrandBar'
import { FlowSteps } from '@/components/FlowSteps'

export function WorkspacesPage() {
  const teams = useFetch<Team[]>('/teams')
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [name, setName] = useState('')

  const create = async () => {
    if (!name.trim()) {
      setError('请输入工作区名称')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await request<Team>('/teams', {
        method: 'POST',
        body: JSON.stringify({ name: name.trim() }),
      })
      setOpen(false)
      setName('')
      teams.refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col">
      <BrandBar title="工作区" />
      <div className="mx-auto w-full max-w-6xl flex-1 px-4 py-8">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold">工作区</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              团队是权限边界：账本归属工作区，成员按角色协作
            </p>
            <FlowSteps current={1} />
          </div>
          <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
            <Plus /> 新建工作区
          </button>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {teams.loading &&
            Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-32 rounded-xl" />
            ))}
          {teams.error && <p className="text-sm text-destructive">加载失败：{teams.error}</p>}
          {teams.data?.map((team) => (
            <Link key={team.id} to="/ledgers" className="group">
              <Card className="h-full transition-colors hover:border-primary">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base">{team.name}</CardTitle>
                    <Badge variant={team.role === 'owner' ? 'default' : 'secondary'}>
                      {team.role}
                    </Badge>
                  </div>
                  <CardDescription>
                    {team.member_count ?? 0} 位成员 · {team.ledger_count ?? 0} 个账本
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex justify-end">
                  <span className="inline-flex items-center gap-1 rounded-lg border border-border px-2.5 py-1 text-xs font-medium text-primary transition-colors group-hover:border-primary/50 group-hover:bg-primary/5">
                    进入 <ArrowRight className="size-3" />
                  </span>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
        {!teams.loading && !teams.error && teams.data && teams.data.length === 0 && (
          <Card className="mt-6">
            <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
              <p className="text-sm text-muted-foreground">
                还没有工作区（首次登录会自动创建个人工作区），也可以手动创建
              </p>
              <button type="button" className={buttonVariants()} onClick={() => setOpen(true)}>
                <Plus /> 新建工作区
              </button>
            </CardContent>
          </Card>
        )}

        <Dialog open={open} onOpenChange={(o) => !o && setOpen(false)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>新建工作区</DialogTitle>
              <DialogDescription>工作区是权限边界，创建者自动成为 owner</DialogDescription>
            </DialogHeader>
            <div className="grid gap-1.5">
              <Label>工作区名称</Label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="如：家庭"
                onKeyDown={(e) => e.key === 'Enter' && create()}
              />
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
                {busy ? '创建中…' : '创建'}
              </button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}

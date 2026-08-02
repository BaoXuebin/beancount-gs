import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
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
import type { Ledger, Membership } from '@/api/types'
import { NotImplemented } from '@/components/NotImplemented'
import { LoadingHint } from '@/components/LoadingHint'

export function SettingsPage() {
  const { ledgerId = '' } = useParams()
  const navigate = useNavigate()
  const ledger = useFetch<Ledger>(`/ledgers/${ledgerId}`)
  const members = useFetch<Membership[]>(`/ledgers/${ledgerId}/members`)
  const [inviteOpen, setInviteOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [invite, setInvite] = useState({ github_login_or_email: '', role: 'editor' })

  const membersNotImplemented = members.errorStatus === 404

  const sendInvite = async () => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await request(`/ledgers/${ledgerId}/members`, {
        method: 'POST',
        body: JSON.stringify(invite),
      })
      setInviteOpen(false)
      setNotice(`已邀请 ${invite.github_login_or_email}（${invite.role}）`)
      members.refetch()
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('成员邀请接口尚未实现（OpenAPI 已定义），无法发送')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setBusy(false)
    }
  }

  const removeLedger = async () => {
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}`, { method: 'DELETE', revision: rev.revision })
      navigate('/ledgers')
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('删除账本接口尚未实现（OpenAPI 已定义），无法删除')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
      setDeleteOpen(false)
    } finally {
      setBusy(false)
    }
  }

  const l = ledger.data

  return (
    <div>
      {ledger.loading && <LoadingHint className="mb-2" />}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">设置</h1>
          <p className="mt-1 text-sm text-muted-foreground">账本「{l?.name ?? '…'}」 · 我的角色 {l?.role}</p>
        </div>
      </div>

      {notice && <p className="mt-4 text-sm text-emerald-600">{notice}</p>}
      {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

      {ledger.loading && <Skeleton className="mt-6 h-48" />}
      {ledger.error && <p className="mt-6 text-sm text-destructive">加载失败：{ledger.error}</p>}
      {l && (
        <>
          <Card className="mt-6">
            <CardHeader>
              <CardTitle className="text-base">账本设置</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              <div>
                <p className="text-xs text-muted-foreground">账本名称</p>
                <p className="mt-0.5 text-sm">{l.name}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">记账本位币</p>
                <p className="mt-0.5 text-sm">{l.operating_currency}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">起始日期</p>
                <p className="mt-0.5 text-sm">{l.start_date ?? '—'}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">期初权益账户</p>
                <p className="mt-0.5 font-mono text-sm">{l.opening_balances ?? 'Equity:OpeningBalances'}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">修订号</p>
                <p className="mt-0.5 text-sm">#{l.revision}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">写前备份</p>
                <p className="mt-0.5 text-sm">{l.is_bak ? '已开启（bak/ 快照）' : '已关闭'}</p>
              </div>
            </CardContent>
          </Card>

          <Card className="mt-4">
            <CardHeader>
              <CardTitle className="text-base">成员</CardTitle>
              <CardDescription>账本成员与角色 · 角色决定可执行的操作</CardDescription>
            </CardHeader>
            <CardContent>
              {members.loading && <Skeleton className="h-20" />}
              {membersNotImplemented && <NotImplemented feature="成员管理" />}
              {!members.loading && !membersNotImplemented && members.error && (
                <p className="text-sm text-destructive">加载失败：{members.error}</p>
              )}
              {members.data && (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>用户</TableHead>
                      <TableHead>角色</TableHead>
                      <TableHead>状态</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {members.data.map((m) => (
                      <TableRow key={m.user_id}>
                        <TableCell>{m.display_name || m.github_login}</TableCell>
                        <TableCell>{m.role}</TableCell>
                        <TableCell>{m.status}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
              <div className="mt-3">
                <button
                  type="button"
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                  onClick={() => {
                    setError(null)
                    setInviteOpen(true)
                  }}
                >
                  + 邀请成员（GitHub 用户名 / 邮箱）
                </button>
              </div>
            </CardContent>
          </Card>

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">备份与导出</CardTitle>
                <CardDescription>写前快照自动备份到 bak/；账本文件即资产，可随时带走</CardDescription>
              </CardHeader>
              <CardContent>
                <Link
                  to={`/ledgers/${ledgerId}/settings/export`}
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                >
                  导出与备份管理
                </Link>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">AI 与集成</CardTitle>
                <CardDescription>AI 能力开关与模型配置；MCP / API 让 Agent 接入账本</CardDescription>
              </CardHeader>
              <CardContent className="flex gap-2">
                <Link
                  to={`/ledgers/${ledgerId}/settings/ai`}
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                >
                  AI 设置
                </Link>
                <Link
                  to={`/ledgers/${ledgerId}/settings/integrations`}
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                >
                  MCP 与 API
                </Link>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">审计</CardTitle>
                <CardDescription>谁在何时对账本做了什么操作，仅 owner 可见</CardDescription>
              </CardHeader>
              <CardContent>
                <Link
                  to={`/ledgers/${ledgerId}/settings/audit`}
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                >
                  打开审计日志
                </Link>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">危险区</CardTitle>
                <CardDescription>删除账本需二次确认，仅 owner 可执行</CardDescription>
              </CardHeader>
              <CardContent>
                <button
                  type="button"
                  className={buttonVariants({ variant: 'destructive', size: 'sm' })}
                  onClick={() => {
                    setError(null)
                    setDeleteOpen(true)
                  }}
                >
                  删除账本
                </button>
              </CardContent>
            </Card>
          </div>
        </>
      )}

      <Dialog open={inviteOpen} onOpenChange={(o) => !o && setInviteOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>邀请成员</DialogTitle>
            <DialogDescription>被邀请人需先通过 GitHub 登录，接受后按角色开放权限</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>GitHub 用户名 / 邮箱</Label>
              <Input
                value={invite.github_login_or_email}
                onChange={(e) => setInvite((f) => ({ ...f, github_login_or_email: e.target.value }))}
                placeholder="多个用逗号分隔"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>角色</Label>
              <Input
                value={invite.role}
                onChange={(e) => setInvite((f) => ({ ...f, role: e.target.value }))}
                placeholder="editor（默认）/ owner / viewer"
              />
            </div>
          </div>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setInviteOpen(false)}
            >
              取消
            </button>
            <button type="button" className={buttonVariants()} disabled={busy} onClick={sendInvite}>
              {busy ? '发送中…' : '发送邀请'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteOpen} onOpenChange={(o) => !o && setDeleteOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>删除账本</DialogTitle>
            <DialogDescription>
              此操作不可恢复，将删除账本文件与元数据。确认继续？
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => setDeleteOpen(false)}
            >
              取消
            </button>
            <button
              type="button"
              className={buttonVariants({ variant: 'destructive' })}
              disabled={busy}
              onClick={removeLedger}
            >
              {busy ? '删除中…' : '确认删除'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

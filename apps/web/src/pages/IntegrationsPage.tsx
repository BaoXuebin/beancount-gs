import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Copy, Plus } from 'lucide-react'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
import type { ApiKey } from '@/api/types'
import { LoadingHint } from '@/components/LoadingHint'

const tools = [
  ['list_ledgers', '列出当前用户可访问的账本', '只读'],
  ['query_transactions', '按日期 / 账户 / 金额条件查询交易', '只读'],
  ['query_accounts', '账户余额与持仓', '只读'],
  ['query_stats', '统计与趋势数据', '只读'],
  ['create_transaction', '新建交易（先预览后提交）', '读写'],
  ['update_transaction', '更新交易', '读写'],
  ['delete_transaction', '删除交易', '读写'],
  ['import_transactions', '批量导入账单', '读写'],
  ['read_source_file', '读取 bean 源文件', '只读'],
  ['write_source_file', '编辑源文件（带修订号校验）', '读写'],
  ['ai_record', '自然语言记账（AI 生成 + 确认）', 'AI'],
  ['ai_summarize', '账本总结与洞察', 'AI'],
]

export function IntegrationsPage() {
  const { ledgerId = '' } = useParams()
  const keys = useFetch<ApiKey[]>('/api-keys')
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [form, setForm] = useState({ name: '', scope: 'read-only' })
  const [secret, setSecret] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const mcpUrl = `${window.location.origin}/mcp`
  const configJson = JSON.stringify(
    {
      mcpServers: {
        'beancount-gs': {
          type: 'http',
          url: mcpUrl,
          headers: { Authorization: 'Bearer <API_KEY>' },
        },
      },
    },
    null,
    2,
  )

  const createKey = async () => {
    setBusy(true)
    setError(null)
    setSecret(null)
    try {
      const created = await request<ApiKey & { secret: string }>('/api-keys', {
        method: 'POST',
        body: JSON.stringify({ name: form.name, scope: form.scope }),
      })
      setSecret(created.secret)
      setForm({ name: '', scope: 'read-only' })
      keys.refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const revoke = async (id: string) => {
    setError(null)
    try {
      await request(`/api-keys/${id}`, { method: 'DELETE' })
      keys.refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  const copy = async (text: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div>
      {keys.loading && <LoadingHint className="mb-2" />}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">集成与 MCP</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            通过 MCP / API 让 Agent（Claude、Cursor 等）接入账本
          </p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/settings`} className={buttonVariants({ variant: 'outline' })}>
            返回设置
          </Link>
          <Link to={`/ledgers/${ledgerId}/settings/api-docs`} className={buttonVariants({ variant: 'outline' })}>
            API 文档
          </Link>
        </div>
      </div>

      {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">MCP Server</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-1.5 text-sm">
          <div className="flex items-center gap-2">
            状态
            <Badge variant="secondary">运行中</Badge>
          </div>
          <div>
            地址 <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">{mcpUrl}</code>
          </div>
          <div>传输：Streamable HTTP（Bearer API Key 认证）</div>
          <div className="mt-2 flex gap-2">
            <button
              type="button"
              className={buttonVariants({ variant: 'outline', size: 'sm' })}
              onClick={() => copy(configJson)}
            >
              <Copy /> {copied ? '已复制' : '复制 MCP 配置'}
            </button>
            <Link
              to={`/ledgers/${ledgerId}/settings/audit`}
              className={buttonVariants({ variant: 'ghost', size: 'sm' })}
            >
              查看调用记录
            </Link>
          </div>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">暴露的工具</CardTitle>
          <CardDescription>工具权限继承当前用户角色：viewer 只读，editor 可写，owner 可管理</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>工具</TableHead>
                <TableHead>说明</TableHead>
                <TableHead>权限</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tools.map(([name, desc, scope]) => (
                <TableRow key={name}>
                  <TableCell className="font-mono">{name}</TableCell>
                  <TableCell>{desc}</TableCell>
                  <TableCell>
                    <Badge variant={scope === '只读' ? 'outline' : scope === '读写' ? 'secondary' : 'default'}>
                      {scope}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">API Keys（Agent 接入凭据）</CardTitle>
          <CardDescription>权限范围：read-only / read-write / ai；密钥仅创建时显示一次</CardDescription>
        </CardHeader>
        <CardContent>
          {keys.loading && <Skeleton className="h-24" />}
          {keys.error && <p className="text-sm text-destructive">加载失败：{keys.error}</p>}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>权限范围</TableHead>
                <TableHead>最近使用</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(keys.data ?? []).map((k) => (
                <TableRow key={k.id}>
                  <TableCell>{k.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{k.scope}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{k.last_used_at ?? '从未使用'}</TableCell>
                  <TableCell>{k.revoked ? '已吊销' : '有效'}</TableCell>
                  <TableCell className="text-right">
                    {!k.revoked && (
                      <button
                        type="button"
                        className={buttonVariants({ variant: 'destructive', size: 'xs' })}
                        onClick={() => revoke(k.id)}
                      >
                        吊销
                      </button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="mt-3">
            <button
              type="button"
              className={buttonVariants({ variant: 'outline', size: 'sm' })}
              onClick={() => setOpen(true)}
            >
              <Plus /> 创建 API Key
            </button>
          </div>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">接入示例</CardTitle>
          <CardDescription>适用于 Claude Code / Cursor 等支持 MCP 的 Agent</CardDescription>
        </CardHeader>
        <CardContent>
          <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-xs">{configJson}</pre>
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={(o) => !o && setOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>创建 API Key</DialogTitle>
            <DialogDescription>密钥只显示一次，请立即复制保存</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-1.5">
              <Label>名称</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="claude-code / cursor-agent"
              />
            </div>
            <div className="grid gap-1.5">
              <Label>权限范围</Label>
              <Select
                value={form.scope}
                onValueChange={(value) => value && setForm((f) => ({ ...f, scope: value }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="read-only">read-only</SelectItem>
                  <SelectItem value="read-write">read-write</SelectItem>
                  <SelectItem value="ai">ai</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {secret && (
              <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/5 p-3">
                <p className="text-xs text-muted-foreground">新密钥（仅此一次可见）：</p>
                <code className="mt-1 block break-all font-mono text-sm">{secret}</code>
              </div>
            )}
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
          <DialogFooter>
            {secret ? (
              <button
                type="button"
                className={buttonVariants()}
                onClick={() => {
                  copy(secret)
                  setSecret(null)
                  setOpen(false)
                }}
              >
                我已复制，关闭
              </button>
            ) : (
              <>
                <button
                  type="button"
                  className={buttonVariants({ variant: 'outline' })}
                  onClick={() => setOpen(false)}
                >
                  取消
                </button>
                <button
                  type="button"
                  className={buttonVariants()}
                  disabled={busy || !form.name.trim()}
                  onClick={createKey}
                >
                  {busy ? '创建中…' : '创建'}
                </button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

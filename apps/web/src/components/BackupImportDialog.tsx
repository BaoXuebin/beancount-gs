import { useState } from 'react'
import { buttonVariants } from '@/components/ui/button'
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
import { ApiError, request } from '@/api/client'
import type { BackupImportResult, Ledger, Team } from '@/api/types'
import { cn } from '@/lib/utils'

type Mode = 'new' | 'existing'

export function BackupImportDialog({
  open,
  onOpenChange,
  teams,
  ledgers,
  onImported,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  teams: Team[]
  ledgers: Ledger[]
  onImported: () => void
}) {
  const [mode, setMode] = useState<Mode>('new')
  const [file, setFile] = useState<File | null>(null)
  const [teamId, setTeamId] = useState('')
  const [name, setName] = useState('')
  const [currency, setCurrency] = useState('CNY')
  const [ledgerId, setLedgerId] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<BackupImportResult | null>(null)

  const openChange = (next: boolean) => {
    if (!next) {
      onOpenChange(false)
      return
    }
    setMode('new')
    setFile(null)
    setError(null)
    setResult(null)
    setTeamId(teams[0]?.id ?? '')
    setLedgerId(ledgers[0]?.id ?? '')
    setName('')
    setCurrency('CNY')
    onOpenChange(true)
  }

  const submit = async () => {
    if (!file) {
      setError('请选择 zip 备份文件')
      return
    }
    if (mode === 'new' && !teamId) {
      setError('请选择工作区')
      return
    }
    if (mode === 'new' && !name.trim()) {
      setError('请填写账本名称')
      return
    }
    if (mode === 'existing' && !ledgerId) {
      setError('请选择目标账本')
      return
    }
    setBusy(true)
    setError(null)
    setResult(null)
    try {
      const form = new FormData()
      form.append('file', file)
      let res: BackupImportResult
      if (mode === 'new') {
        form.append('team_id', teamId)
        form.append('name', name.trim())
        form.append('operating_currency', currency.trim() || 'CNY')
        res = await request<BackupImportResult>('/ledgers/import', {
          method: 'POST',
          body: form,
        })
      } else {
        const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
        res = await request<BackupImportResult>(`/ledgers/${ledgerId}/import`, {
          method: 'POST',
          body: form,
          revision: rev.revision,
        })
      }
      setResult(res)
      onImported()
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
    <Dialog open={open} onOpenChange={openChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>导入 v1 账本 / 备份 zip</DialogTitle>
          <DialogDescription>
            解压并校验（目录结构 + bean-check）后导入；zip 必须包含根目录 index.bean
          </DialogDescription>
        </DialogHeader>

        <div className="flex gap-1 rounded-lg border bg-muted/40 p-1">
          {(
            [
              ['new', '新建账本'],
              ['existing', '导入已有账本'],
            ] as [Mode, string][]
          ).map(([m, label]) => (
            <button
              key={m}
              type="button"
              className={cn(
                'flex-1 rounded-md px-3 py-1.5 text-sm transition-colors',
                mode === m
                  ? 'bg-background font-medium shadow-sm'
                  : 'text-muted-foreground hover:text-foreground',
              )}
              onClick={() => {
                setMode(m)
                setError(null)
                setResult(null)
              }}
            >
              {label}
            </button>
          ))}
        </div>

        <div className="grid gap-4">
          {mode === 'new' ? (
            <>
              <div className="grid gap-1.5">
                <Label>工作区</Label>
                <Select
                  value={teamId || null}
                  onValueChange={(value) => value && setTeamId(value)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="选择工作区" />
                  </SelectTrigger>
                  <SelectContent>
                    {teams.map((t) => (
                      <SelectItem key={t.id} value={t.id}>
                        {t.name}（{t.role}）
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-1.5">
                <Label>账本名称</Label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="如：导入的家庭账本"
                />
              </div>
              <div className="grid gap-1.5">
                <Label>记账本位币</Label>
                <Input
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value)}
                  placeholder="CNY"
                />
              </div>
            </>
          ) : (
            <div className="grid gap-1.5">
              <Label>目标账本</Label>
              <Select
                value={ledgerId || null}
                onValueChange={(value) => value && setLedgerId(value)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择账本" />
                </SelectTrigger>
                <SelectContent>
                  {ledgers.map((l) => (
                    <SelectItem key={l.id} value={l.id}>
                      {l.name}（修订 #{l.revision}）
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                覆盖账本文件前会先快照到 bak/，导入后修订号 +1
              </p>
            </div>
          )}

          <div className="grid gap-1.5">
            <Label>备份文件（.zip）</Label>
            <Input
              type="file"
              accept=".zip"
              onChange={(e) => {
                setFile(e.target.files?.[0] ?? null)
                setResult(null)
              }}
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}
          {result && (
            <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/5 p-3 text-sm">
              <p className="font-medium">导入成功</p>
              {result.ledger && (
                <p className="mt-0.5 text-xs text-muted-foreground">
                  账本：{result.ledger.name}（{result.ledger.operating_currency}）
                </p>
              )}
              {result.revision !== undefined && (
                <p className="text-xs text-muted-foreground">修订号：#{result.revision}</p>
              )}
              <p className="mt-1 text-xs text-muted-foreground">
                写入 {result.files?.length ?? 0} 个文件：
              </p>
              <ul className="mt-0.5 max-h-24 overflow-auto font-mono text-[11px] text-muted-foreground">
                {(result.files ?? []).slice(0, 20).map((f) => (
                  <li key={f}>{f}</li>
                ))}
                {(result.files?.length ?? 0) > 20 && <li>…</li>}
              </ul>
            </div>
          )}
        </div>

        <DialogFooter>
          <button
            type="button"
            className={buttonVariants({ variant: 'outline' })}
            onClick={() => onOpenChange(false)}
          >
            关闭
          </button>
          <button
            type="button"
            className={buttonVariants()}
            disabled={busy || !file || result != null}
            onClick={submit}
          >
            {busy ? '导入中…' : '开始导入'}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

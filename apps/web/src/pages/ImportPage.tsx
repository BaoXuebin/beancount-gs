import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { LedgerNav } from '@/components/LedgerNav'
import { request } from '@/api/client'
import type { ImportResult, ImportRow } from '@/api/types'

const sources = [
  { value: 'alipay', label: '支付宝' },
  { value: 'wechat', label: '微信支付' },
  { value: 'icbc', label: '工商银行' },
  { value: 'abc', label: '农业银行' },
]

export function ImportPage() {
  const { ledgerId = '' } = useParams()
  const [source, setSource] = useState('alipay')
  const [file, setFile] = useState<File | null>(null)
  const [rows, setRows] = useState<ImportRow[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [busy, setBusy] = useState(false)

  const upload = async () => {
    if (!file) return
    setBusy(true)
    setError(null)
    setResult(null)
    try {
      const form = new FormData()
      form.append('file', file)
      const data = await request<ImportRow[]>(`/ledgers/${ledgerId}/imports/${source}`, {
        method: 'POST',
        body: form,
      })
      setRows(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const confirm = async () => {
    if (!rows) return
    setBusy(true)
    setError(null)
    try {
      const rev = await request<{ ledger_id: string; revision: number }>(
        `/ledgers/${ledgerId}/revision`,
      )
      const data = await request<ImportResult>(
        `/ledgers/${ledgerId}/imports/${source}/confirm`,
        {
          method: 'POST',
          body: JSON.stringify({ rows }),
          revision: rev.revision,
        },
      )
      setResult(data)
      setRows(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  const total = useMemo(
    () => rows?.reduce((sum, r) => sum + Math.abs(Number(r.number)), 0) ?? 0,
    [rows],
  )

  return (
    <div>
      <h1 className="text-xl font-semibold">导入账单</h1>
      <LedgerNav />

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">上传账单</CardTitle>
          <CardDescription>GBK / UTF-8 自动识别 · 默认币种 CNY · 预览后再确认落账</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-wrap items-end gap-4">
            <div className="grid gap-1.5">
              <Label>来源</Label>
              <Select value={source} onValueChange={(value) => value && setSource(value)}>
                <SelectTrigger className="w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {sources.map((s) => (
                    <SelectItem key={s.value} value={s.value}>
                      {s.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <Label>CSV 文件</Label>
              <Input type="file" accept=".csv" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
            </div>
            <Button onClick={upload} disabled={busy || !file}>
              上传预览
            </Button>
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          {result && (
            <p className="text-sm text-muted-foreground">
              导入完成：成功 {result.created} 笔，失败 {result.failed.length} 笔
            </p>
          )}
        </CardContent>
      </Card>

      {rows && (
        <Card className="mt-4">
          <CardHeader>
            <CardTitle className="text-base">预览（共 {rows.length} 行，合计 {total.toFixed(2)}）</CardTitle>
            <CardDescription>请为每行确认账户（Expenses: / Income: 前缀），确认后写入账本</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>日期</TableHead>
                  <TableHead>收款方</TableHead>
                  <TableHead>描述</TableHead>
                  <TableHead className="text-right">金额</TableHead>
                  <TableHead>账户</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row) => (
                  <TableRow key={row.index}>
                    <TableCell className="whitespace-nowrap">{row.date}</TableCell>
                    <TableCell>{row.payee ?? '-'}</TableCell>
                    <TableCell>{row.narration ?? '-'}</TableCell>
                    <TableCell className="text-right font-mono">{row.number}</TableCell>
                    <TableCell>
                      <Input
                        defaultValue={row.suggested_account ?? ''}
                        onChange={(e) => {
                          const next = [...rows]
                          next[row.index].suggested_account = e.target.value
                          setRows(next)
                        }}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <div>
              <Button onClick={confirm} disabled={busy}>
                确认导入
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

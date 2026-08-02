import { useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Send } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { API_BASE } from '@/api/client'
import { useFetch } from '@/api/useFetch'

const methodColors: Record<string, string> = {
  GET: 'bg-emerald-500/15 text-emerald-700',
  POST: 'bg-sky-500/15 text-sky-700',
  PUT: 'bg-amber-500/15 text-amber-700',
  DELETE: 'bg-rose-500/15 text-rose-700',
}

export function ApiDocsPage() {
  const { ledgerId = '' } = useParams()
  const spec = useFetch<{
    paths?: Record<string, Record<string, { summary?: string; tags?: string[] }>>
  }>('/openapi.json')

  const [method, setMethod] = useState('GET')
  const [path, setPath] = useState('/ledgers/{ledger_id}/transactions')
  const [headers, setHeaders] = useState('')
  const [body, setBody] = useState('')
  const [sending, setSending] = useState(false)
  const [status, setStatus] = useState<string | null>(null)
  const [response, setResponse] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const groups = useMemo(() => {
    const map = new Map<string, [string, string, string][]>()
    for (const [p, ops] of Object.entries(spec.data?.paths ?? {})) {
      for (const [m, op] of Object.entries(ops)) {
        if (!['get', 'post', 'put', 'delete', 'patch'].includes(m)) continue
        const tag = op.tags?.[0] ?? p.split('/')[2] ?? 'other'
        if (!map.has(tag)) map.set(tag, [])
        map.get(tag)!.push([m.toUpperCase(), p, op.summary ?? ''])
      }
    }
    return [...map.entries()]
  }, [spec.data])

  const sendRequest = async () => {
    setSending(true)
    setError(null)
    setResponse(null)
    setStatus(null)
    try {
      const parsedHeaders = new Headers()
      for (const line of headers.split('\n')) {
        const idx = line.indexOf(':')
        if (idx > 0) parsedHeaders.set(line.slice(0, idx).trim(), line.slice(idx + 1).trim())
      }
      const res = await fetch(`${API_BASE}${path}`, {
        method,
        headers: parsedHeaders,
        body: ['POST', 'PUT', 'PATCH'].includes(method) && body ? body : undefined,
        credentials: 'include',
      })
      setStatus(`${res.status} ${res.statusText}`)
      setResponse(await res.text())
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSending(false)
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">API 文档与调试</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            OpenAPI 3.0 · Base URL：{window.location.origin}
            {API_BASE} · 认证：Cookie / Bearer
          </p>
        </div>
        <div className="flex gap-2">
          <Link to={`/ledgers/${ledgerId}/settings/integrations`} className={buttonVariants({ variant: 'outline' })}>
            返回集成
          </Link>
          <a
            className={buttonVariants({ variant: 'outline' })}
            href={`${API_BASE}/openapi.json`}
            target="_blank"
            rel="noreferrer"
          >
            查看 openapi.json
          </a>
        </div>
      </div>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="flex items-center gap-1.5 text-base">
            <Send className="size-4" /> 接口测试面板
          </CardTitle>
          <CardDescription>使用当前登录会话发送请求，查看真实响应</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="grid gap-3 sm:grid-cols-[110px_1fr]">
            <div className="grid gap-1.5">
              <Label>Method</Label>
              <Select value={method} onValueChange={(value) => value && setMethod(value)}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {['GET', 'POST', 'PUT', 'DELETE'].map((m) => (
                    <SelectItem key={m} value={m}>
                      {m}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <Label>URL（相对 {API_BASE}）</Label>
              <Input
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/ledgers/{ledger_id}/transactions"
                className="font-mono"
              />
            </div>
          </div>
          <div className="grid gap-1.5">
            <Label>Headers（每行一个 Key: Value）</Label>
            <Input
              value={headers}
              onChange={(e) => setHeaders(e.target.value)}
              placeholder="Authorization: Bearer bgsk_xxx&#10;If-Revision-Match: 128"
              className="font-mono"
            />
          </div>
          <div className="grid gap-1.5">
            <Label>Body (JSON)</Label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder='{"date":"2026-08-02","postings":[...]}'
              className="h-28 w-full resize-y rounded-lg border bg-muted p-3 font-mono text-xs outline-none focus:border-primary"
            />
          </div>
          <div className="flex items-center gap-3">
            <button
              type="button"
              className={buttonVariants()}
              disabled={sending || !path}
              onClick={sendRequest}
            >
              {sending ? '发送中…' : '发送请求'}
            </button>
            {status && <Badge variant="outline">{status}</Badge>}
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          {response != null && (
            <pre className="max-h-64 overflow-auto rounded-lg bg-muted p-3 font-mono text-xs">
              {response}
            </pre>
          )}
        </CardContent>
      </Card>

      <div className="mt-6 flex flex-col gap-6">
        {spec.loading && <Skeleton className="h-64" />}
        {spec.error && <p className="text-sm text-destructive">加载契约失败：{spec.error}</p>}
        {groups.map(([tag, ops]) => (
          <div key={tag}>
            <h2 className="mb-2 text-sm font-semibold">{tag}</h2>
            <Card>
              <CardContent className="flex flex-col divide-y">
                {ops.map(([m, p, summary]) => (
                  <div key={`${m}${p}`} className="flex flex-wrap items-center gap-2 py-2.5 text-sm">
                    <span className={`w-16 rounded px-1.5 py-0.5 text-center font-mono text-xs ${methodColors[m]}`}>
                      {m}
                    </span>
                    <code className="font-mono text-xs">{p}</code>
                    <span className="text-xs text-muted-foreground">{summary}</span>
                    <button
                      type="button"
                      className="ml-auto text-xs text-primary hover:underline"
                      onClick={() => {
                        setMethod(m)
                        setPath(p)
                      }}
                    >
                      填入测试
                    </button>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>
        ))}
      </div>
    </div>
  )
}

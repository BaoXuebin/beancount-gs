import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError, request } from '@/api/client'
import { useFetch } from '@/api/useFetch'
import { NotImplemented } from '@/components/NotImplemented'
import { cn } from '@/lib/utils'
import { LoadingHint } from '@/components/LoadingHint'

export function SourceFilesPage() {
  const { ledgerId = '' } = useParams()
  const files = useFetch<string[]>(`/ledgers/${ledgerId}/source-files`)
  const [selected, setSelected] = useState<string | null>(null)
  const [content, setContent] = useState<string | null>(null)
  const [loadingFile, setLoadingFile] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const notImplemented = files.errorStatus === 404

  const loadFile = async (path: string) => {
    setSelected(path)
    setLoadingFile(true)
    setError(null)
    setNotice(null)
    try {
      const text = await request<string>(`/ledgers/${ledgerId}/source-files/${encodeURIComponent(path)}`)
      setContent(text)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      setContent(null)
    } finally {
      setLoadingFile(false)
    }
  }

  const save = async () => {
    if (!selected || content == null) return
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      const rev = await request<{ revision: number }>(`/ledgers/${ledgerId}/revision`)
      await request(`/ledgers/${ledgerId}/source-files/${encodeURIComponent(selected)}`, {
        method: 'PUT',
        body: content,
        headers: { 'Content-Type': 'text/plain' },
        revision: rev.revision,
      })
      setNotice('已保存并通过 bean-check 校验')
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
      {files.loading && <LoadingHint className="mb-2" />}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">源文件</h1>
          <p className="mt-1 text-sm text-muted-foreground">beancount 文本账本 · 保存前校验语法</p>
        </div>
        <button
          type="button"
          className={buttonVariants()}
          disabled={busy || selected == null || content == null}
          onClick={save}
        >
          {busy ? '保存中…' : '保存并校验'}
        </button>
      </div>

      {notice && <p className="mt-4 text-sm text-emerald-600">{notice}</p>}
      {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

      {files.loading && <Skeleton className="mt-6 h-64" />}
      {notImplemented && <NotImplemented feature="源文件" />}
      {!files.loading && !notImplemented && files.error && (
        <p className="mt-6 text-sm text-destructive">加载失败：{files.error}</p>
      )}

      {files.data && (
        <div className="mt-6 grid gap-4 lg:grid-cols-[220px_1fr]">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">文件树</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-0.5">
              {files.data.map((path) => (
                <button
                  key={path}
                  type="button"
                  className={cn(
                    'rounded-md px-2 py-1.5 text-left font-mono text-xs hover:bg-accent',
                    selected === path && 'bg-accent font-medium',
                  )}
                  onClick={() => loadFile(path)}
                >
                  {path}
                </button>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">编辑器{selected ? `（${selected}）` : ''}</CardTitle>
              <CardDescription>修改后保存会先备份到 bak/ 并运行 bean-check</CardDescription>
            </CardHeader>
            <CardContent>
              {loadingFile && <Skeleton className="h-64" />}
              {!loadingFile && selected == null && (
                <p className="py-16 text-center text-sm text-muted-foreground">
                  从左侧选择文件开始编辑
                </p>
              )}
              {!loadingFile && selected != null && (
                <textarea
                  value={content ?? ''}
                  onChange={(e) => setContent(e.target.value)}
                  className="h-96 w-full resize-y rounded-lg border bg-muted p-3 font-mono text-xs outline-none focus:border-primary"
                  spellCheck={false}
                />
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}

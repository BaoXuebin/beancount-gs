import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { API_BASE } from '@/api/client'

export function LoginPage() {
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [backendOk, setBackendOk] = useState<boolean | null>(null)

  useEffect(() => {
    fetch(`${API_BASE}/health`, { credentials: 'include' })
      .then((res) => setBackendOk(res.ok))
      .catch(() => setBackendOk(false))
  }, [])

  const login = async () => {
    setBusy(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/auth/github/login`, {
        method: 'GET',
        redirect: 'manual',
        credentials: 'include',
      })
      if (res.type === 'opaqueredirect') {
        window.location.href = `${API_BASE}/auth/github/login`
        return
      }
      if (res.status === 503) {
        setError('GitHub OAuth 尚未配置：请在后端 config.yaml 中填写 github_client_id / github_client_secret 后重启')
        return
      }
      if (res.status === 500) {
        setError('后端服务不可用（500）：请确认后端已启动（cd apps/api && go run ./cmd/server），且端口与 Vite 代理一致（默认 10000）')
        return
      }
      const body = (await res.json().catch(() => null)) as { message?: string } | null
      setError(body?.message ?? `登录失败（${res.status}）`)
    } catch {
      setError('无法连接后端服务，请确认后端已启动（go run ./cmd/server）')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">beancount-gs</CardTitle>
          <CardDescription>多人协作的 beancount 记账服务</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button onClick={login} disabled={busy}>
            {busy ? '跳转中…' : '使用 GitHub 登录'}
          </Button>
          {backendOk === false && (
            <p className="text-sm text-destructive">
              无法连接后端服务：请先启动后端（cd apps/api && go run ./cmd/server），再刷新本页
            </p>
          )}
          {error && <p className="text-sm text-destructive">{error}</p>}
          <p className="text-center text-xs text-muted-foreground">
            首次登录自动创建个人工作区；邀请成员后即可多人协作同一账本
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

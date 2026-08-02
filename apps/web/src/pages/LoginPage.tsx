import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { API_BASE } from '@/api/client'

export function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">beancount-gs</CardTitle>
          <CardDescription>多人协作的 beancount 记账服务</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <a
            href={`${API_BASE}/auth/github/login`}
            className="inline-flex h-10 items-center justify-center rounded-md bg-zinc-900 px-4 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
          >
            使用 GitHub 登录
          </a>
          <p className="text-center text-xs text-muted-foreground">
            首次登录自动创建个人工作区；邀请成员后即可多人协作同一账本
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

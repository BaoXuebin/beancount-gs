import { Link } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export function ErrorPage({ code = 404 }: { code?: 403 | 404 }) {
  const is403 = code === 403
  return (
    <div className="flex min-h-[60vh] items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col items-center gap-4 py-10 text-center">
          <p className="text-5xl font-bold">{code}</p>
          <h1 className="text-xl font-semibold">{is403 ? '权限不足' : '页面不存在'}</h1>
          <p className="text-sm text-muted-foreground">
            {is403
              ? '当前角色为 viewer，无法执行写操作（记账 / 导入 / 编辑源文件）。请联系账本 owner 提升角色，或切换到有编辑权限的账本。'
              : '链接可能已失效，或账本已被删除 / 你没有访问权限。'}
          </p>
          <div className="flex gap-2">
            <Link to={is403 ? '/workspaces' : '/ledgers'} className={buttonVariants({ variant: 'outline' })}>
              {is403 ? '切换账本' : '前往账本列表'}
            </Link>
            <Link to="/workspaces" className={buttonVariants()}>
              前往工作区
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

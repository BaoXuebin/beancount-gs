import { Link, useParams } from 'react-router-dom'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export function ExportPage() {
  const { ledgerId = '' } = useParams()
  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">导出与备份</h1>
          <p className="mt-1 text-sm text-muted-foreground">数据可随时带走，账本文件即资产</p>
        </div>
        <Link to={`/ledgers/${ledgerId}/settings`} className={buttonVariants({ variant: 'outline' })}>
          返回设置
        </Link>
      </div>

      <div className="mt-6 grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">导出账本文件</CardTitle>
            <CardDescription>打包 index.bean / month / account / price / event 为 zip</CardDescription>
          </CardHeader>
          <CardContent>
            <button
              type="button"
              className={buttonVariants()}
              onClick={() => window.alert('导出接口尚未实现（计划中），可先通过「源文件」逐文件查看或下载')}
            >
              立即导出
            </button>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">导出元数据</CardTitle>
            <CardDescription>交易模板 / 账户类型 / 币种配置（JSON）</CardDescription>
          </CardHeader>
          <CardContent>
            <button
              type="button"
              className={buttonVariants({ variant: 'outline' })}
              onClick={() => window.alert('元数据导出接口尚未实现（计划中）')}
            >
              导出元数据
            </button>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">备份历史</CardTitle>
          <CardDescription>写前快照自动备份到 bak/；SQLite 元数据每日快照</CardDescription>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          最近备份：由服务端 cron 每日执行；写前快照随每次编辑生成。
        </CardContent>
      </Card>
    </div>
  )
}

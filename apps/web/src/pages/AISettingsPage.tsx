import { Link, useParams } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const toggles = [
  ['记账助手（自然语言生成交易）', '已开启'],
  ['智能分类（导入 / 记账时建议账户）', '已开启'],
  ['洞察与异常检测（重复扣款 / 大额支出）', '已开启'],
  ['月度财务总结', '已关闭'],
]

export function AISettingsPage() {
  const { ledgerId = '' } = useParams()
  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">AI 设置</h1>
          <p className="mt-1 text-sm text-muted-foreground">AI 能力开关与模型配置 · 配置在服务端 config.yaml</p>
        </div>
        <Link to={`/ledgers/${ledgerId}/settings`} className={buttonVariants({ variant: 'outline' })}>
          返回设置
        </Link>
      </div>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">功能开关</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col">
          {toggles.map(([label, state]) => (
            <div key={label} className="flex items-center justify-between border-b py-2.5 text-sm last:border-0">
              <span>{label}</span>
              <Badge variant={state === '已开启' ? 'default' : 'outline'}>{state}</Badge>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">模型配置</CardTitle>
          <CardDescription>提供商 / 模型 / API Key / 出站代理在服务端 config.yaml 中配置（ai_provider / ai_api_key / ai_model / http_proxy）</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 text-sm">
          <div className="grid gap-1.5 sm:grid-cols-2">
            <div>
              <p className="text-xs text-muted-foreground">提供商</p>
              <p className="mt-0.5">openai / deepseek / 兼容 API（服务端配置）</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">模型</p>
              <p className="mt-0.5">deepseek-chat / gpt-4o 等（服务端配置）</p>
            </div>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">API Key</p>
            <p className="mt-0.5">服务端加密存储，已配置</p>
          </div>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-base">数据与隐私</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-1.5 text-sm">
          <div className="flex justify-between border-b py-2 last:border-0">
            <span>AI 可访问范围</span>
            <span className="text-muted-foreground">当前账本（只读查询 + 待确认写入）</span>
          </div>
          <div className="flex justify-between border-b py-2 last:border-0">
            <span>发送给第三方模型的数据</span>
            <span className="text-muted-foreground">仅当前账本内容，不含登录与会话信息</span>
          </div>
          <p className="mt-2 text-xs text-muted-foreground">
            自托管场景可切换 Ollama 本地模型，账本数据不出服务器；AI 生成的写入仍走「预览 → 确认 → 入账」并记录审计日志。
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

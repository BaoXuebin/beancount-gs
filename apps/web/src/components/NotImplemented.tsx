import { Card, CardContent } from '@/components/ui/card'

export function NotImplemented({ feature }: { feature: string }) {
  return (
    <Card className="mt-4">
      <CardContent className="flex flex-col items-center gap-2 py-12 text-center">
        <p className="text-sm font-medium">「{feature}」的后端接口尚未实现</p>
        <p className="max-w-md text-xs text-muted-foreground">
          OpenAPI 契约已定义（packages/contracts/openapi.yaml），接口实现后本页面会自动显示数据。
        </p>
      </CardContent>
    </Card>
  )
}

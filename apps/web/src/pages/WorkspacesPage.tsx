import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useFetch } from '@/api/useFetch'
import type { Team } from '@/api/types'

export function WorkspacesPage() {
  const { data, error, loading } = useFetch<Team[]>('/teams')

  return (
    <div>
      <h1 className="text-xl font-semibold">工作区</h1>
      <p className="mt-1 text-sm text-muted-foreground">团队是权限边界：账本归属工作区，成员按角色协作</p>
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {loading &&
          Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-32 rounded-xl" />)}
        {error && <p className="text-sm text-destructive">加载失败：{error}</p>}
        {data?.map((team) => (
          <Link key={team.id} to="/ledgers">
            <Card className="h-full transition-colors hover:border-primary">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">{team.name}</CardTitle>
                  <Badge variant={team.role === 'owner' ? 'default' : 'secondary'}>{team.role}</Badge>
                </div>
                <CardDescription>
                  成员 {team.member_count ?? 0} · 账本 {team.ledger_count ?? 0}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <span className="text-sm text-primary">进入 →</span>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  )
}

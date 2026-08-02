import { Link, NavLink, Outlet, useParams } from 'react-router-dom'
import {
  BarChart3,
  CalendarDays,
  FileCode,
  LayoutDashboard,
  ListOrdered,
  ListRestart,
  Settings,
  Sparkles,
  Upload,
  Wallet,
} from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { UserMenu } from '@/components/UserMenu'
import { useFetch } from '@/api/useFetch'
import type { Ledger } from '@/api/types'
import { cn } from '@/lib/utils'

const navItems = [
  { to: 'dashboard', label: '仪表盘', desc: '资产与月度概览', icon: LayoutDashboard },
  { to: 'transactions', label: '交易', desc: '流水 / 记账 / 筛选', icon: ListOrdered },
  { to: 'accounts', label: '账户', desc: '持仓与余额', icon: Wallet },
  { to: 'stats', label: '统计', desc: '趋势 / 占比 / 桑基', icon: BarChart3 },
  { to: 'import', label: '导入', desc: '账单 CSV 导入', icon: Upload },
  { to: 'events', label: '事件', desc: '时间线', icon: CalendarDays },
  { to: 'source', label: '源文件', desc: 'bean 文件编辑', icon: FileCode },
  { to: 'ai', label: 'AI 助手', desc: '自然语言记账 / 问答', icon: Sparkles },
  { to: 'settings', label: '设置', desc: '账本与成员', icon: Settings },
]

export function LedgerLayout() {
  const { ledgerId = '' } = useParams()
  const ledger = useFetch<Ledger>(`/ledgers/${ledgerId}`)

  return (
    <div className="flex flex-col gap-6 lg:flex-row">
      <aside className="flex w-full shrink-0 flex-col gap-4 lg:sticky lg:top-0 lg:h-screen lg:w-56 lg:py-6">
        {/* 我的账本卡片 */}
        <div className="rounded-xl border bg-card p-3">
          <p className="text-[10px] tracking-wide text-muted-foreground uppercase">我的账本</p>
          {ledger.loading ? (
            <Skeleton className="mt-2 h-4 w-24" />
          ) : (
            <p className="mt-1 truncate text-sm font-semibold">{ledger.data?.name ?? '账本'}</p>
          )}
          <p className="mt-0.5 text-xs text-muted-foreground">
            {ledger.loading ? (
              <Skeleton className="mt-1 h-3 w-20" />
            ) : (
              <>
                本位币 {ledger.data?.operating_currency} · 修订 #{ledger.data?.revision ?? 0}
              </>
            )}
          </p>
          <Link
            to="/ledgers"
            className="mt-2 flex items-center gap-1 text-xs text-primary transition-colors hover:text-primary/80"
          >
            <ListRestart className="size-3.5" /> 返回账本列表
          </Link>
        </div>

        {/* 菜单 */}
        <nav className="flex flex-1 flex-wrap gap-1 lg:flex-col">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={`/ledgers/${ledgerId}/${item.to}`}
              className={({ isActive }) =>
                cn(
                  'relative flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground',
                  isActive && 'bg-primary/10 font-medium text-primary hover:bg-primary/10 hover:text-primary',
                )
              }
            >
              {({ isActive }) => (
                <>
                  <span
                    className={cn(
                      'absolute top-1/2 left-0 h-4 w-0.5 -translate-y-1/2 rounded-full bg-primary transition-opacity',
                      isActive ? 'opacity-100' : 'opacity-0',
                    )}
                  />
                  <item.icon className="size-4 shrink-0" />
                  <span className="flex flex-col">
                    <span>{item.label}</span>
                    <span className="hidden text-[10px] text-muted-foreground/70 lg:inline">
                      {item.desc}
                    </span>
                  </span>
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* 用户信息 */}
        <div className="border-t pt-2 lg:mt-auto">
          <UserMenu />
        </div>
      </aside>
      <main className="min-w-0 flex-1">
        <Outlet />
      </main>
    </div>
  )
}

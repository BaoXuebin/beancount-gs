import { useState } from 'react'
import { NavLink, Outlet, useParams } from 'react-router-dom'
import {
  BarChart3,
  CalendarDays,
  FileCode,
  LayoutDashboard,
  ListOrdered,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
  Sparkles,
  Upload,
  Wallet,
} from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { UserMenu } from '@/components/UserMenu'
import { Breadcrumb } from '@/components/Breadcrumb'
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

const sidebarCollapsedKey = 'beancount-gs.sidebar.collapsed'

function initialSidebarCollapsed(): boolean {
  try {
    return localStorage.getItem(sidebarCollapsedKey) === '1'
  } catch {
    return false
  }
}

export function LedgerLayout() {
  const { ledgerId = '' } = useParams()
  const ledger = useFetch<Ledger>(`/ledgers/${ledgerId}`)
  const [collapsed, setCollapsed] = useState(initialSidebarCollapsed)

  const toggleSidebar = () => {
    setCollapsed((prev) => {
      const next = !prev
      try {
        localStorage.setItem(sidebarCollapsedKey, next ? '1' : '0')
      } catch {
        // 忽略存储失败（如隐私模式）
      }
      return next
    })
  }

  return (
    <div className="flex min-h-screen flex-col lg:h-screen lg:overflow-hidden">
      {/* 顶部栏：左侧账本信息 + 返回账本列表，右侧用户 */}
      <header className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
        <div className="flex w-full items-center justify-between gap-3 px-4 py-2.5">
          <div className="flex min-w-0 items-center gap-2">
            <Breadcrumb
              items={[
                { label: '工作区', to: '/workspaces' },
                { label: '账本', to: '/ledgers' },
                { label: ledger.data?.name ?? (ledger.loading ? '加载中…' : '账本') },
              ]}
            />
            {ledger.loading ? (
              <Skeleton className="h-3 w-24 shrink-0" />
            ) : ledger.data ? (
              <span className="hidden shrink-0 border-l pl-2 text-xs text-muted-foreground sm:inline">
                本位币 {ledger.data.operating_currency} · 修订 #{ledger.data.revision}
              </span>
            ) : null}
          </div>
          <UserMenu align="end" />
        </div>
      </header>

      <div className="flex w-full flex-1 flex-col gap-6 px-4 py-6 lg:min-h-0 lg:flex-row lg:overflow-hidden">
        <aside className={cn('w-full shrink-0', collapsed ? 'lg:w-14' : 'lg:w-52')}>
          <div className={cn('hidden lg:flex', collapsed ? 'justify-center' : 'justify-end')}>
            <button
              type="button"
              onClick={toggleSidebar}
              aria-label={collapsed ? '展开侧栏' : '折叠侧栏'}
              title={collapsed ? '展开侧栏' : '折叠侧栏'}
              className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              {collapsed ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
            </button>
          </div>
          <nav
            className={cn(
              'flex flex-wrap gap-1 lg:flex-col',
              !collapsed && 'lg:pr-1',
            )}
          >
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={`/ledgers/${ledgerId}/${item.to}`}
                title={`${item.label} · ${item.desc}`}
                className={({ isActive }) =>
                  cn(
                    'relative flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground',
                    isActive && 'bg-primary/10 font-medium text-primary hover:bg-primary/10 hover:text-primary',
                    collapsed && 'lg:justify-center lg:px-0',
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
                    {!collapsed && (
                      <span className="flex flex-col">
                        <span>{item.label}</span>
                        <span className="hidden text-[10px] text-muted-foreground/70 lg:inline">
                          {item.desc}
                        </span>
                      </span>
                    )}
                  </>
                )}
              </NavLink>
            ))}
          </nav>
        </aside>
        <main className="min-w-0 flex-1 lg:overflow-y-auto lg:pr-1">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

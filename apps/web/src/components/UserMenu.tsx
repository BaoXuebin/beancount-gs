import { LogOut } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { request } from '@/api/client'
import { cn } from '@/lib/utils'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

function initials(name: string): string {
  return name.trim().charAt(0).toUpperCase() || '?'
}

export function UserMenu({
  align = 'start',
  compact = false,
}: {
  align?: 'start' | 'end'
  compact?: boolean
}) {
  const { user } = useAuth()
  const name = user?.display_name || user?.github_login || '用户'

  const logout = async () => {
    try {
      await request('/auth/logout', { method: 'POST' })
    } catch {
      // 忽略退出登录错误，仍然跳转
    }
    window.location.href = '/login'
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            className={cn(
              'flex items-center gap-2 rounded-lg text-left transition-colors hover:bg-accent data-popup-open:bg-accent',
              compact ? 'px-1 py-1' : 'w-full px-2 py-2',
            )}
            title={name}
          >
            <Avatar size="sm">
              <AvatarFallback>{initials(name)}</AvatarFallback>
            </Avatar>
            {!compact && (
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="truncate text-sm font-medium">{name}</span>
                <span className="truncate text-[10px] text-muted-foreground">
                  {user?.github_login ?? ''}
                </span>
              </span>
            )}
          </button>
        }
      />
      <DropdownMenuContent align={align} side="top" sideOffset={8} className="w-60">
        <DropdownMenuLabel>用户信息</DropdownMenuLabel>
        <div className="flex items-center gap-2 px-1.5 py-1.5">
          <Avatar>
            <AvatarFallback>{initials(name)}</AvatarFallback>
          </Avatar>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{name}</p>
            <p className="truncate text-xs text-muted-foreground">@{user?.github_login}</p>
          </div>
        </div>
        {user?.email && (
          <p className="px-1.5 pb-1 text-xs text-muted-foreground">{user.email}</p>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onClick={logout}>
          <LogOut /> 退出登录
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

import { AtSign, LogOut, Mail } from 'lucide-react'
import { useRef, useState } from 'react'
import { useAuth } from '@/auth/AuthContext'
import { request } from '@/api/client'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

function initials(name: string): string {
  return name.trim().charAt(0).toUpperCase() || '?'
}

export function UserMenu({ align = 'start' }: { align?: 'start' | 'end' }) {
  const { user } = useAuth()
  const name = user?.display_name || user?.github_login || '用户'
  const [open, setOpen] = useState(false)
  const closeTimer = useRef<number | undefined>(undefined)

  const logout = async () => {
    try {
      await request('/auth/logout', { method: 'POST' })
    } catch {
      // 忽略退出登录错误，仍然跳转
    }
    window.location.href = '/login'
  }

  const openMenu = () => {
    if (closeTimer.current !== undefined) window.clearTimeout(closeTimer.current)
    setOpen(true)
  }

  const closeMenu = () => {
    closeTimer.current = window.setTimeout(() => setOpen(false), 150)
  }

  return (
    <div onMouseEnter={openMenu} onMouseLeave={closeMenu}>
      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger
          render={
            <button
              type="button"
              className="flex shrink-0 items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-accent data-popup-open:bg-accent"
              title={name}
            >
            <Avatar size="sm">
              <AvatarFallback>{initials(name)}</AvatarFallback>
            </Avatar>
            <span className="max-w-[120px] truncate text-sm font-medium">{name}</span>
            </button>
          }
        />
        <DropdownMenuContent
          align={align}
          side="bottom"
          sideOffset={6}
          className="w-60"
          onMouseEnter={openMenu}
          onMouseLeave={closeMenu}
        >
          <DropdownMenuGroup>
            <DropdownMenuLabel>用户信息</DropdownMenuLabel>
          </DropdownMenuGroup>
          <div className="flex items-center gap-2 px-1.5 py-1.5">
            <Avatar>
              <AvatarFallback>{initials(name)}</AvatarFallback>
            </Avatar>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{name}</p>
          </div>
        </div>
        <div className="flex flex-col gap-1.5 px-1.5 pb-1.5">
          <div className="flex items-center gap-1.5 text-xs">
            <AtSign className="size-3.5 shrink-0 text-muted-foreground" />
            <span className="w-10 shrink-0 text-muted-foreground">账户名</span>
            <span className="truncate">@{user?.github_login}</span>
          </div>
          {user?.email && (
            <div className="flex items-center gap-1.5 text-xs">
              <Mail className="size-3.5 shrink-0 text-muted-foreground" />
              <span className="w-10 shrink-0 text-muted-foreground">邮箱</span>
              <span className="truncate">{user.email}</span>
            </div>
          )}
        </div>
        <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={logout}>
            <LogOut /> 退出登录
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

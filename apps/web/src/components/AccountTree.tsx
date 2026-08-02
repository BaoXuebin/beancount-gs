import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import type { Account } from '@/api/types'
import { cn } from '@/lib/utils'

interface AccountNode {
  name: string
  path: string
  account?: Account
  children: AccountNode[]
}

function buildTree(accounts: Account[]): AccountNode[] {
  const roots: AccountNode[] = []
  for (const acc of accounts) {
    const segments = acc.account.split(':').filter(Boolean)
    let level = roots
    let path = ''
    for (let i = 0; i < segments.length; i++) {
      path = path ? `${path}:${segments[i]}` : segments[i]
      let node = level.find((n) => n.name === segments[i])
      if (!node) {
        node = { name: segments[i], path, children: [] }
        level.push(node)
      }
      if (i === segments.length - 1) {
        node.account = acc
      }
      level = node.children
    }
  }
  const sortByName = (nodes: AccountNode[]) => {
    nodes.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
    nodes.forEach((n) => sortByName(n.children))
  }
  sortByName(roots)
  return roots
}

function leafAmount(account: Account): string {
  if (account.market_number) return `${account.market_number} ${account.market_currency ?? ''}`.trim()
  if (account.positions?.length) {
    return account.positions.map((p) => `${p.number} ${p.currency}`).join(' · ')
  }
  return ''
}

export function AccountTree({
  accounts,
  ledgerId,
}: {
  accounts: Account[]
  ledgerId: string
}) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())
  const roots = buildTree(accounts)

  const toggle = (path: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  const renderNode = (node: AccountNode, depth: number) => {
    const isCollapsed = collapsed.has(node.path)
    const hasChildren = node.children.length > 0
    return (
      <div key={node.path}>
        <div
          className={cn(
            'flex min-w-0 items-center gap-1.5 rounded-md py-1.5 pr-2 text-sm',
            depth > 0 && 'hover:bg-accent/50',
          )}
          style={{ paddingLeft: depth * 18 }}
        >
          {hasChildren ? (
            <button
              type="button"
              className="shrink-0 rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
              onClick={() => toggle(node.path)}
            >
              {isCollapsed ? (
                <ChevronRight className="size-3.5" />
              ) : (
                <ChevronDown className="size-3.5" />
              )}
            </button>
          ) : (
            <span className="w-4 shrink-0" />
          )}

          {node.account ? (
            <Link
              to={`/ledgers/${ledgerId}/accounts/${encodeURIComponent(node.account.account)}`}
              className="flex min-w-0 flex-1 items-center justify-between gap-2 hover:text-primary"
            >
              <span className="min-w-0">
                <span className="truncate">{node.name}</span>
                {node.children.length > 0 && (
                  <Badge variant="outline" className="ml-1.5 align-middle text-[10px]">
                    {node.children.length}
                  </Badge>
                )}
              </span>
              <span className="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">
                {leafAmount(node.account)}
              </span>
            </Link>
          ) : (
            <button
              type="button"
              className="flex min-w-0 flex-1 items-center justify-between gap-2 text-left font-medium"
              onClick={() => hasChildren && toggle(node.path)}
            >
              <span className="truncate">{node.name}</span>
              {node.children.length > 0 && (
                <Badge variant="outline" className="shrink-0 text-[10px]">
                  {node.children.length}
                </Badge>
              )}
            </button>
          )}
        </div>
        {hasChildren && !isCollapsed && (
          <div>{node.children.map((child) => renderNode(child, depth + 1))}</div>
        )}
      </div>
    )
  }

  if (roots.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-muted-foreground">该类型下暂无账户</p>
    )
  }

  return <div className="flex flex-col">{roots.map((node) => renderNode(node, 0))}</div>
}

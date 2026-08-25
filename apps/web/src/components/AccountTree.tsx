import { useMemo, useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  ChevronsDown,
  ChevronsUp,
  CircleDollarSign,
  Eye,
  Plus,
  Search,
  X,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import type { Account } from '@/api/types'
import {
  ACCOUNT_TYPE_META,
  AccountType,
  accountCurrencies,
  accountIcon,
  formatNumber,
} from '@/lib/accountMeta'
import { cn } from '@/lib/utils'

interface AccountNode {
  name: string
  path: string
  account?: Account
  children: AccountNode[]
  /** 子树内账户（叶子）总数 */
  leafCount: number
  /** 币种 → 子树叶子市值之和（按本位币折算后的 market_number） */
  balances: Map<string, number>
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
        node = { name: segments[i], path, children: [], leafCount: 0, balances: new Map() }
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
  aggregate(roots)
  return roots
}

function aggregate(nodes: AccountNode[]): void {
  for (const node of nodes) {
    let leafCount = 0
    const balances = new Map<string, number>()
    if (node.account) {
      leafCount += 1
      for (const [cur, v] of leafBalances(node.account)) {
        balances.set(cur, (balances.get(cur) ?? 0) + v)
      }
    }
    for (const child of node.children) {
      aggregate([child])
      leafCount += child.leafCount
      for (const [cur, v] of child.balances) {
        balances.set(cur, (balances.get(cur) ?? 0) + v)
      }
    }
    node.leafCount = leafCount
    node.balances = balances
  }
}

function formatBalances(balances: Map<string, number>): string {
  if (balances.size === 0) return ''
  return Array.from(balances.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([cur, v]) => `${formatNumber(v)} ${cur}`)
    .join(' · ')
}

/**
 * 账户余额（币种 → 数量）：
 * 优先取后端折算好的市值（本位币）；没有市值时按币种汇总 positions（原生单位）。
 * 二选一，避免同一账户的市值与持仓重复计入。
 */
function leafBalances(account: Account): Map<string, number> {
  const balances = new Map<string, number>()
  if (account.market_number && account.market_currency) {
    const n = Number(account.market_number)
    if (Number.isFinite(n)) balances.set(account.market_currency, n)
    return balances
  }
  for (const p of account.positions ?? []) {
    if (!p.currency || p.number == null) continue
    const n = Number(p.number)
    if (!Number.isFinite(n)) continue
    balances.set(p.currency, (balances.get(p.currency) ?? 0) + n)
  }
  return balances
}

function leafAmount(account: Account): string {
  return formatBalances(leafBalances(account))
}

/**
 * 折叠“纯分组前缀”的单子节点链：如 Fixed → House → 中海房产，
 * 把 Fixed、House 压成面包屑，只渲染最终节点一行。
 * 遇到真实账户（account 存在）或分支节点（children>1）即停止。
 */
function foldPrefix(node: AccountNode): { head: AccountNode; crumbs: string[] } {
  const crumbs: string[] = []
  let cur = node
  while (cur.account == null && cur.children.length === 1) {
    crumbs.push(cur.name)
    cur = cur.children[0]
  }
  return { head: cur, crumbs }
}

/** 高亮命中的子串（大小写不敏感）。 */
function Highlight({ text, query }: { text: string; query: string }) {
  if (!query) return <>{text}</>
  const idx = text.toLowerCase().indexOf(query)
  if (idx === -1) return <>{text}</>
  return (
    <>
      {text.slice(0, idx)}
      <mark className="rounded-sm bg-primary/25 text-foreground">
        {text.slice(idx, idx + query.length)}
      </mark>
      {text.slice(idx + query.length)}
    </>
  )
}

interface AccountTreeProps {
  accounts: Account[]
  onSelect?: (account: Account) => void
  onAddChild?: (parentPath: string) => void
}

export function AccountTree({ accounts, onSelect, onAddChild }: AccountTreeProps) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())
  const [query, setQuery] = useState('')
  const [showAmounts, setShowAmounts] = useState(true)
  const q = query.trim().toLowerCase()

  const { roots, matchCount, visiblePaths } = useMemo(() => {
    const built = buildTree(accounts)
    let matches = 0
    const visible = new Set<string>()
    if (q) {
      const mark = (nodes: AccountNode[], ancestors: string[]) => {
        for (const n of nodes) {
          if (n.account && n.path.toLowerCase().includes(q)) {
            matches++
            for (const a of ancestors) visible.add(a)
            visible.add(n.path)
          }
          mark(n.children, [...ancestors, n.path])
        }
      }
      mark(built, [])
    }
    return { roots: built, matchCount: matches, visiblePaths: q ? visible : null }
  }, [accounts, q])

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

  const expandAll = () => setCollapsed(new Set())
  const collapseAll = () => {
    const all: string[] = []
    const collect = (nodes: AccountNode[]) => {
      for (const n of nodes) {
        // 只收起真正可展开的分支节点（单子节点链已被折叠）
        if (n.children.length > 1) all.push(n.path)
        collect(n.children)
      }
    }
    collect(roots)
    setCollapsed(new Set(all))
  }

  const renderNode = (node: AccountNode, depth: number): React.ReactNode => {
    const { head, crumbs } = foldPrefix(node)
    const hasChildren = head.children.length > 0
    // 搜索时只显示“含匹配祖先链”的节点
    if (visiblePaths && !visiblePaths.has(node.path)) return null
    const isExpanded = q !== '' || !collapsed.has(head.path)
    const type = ((head.account?.type ?? node.path.split(':')[0]) as AccountType) || 'Assets'
    const meta = ACCOUNT_TYPE_META[type] ?? ACCOUNT_TYPE_META.Assets
    const isLeaf = !hasChildren
    const Icon =
      depth === 0 && crumbs.length === 0 && hasChildren
        ? meta.icon
        : accountIcon(head.name, type, hasChildren)
    const balancesText = isLeaf ? '' : formatBalances(head.balances)
    const amount = isLeaf && head.account ? leafAmount(head.account) : balancesText
    const currencies = head.account ? accountCurrencies(head.account) : []

    return (
      <div key={node.path} className="min-w-0">
        <div
          className={cn(
            'group flex min-w-0 items-start gap-1.5 rounded-md py-1.5 pr-1.5 text-sm',
            depth > 0 && 'hover:bg-accent/50',
            isLeaf && head.account && 'cursor-pointer',
          )}
          role={isLeaf && head.account ? 'button' : undefined}
          tabIndex={isLeaf && head.account ? 0 : undefined}
          onClick={() => {
            if (isLeaf && head.account) {
              onSelect?.(head.account)
            } else if (hasChildren) {
              toggle(head.path)
            }
          }}
          onKeyDown={(e) => {
            if ((e.key === 'Enter' || e.key === ' ') && isLeaf && head.account) {
              e.preventDefault()
              onSelect?.(head.account)
            }
          }}
        >
          {hasChildren ? (
            <button
              type="button"
              aria-label={isExpanded ? '收起' : '展开'}
              className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
              onClick={(e) => {
                e.stopPropagation()
                toggle(head.path)
              }}
            >
              {isExpanded ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
            </button>
          ) : (
            <span className="mt-0.5 h-4 w-4 shrink-0" />
          )}

          <Icon className={cn('mt-0.5 size-4 shrink-0', meta.iconClass)} />

          <div className="min-w-0 flex-1">
            <div className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5">
              {crumbs.map((crumb) => (
                <span key={crumb} className="flex min-w-0 items-center gap-1.5">
                  <span className="whitespace-nowrap text-muted-foreground/80">
                    <Highlight text={crumb} query={q} />
                  </span>
                  <span className="text-muted-foreground/50">›</span>
                </span>
              ))}
              <span
                className={cn(
                  'truncate font-medium',
                  head.account?.status === 'closed' && 'text-muted-foreground line-through decoration-muted-foreground/50',
                )}
              >
                <Highlight text={head.name} query={q} />
              </span>
              {hasChildren && (
                <Badge variant="outline" className="shrink-0 text-[10px] text-muted-foreground">
                  {head.leafCount}
                </Badge>
              )}
              {head.account?.status === 'closed' && (
                <Badge variant="secondary" className="shrink-0 text-[10px]">
                  已关闭
                </Badge>
              )}
            </div>

            <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5">
              {isLeaf && currencies.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {currencies.map((cur) => (
                    <span
                      key={cur}
                      className="rounded border border-border bg-muted/40 px-1 py-px font-mono text-[10px] text-muted-foreground"
                    >
                      {cur}
                    </span>
                  ))}
                </div>
              )}
              {!isLeaf && head.leafCount > 0 && (
                <span className="text-xs text-muted-foreground">{head.leafCount} 个账户</span>
              )}
              {isLeaf && head.account?.opened_on && (
                <span className="text-xs text-muted-foreground">开户 {head.account.opened_on}</span>
              )}
            </div>
          </div>

          {showAmounts && amount && (
            <div className="shrink-0 self-center whitespace-nowrap font-mono text-xs tabular-nums text-muted-foreground">
              {amount}
            </div>
          )}

          <div className="hidden shrink-0 items-center gap-0.5 self-center group-hover:flex">
            {head.account && (
              <button
                type="button"
                title="查看详情"
                className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
                onClick={(e) => {
                  e.stopPropagation()
                  onSelect?.(head.account!)
                }}
              >
                <Eye className="size-3.5" />
              </button>
            )}
            <button
              type="button"
              title="新增子账户"
              className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
              onClick={(e) => {
                e.stopPropagation()
                onAddChild?.(head.path)
              }}
            >
              <Plus className="size-3.5" />
            </button>
          </div>
        </div>

        {hasChildren && isExpanded && (
          <div className="ml-[21px] border-l border-border/70 pl-3">
            {head.children.map((child) => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    )
  }

  if (roots.length === 0) {
    return (
      <div className="py-10 text-center text-sm text-muted-foreground">
        <p>该类型下暂无账户</p>
      </div>
    )
  }

  return (
    <div className="min-w-0">
      <div className="mb-3 flex items-center gap-2">
        <div className="relative min-w-0 flex-1">
          <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索当前类型下的账户"
            className="pl-8 pr-8"
          />
          {query && (
            <button
              type="button"
              aria-label="清除搜索"
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
              onClick={() => setQuery('')}
            >
              <X className="size-3.5" />
            </button>
          )}
        </div>
        {q && (
          <span className="shrink-0 text-xs text-muted-foreground">{matchCount} 个匹配</span>
        )}
        <button
          type="button"
          title={showAmounts ? '隐藏金额' : '显示金额'}
          className={cn(
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-md border text-muted-foreground hover:bg-accent hover:text-foreground',
            showAmounts
              ? 'border-primary/40 bg-primary/10 text-primary'
              : 'border-input',
          )}
          onClick={() => setShowAmounts((v) => !v)}
        >
          <CircleDollarSign className="size-4" />
        </button>
        <button
          type="button"
          title="全部展开"
          className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-input text-muted-foreground hover:bg-accent hover:text-foreground"
          onClick={expandAll}
        >
          <ChevronsDown className="size-4" />
        </button>
        <button
          type="button"
          title="全部收起"
          className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-input text-muted-foreground hover:bg-accent hover:text-foreground"
          onClick={collapseAll}
        >
          <ChevronsUp className="size-4" />
        </button>
      </div>

      {q && matchCount === 0 ? (
        <div className="py-10 text-center text-sm text-muted-foreground">
          没有匹配「{query}」的账户
        </div>
      ) : (
        <div className="flex min-w-0 flex-col">{roots.map((node) => renderNode(node, 0))}</div>
      )}
    </div>
  )
}

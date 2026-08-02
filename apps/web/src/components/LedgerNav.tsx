import { Link, useParams } from 'react-router-dom'

const items = [
  { to: 'transactions', label: '交易' },
  { to: 'stats', label: '统计' },
  { to: 'import', label: '导入' },
]

export function LedgerNav() {
  const { ledgerId } = useParams()
  return (
    <nav className="mt-4 flex flex-wrap gap-2">
      {items.map((item) => (
        <Link
          key={item.to}
          to={`/ledgers/${ledgerId}/${item.to}`}
          className="rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:border-primary hover:text-foreground"
        >
          {item.label}
        </Link>
      ))}
    </nav>
  )
}

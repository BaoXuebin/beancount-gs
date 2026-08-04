import { useMemo, useState } from 'react'
import { Input } from '@/components/ui/input'
import type { Account } from '@/api/types'
import { accountCurrencies } from '@/lib/accountMeta'
import { cn } from '@/lib/utils'

interface AccountPickerProps {
  value: string
  accounts: Account[]
  /** 选中账户时回调：账户名 + 该账户的主币种（用于自动填充） */
  onChange: (account: string, currency?: string) => void
  placeholder?: string
}

/** 账户模糊选择：输入即过滤（全路径或末段），点击选中后自动带出默认币种。 */
export function AccountPicker({ value, accounts, onChange, placeholder }: AccountPickerProps) {
  const [open, setOpen] = useState(false)

  const matches = useMemo(() => {
    const q = value.trim().toLowerCase()
    if (!q) return []
    return accounts
      .filter((a) =>
        a.account.toLowerCase().includes(q) ||
        (a.account.split(':').pop() ?? '').toLowerCase().includes(q),
      )
      .slice(0, 10)
  }, [accounts, value])

  return (
    <div className="relative">
      <Input
        value={value}
        placeholder={placeholder}
        className="font-mono"
        onChange={(e) => onChange(e.target.value)}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && matches.length === 1) {
            const m = matches[0]
            onChange(m.account, accountCurrencies(m)[0])
            setOpen(false)
          }
        }}
      />
      {open && matches.length > 0 && (
        <div className="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-md border bg-background py-1 shadow-lg">
          {matches.map((a) => (
            <button
              key={a.account}
              type="button"
              className={cn(
                'block w-full truncate px-2.5 py-1 text-left font-mono text-xs hover:bg-accent',
                a.account === value && 'bg-accent/60',
              )}
              onMouseDown={(e) => {
                e.preventDefault()
                onChange(a.account, accountCurrencies(a)[0])
                setOpen(false)
              }}
            >
              {a.account}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

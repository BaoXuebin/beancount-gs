import type { LucideIcon } from 'lucide-react'
import {
  BadgeDollarSign,
  BookOpen,
  Car,
  CandlestickChart,
  Coffee,
  CreditCard,
  Dumbbell,
  Folder,
  Gamepad2,
  Gift,
  GraduationCap,
  Home,
  Landmark,
  Music,
  Phone,
  PieChart,
  Plane,
  PlugZap,
  Receipt,
  Shirt,
  ShoppingBag,
  Smartphone,
  Stethoscope,
  TrendingUp,
  Utensils,
  Wallet,
} from 'lucide-react'
import type { Account } from '@/api/types'

export type AccountType = 'Assets' | 'Liabilities' | 'Income' | 'Expenses' | 'Equity'


export interface AccountTypeMeta {
  label: string
  icon: LucideIcon
  /** 图标颜色（含 dark 模式） */
  iconClass: string
  /** 汇总条数字颜色 */
  chipClass: string
}

export const ACCOUNT_TYPE_META: Record<AccountType, AccountTypeMeta> = {
  Assets: {
    label: '资产',
    icon: Wallet,
    iconClass: 'text-sky-600 dark:text-sky-400',
    chipClass: 'text-sky-700 dark:text-sky-300',
  },
  Liabilities: {
    label: '负债',
    icon: CreditCard,
    iconClass: 'text-amber-600 dark:text-amber-400',
    chipClass: 'text-amber-700 dark:text-amber-300',
  },
  Income: {
    label: '收入',
    icon: TrendingUp,
    iconClass: 'text-emerald-600 dark:text-emerald-400',
    chipClass: 'text-emerald-700 dark:text-emerald-300',
  },
  Expenses: {
    label: '费用',
    icon: Receipt,
    iconClass: 'text-rose-600 dark:text-rose-400',
    chipClass: 'text-rose-700 dark:text-rose-300',
  },
  Equity: {
    label: '权益',
    icon: PieChart,
    iconClass: 'text-violet-600 dark:text-violet-400',
    chipClass: 'text-violet-700 dark:text-violet-300',
  },
}

const KEYWORD_ICONS: Array<[RegExp, LucideIcon]> = [
  [/银行|储蓄|bank/i, Landmark],
  [/现金|cash/i, Wallet],
  [/支付宝|微信|支付|pay/i, Smartphone],
  [/信用卡|贷记/i, CreditCard],
  [/证券|股票|基金|投资|理财/i, CandlestickChart],
  [/工资|薪|奖金|bonus/i, BadgeDollarSign],
  [/餐饮|餐|饭|外卖|食|eat|food/i, Utensils],
  [/交通|出行|打车|地铁|公交|加油|车/i, Car],
  [/医疗|药|医院|健康/i, Stethoscope],
  [/教育|学费|培训/i, GraduationCap],
  [/娱乐|游戏|电影|game/i, Gamepad2],
  [/通讯|话费|手机|流量|phone/i, Phone],
  [/水电|燃气|物业|电费|水费|煤气/i, PlugZap],
  [/房租|住房|房贷|租金|home/i, Home],
  [/购物|电商|淘宝|京东|买|shop/i, ShoppingBag],
  [/衣服|服饰|shirt/i, Shirt],
  [/礼物|红包|gift/i, Gift],
  [/旅行|旅游|机票|酒店|flight/i, Plane],
  [/音乐|视频|影音/i, Music],
  [/健身|运动|gym/i, Dumbbell],
  [/咖啡|饮品|coffee/i, Coffee],
  [/图书|书籍|读书/i, BookOpen],
]

/** 按账户段名推断图标；分组节点用文件夹，根类型节点由调用方传入类型图标。 */
export function accountIcon(name: string, type: AccountType, isGroup: boolean): LucideIcon {
  if (isGroup) return Folder
  for (const [re, icon] of KEYWORD_ICONS) {
    if (re.test(name)) return icon
  }
  return ACCOUNT_TYPE_META[type].icon
}

/** 账户的币种列表（去重、保序）：优先 commodities，否则按逗号拆分 currency。 */
export function accountCurrencies(a: Account): string[] {
  const list = (a.commodities ?? []).map((c) => c.currency).filter(Boolean)
  if (list.length === 0 && a.currency) {
    list.push(...a.currency.split(/[,，]/).map((s) => s.trim()).filter(Boolean))
  }
  return Array.from(new Set(list))
}

/** 数值格式化：千分位 + 最多 2 位小数。 */
export function formatNumber(n: number): string {
  return n.toLocaleString('zh-CN', { maximumFractionDigits: 2 })
}
import type { components } from './schema'

export type Ledger = components['schemas']['Ledger']
export type Team = components['schemas']['Team']
export type Transaction = components['schemas']['Transaction']
export type Posting = components['schemas']['Posting']

export interface TransactionListResponse {
  items: Transaction[]
  total: number
}

export interface StatsPoint {
  date?: string
  amount?: string
  operating_currency?: string
}

export interface Insight {
  type: string
  message: string
}

export interface ImportRow {
  index: number
  date: string
  payee?: string
  narration?: string
  number: string
  currency?: string
  suggested_account?: string
  confidence?: number
  status?: string
}

export interface ImportResult {
  created: number
  failed: { index: number; reason: string }[]
}

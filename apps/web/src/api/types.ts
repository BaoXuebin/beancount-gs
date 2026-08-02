import type { components } from './schema'

export type Ledger = components['schemas']['Ledger']
export type Team = components['schemas']['Team']
export type Transaction = components['schemas']['Transaction']
export type Posting = components['schemas']['Posting']

export interface TransactionListResponse {
  items: Transaction[]
  total: number
}

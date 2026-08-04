import type { components } from './schema'

export type Ledger = components['schemas']['Ledger']
export type LedgerCreate = components['schemas']['LedgerCreate']
export type BackupImportResult = components['schemas']['BackupImportResult']
export type Team = components['schemas']['Team']
export type TeamCreate = components['schemas']['TeamCreate']
export type User = components['schemas']['User']
export type Transaction = components['schemas']['Transaction']
export type TransactionCreate = components['schemas']['TransactionCreate']
export type Posting = components['schemas']['Posting']
export type Amount = components['schemas']['Amount']
export type Cost = components['schemas']['Cost']
export type Account = components['schemas']['Account']
export type AccountOpen = components['schemas']['AccountOpen']
export type AccountDetail = components['schemas']['AccountDetail']
export type AccountTypeMapping = components['schemas']['AccountTypeMapping']
export type Event = components['schemas']['Event']
export type Currency = components['schemas']['Currency']
export type TransactionTemplate = components['schemas']['TransactionTemplate']
export type ApiKey = components['schemas']['ApiKey']
export type AuditLog = components['schemas']['AuditLog']
export type Membership = components['schemas']['Membership']
export type StatsPayee = components['schemas']['StatsPayee']
export type AiRecordResult = components['schemas']['AiRecordResult']

export type AiAccountSuggestion = components['schemas']['AiAccountsResult']['accounts'][number]

export interface AiAccountsResult {
  accounts: AiAccountSuggestion[]
  notes?: string
}

export interface AiRecordBatchResult {
  drafts: Transaction[]
  notes?: string
}

export interface AccountOpenBatchResult {
  created: Account[]
  skipped: { account: string; reason: string }[]
}
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

export interface InsightsResponse {
  insights: Insight[]
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

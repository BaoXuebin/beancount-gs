import { BrowserRouter, Link, Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/auth/AuthContext'
import { Button } from '@/components/ui/button'
import { request } from '@/api/client'
import { LedgerLayout } from '@/components/LedgerLayout'
import { AccountDetailPage } from '@/pages/AccountDetailPage'
import { AccountsPage } from '@/pages/AccountsPage'
import { AccountTypesPage } from '@/pages/AccountTypesPage'
import { AIAssistantPage } from '@/pages/AIAssistantPage'
import { AISettingsPage } from '@/pages/AISettingsPage'
import { ApiDocsPage } from '@/pages/ApiDocsPage'
import { AuditPage } from '@/pages/AuditPage'
import { CurrenciesPage } from '@/pages/CurrenciesPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { ErrorPage } from '@/pages/ErrorPage'
import { EventsPage } from '@/pages/EventsPage'
import { ExportPage } from '@/pages/ExportPage'
import { ImportPage } from '@/pages/ImportPage'
import { LedgersPage } from '@/pages/LedgersPage'
import { LoginPage } from '@/pages/LoginPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { SourceFilesPage } from '@/pages/SourceFilesPage'
import { StatsPage } from '@/pages/StatsPage'
import { TemplatesPage } from '@/pages/TemplatesPage'
import { TransactionDetailPage } from '@/pages/TransactionDetailPage'
import { TransactionEditPage } from '@/pages/TransactionEditPage'
import { TransactionsPage } from '@/pages/TransactionsPage'
import { WorkspacesPage } from '@/pages/WorkspacesPage'
import { IntegrationsPage } from '@/pages/IntegrationsPage'

function AppLayout() {
  const { user } = useAuth()
  const logout = async () => {
    try {
      await request('/auth/logout', { method: 'POST' })
    } catch {
      // 忽略登出错误
    }
    window.location.href = '/login'
  }

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <Link to="/workspaces" className="font-medium">
            beancount-gs
          </Link>
          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-muted-foreground sm:inline">
              {user?.display_name ?? user?.github_login}
            </span>
            <Button variant="ghost" size="sm" onClick={logout}>
              退出登录
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}

function FullScreenLoader() {
  return (
    <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
      加载中…
    </div>
  )
}

function HomeRedirect() {
  const { loading, loggedIn } = useAuth()
  if (loading) return <FullScreenLoader />
  return <Navigate to={loggedIn ? '/workspaces' : '/login'} replace />
}

function RequireAuth() {
  const { loading, loggedIn } = useAuth()
  if (loading) return <FullScreenLoader />
  if (!loggedIn) return <Navigate to="/login" replace />
  return <Outlet />
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<HomeRedirect />} />
          <Route path="/login" element={<LoginPage />} />
          <Route element={<RequireAuth />}>
            <Route element={<AppLayout />}>
              <Route path="/workspaces" element={<WorkspacesPage />} />
              <Route path="/ledgers" element={<LedgersPage />} />
              <Route path="/ledgers/:ledgerId" element={<Navigate to="dashboard" replace />} />
              <Route path="/ledgers/:ledgerId/*" element={<LedgerLayout />}>
                <Route path="dashboard" element={<DashboardPage />} />
                <Route path="transactions" element={<TransactionsPage />} />
                <Route path="transactions/new" element={<TransactionEditPage />} />
                <Route path="transactions/:transactionId" element={<TransactionDetailPage />} />
                <Route path="transactions/:transactionId/edit" element={<TransactionEditPage />} />
                <Route path="accounts" element={<AccountsPage />} />
                <Route path="accounts/:account" element={<AccountDetailPage />} />
                <Route path="account-types" element={<AccountTypesPage />} />
                <Route path="currencies" element={<CurrenciesPage />} />
                <Route path="stats" element={<StatsPage />} />
                <Route path="import" element={<ImportPage />} />
                <Route path="events" element={<EventsPage />} />
                <Route path="source" element={<SourceFilesPage />} />
                <Route path="templates" element={<TemplatesPage />} />
                <Route path="ai" element={<AIAssistantPage />} />
                <Route path="settings" element={<SettingsPage />} />
                <Route path="settings/ai" element={<AISettingsPage />} />
                <Route path="settings/integrations" element={<IntegrationsPage />} />
                <Route path="settings/api-docs" element={<ApiDocsPage />} />
                <Route path="settings/audit" element={<AuditPage />} />
                <Route path="settings/export" element={<ExportPage />} />
                <Route path="*" element={<ErrorPage code={404} />} />
              </Route>
            </Route>
          </Route>
          <Route path="/403" element={<ErrorPage code={403} />} />
          <Route path="*" element={<ErrorPage code={404} />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}

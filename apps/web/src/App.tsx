import { BrowserRouter, Link, Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/auth/AuthContext'
import { Button } from '@/components/ui/button'
import { request } from '@/api/client'
import { LedgersPage } from '@/pages/LedgersPage'
import { LoginPage } from '@/pages/LoginPage'
import { ImportPage } from '@/pages/ImportPage'
import { StatsPage } from '@/pages/StatsPage'
import { TransactionsPage } from '@/pages/TransactionsPage'
import { WorkspacesPage } from '@/pages/WorkspacesPage'

function AppLayout() {
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
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <Link to="/workspaces" className="font-medium">
            beancount-gs
          </Link>
          <Button variant="ghost" size="sm" onClick={logout}>
            退出登录
          </Button>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8">
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
              <Route path="/ledgers/:ledgerId/transactions" element={<TransactionsPage />} />
              <Route path="/ledgers/:ledgerId/stats" element={<StatsPage />} />
              <Route path="/ledgers/:ledgerId/import" element={<ImportPage />} />
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}

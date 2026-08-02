import { BrowserRouter, Link, Navigate, Outlet, Route, Routes, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { request } from '@/api/client'
import { LedgersPage } from '@/pages/LedgersPage'
import { LoginPage } from '@/pages/LoginPage'
import { ImportPage } from '@/pages/ImportPage'
import { StatsPage } from '@/pages/StatsPage'
import { TransactionsPage } from '@/pages/TransactionsPage'
import { WorkspacesPage } from '@/pages/WorkspacesPage'

function AppLayout() {
  const navigate = useNavigate()
  const logout = async () => {
    try {
      await request('/auth/logout', { method: 'POST' })
    } catch {
      // 忽略登出错误
    }
    navigate('/login')
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

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<AppLayout />}>
          <Route path="/workspaces" element={<WorkspacesPage />} />
          <Route path="/ledgers" element={<LedgersPage />} />
          <Route path="/ledgers/:ledgerId/transactions" element={<TransactionsPage />} />
          <Route path="/ledgers/:ledgerId/stats" element={<StatsPage />} />
          <Route path="/ledgers/:ledgerId/import" element={<ImportPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

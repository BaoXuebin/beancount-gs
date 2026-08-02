import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { request } from '@/api/client'
import type { User } from '@/api/types'

interface AuthState {
  user: User | null
  loading: boolean
  loggedIn: boolean
  refetch: () => void
}

const AuthContext = createContext<AuthState>({
  user: null,
  loading: true,
  loggedIn: false,
  refetch: () => {},
})

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchUser = useCallback(() => {
    setLoading(true)
    request<User>('/users/me')
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  return (
    <AuthContext.Provider value={{ user, loading, loggedIn: user != null, refetch: fetchUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}

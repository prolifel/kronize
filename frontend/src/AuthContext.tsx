import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import { api } from './api'
import type { User } from './types'

interface AuthContextValue {
  user: User | null
  login: (username: string, password: string) => Promise<User>
  logout: () => Promise<void>
  refreshUser: () => Promise<User>
  isAuthenticated: boolean
  loading: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.getCurrentUser()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const res = await api.login({ username, password })
    setUser(res.user)
    return res.user
  }, [])

  const logout = useCallback(async () => {
    await api.logout()
    setUser(null)
  }, [])

  const refreshUser = useCallback(async () => {
    const currentUser = await api.getCurrentUser()
    setUser(currentUser)
    return currentUser
  }, [])

  return (
    <AuthContext.Provider value={{ user, login, logout, refreshUser, isAuthenticated: user !== null, loading }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}

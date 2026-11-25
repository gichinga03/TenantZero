import { useState, useCallback } from 'react'

interface AuthState {
  token: string
  username: string
}

interface UseAuthReturn {
  auth: AuthState | null
  loginError: string
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

export function useAuth(): UseAuthReturn {
  const [auth, setAuth] = useState<AuthState | null>(null)
  const [loginError, setLoginError] = useState('')

  const login = useCallback(async (username: string, password: string) => {
    setLoginError('')
    try {
      const res = await fetch('http://localhost:8080/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const data = await res.json()
      if (res.ok) {
        setAuth({ token: data.token, username })
      } else {
        setLoginError(data.error ?? 'Login failed')
      }
    } catch {
      setLoginError('Server unreachable — is the Go server running?')
    }
  }, [])

  const logout = useCallback(() => {
    setAuth(null)
    setLoginError('')
  }, [])

  return { auth, loginError, login, logout }
}

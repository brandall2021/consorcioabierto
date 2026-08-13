import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'

type User = components['schemas']['User']
type Membership = components['schemas']['Membership']
type Me = components['schemas']['Me']

export type AuthStatus = 'loading' | 'anon' | 'mfa' | 'select-tenant' | 'authenticated'

interface AuthContextValue {
  status: AuthStatus
  user: User | null
  memberships: Membership[]
  me: Me | null
  mfaToken: string | null
  login: (email: string, password: string) => Promise<void>
  verifyMfa: (code: string) => Promise<void>
  selectTenant: (membershipId: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<User | null>(null)
  const [memberships, setMemberships] = useState<Membership[]>([])
  const [me, setMe] = useState<Me | null>(null)
  const [mfaToken, setMfaToken] = useState<string | null>(null)

  const applyAuthResponse = useCallback((res: { user: User; memberships: Membership[] }) => {
    setUser(res.user)
    setMemberships(res.memberships)
    if (res.memberships.length === 1) {
      return res.memberships[0]
    }
    return null
  }, [])

  const selectTenant = useCallback(
    async (membershipId: string) => {
      const { data, error } = await client.POST('/auth/select-tenant', {
        body: { membership_id: membershipId },
      })
      if (error) throw error
      queryClient.clear()
      setMe(data)
      setStatus('authenticated')
    },
    [queryClient],
  )

  const login = useCallback(
    async (email: string, password: string) => {
      const { data, error } = await client.POST('/auth/login', {
        body: { email, password },
      })
      if (error) throw error
      if ('mfa_required' in data) {
        setMfaToken(data.mfa_token)
        setStatus('mfa')
        return
      }
      const single = applyAuthResponse(data)
      if (single) {
        await selectTenant(single.id)
      } else {
        setStatus('select-tenant')
      }
    },
    [applyAuthResponse, selectTenant],
  )

  const verifyMfa = useCallback(
    async (code: string) => {
      if (!mfaToken) throw new Error('Token MFA ausente')
      const { data, error } = await client.POST('/auth/mfa/verify', {
        body: { mfa_token: mfaToken, code },
      })
      if (error) throw error
      const single = applyAuthResponse(data)
      if (single) {
        await selectTenant(single.id)
      } else {
        setStatus('select-tenant')
      }
    },
    [applyAuthResponse, mfaToken, selectTenant],
  )

  const logout = useCallback(async () => {
    try {
      await client.POST('/auth/logout')
    } finally {
      setMe(null)
      setUser(null)
      setMemberships([])
      setMfaToken(null)
      setStatus('anon')
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    client
      .GET('/me')
      .then(({ data, error }) => {
        if (cancelled) return
        if (data) {
          setMe(data)
          setStatus('authenticated')
        } else if (error && error.status !== 401) {
          setStatus('anon')
        } else {
          setStatus('anon')
        }
      })
      .catch(() => {
        if (!cancelled) setStatus('anon')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const value = useMemo(
    () => ({
      status,
      user,
      memberships,
      me,
      mfaToken,
      login,
      verifyMfa,
      selectTenant,
      logout,
    }),
    [status, user, memberships, me, mfaToken, login, verifyMfa, selectTenant, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth debe usarse dentro de AuthProvider')
  return ctx
}

import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router'
import { useAuth } from '@/auth/AuthProvider'
import type { components } from '@/api/generated.d'

type Problem = components['schemas']['Problem']

function errorMessage(err: unknown): string {
  const p = err as Problem
  if (p?.detail) return p.detail
  return 'Error de conexión con el servidor'
}

export function Login() {
  const { status, memberships, login, verifyMfa, selectTenant } = useAuth()
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (status === 'authenticated') {
      navigate('/app', { replace: true })
    }
  }, [status, navigate])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      if (status === 'mfa') {
        await verifyMfa(code.trim())
      } else {
        await login(email, password)
      }
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleSelectTenant(membershipId: string) {
    setError(null)
    setSubmitting(true)
    try {
      await selectTenant(membershipId)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  const selectOnly = status === 'select-tenant' && memberships.length > 0

  return (
    <main className="grid min-h-screen place-items-center p-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-semibold">ConsorcioAbierto</h1>

        {selectOnly ? (
          <section aria-label="Seleccionar consorcio" className="mt-6">
            <p className="text-sm text-gray-600">
              Elegí el consorcio al que querés acceder:
            </p>
            <ul className="mt-3 space-y-2">
              {memberships.map((m) => (
                <li key={m.id}>
                  <button
                    type="button"
                    disabled={submitting}
                    onClick={() => handleSelectTenant(m.id)}
                    className="w-full rounded-lg border bg-white px-4 py-3 text-left shadow-sm transition hover:border-gray-400 disabled:opacity-50"
                  >
                    <span className="font-medium">{m.tenant_name}</span>
                    <span className="ml-2 text-xs text-gray-500">{m.roles.join(', ')}</span>
                    {m.tenant_status === 'suspendido' && (
                      <span className="ml-2 rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-700">
                        suspendido
                      </span>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          </section>
        ) : (
          <form onSubmit={handleSubmit} className="mt-6 space-y-4">
            {status === 'mfa' ? (
              <div>
                <label htmlFor="code" className="block text-sm font-medium text-gray-700">
                  Código de verificación
                </label>
                <input
                  id="code"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  maxLength={6}
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  required
                  autoFocus
                  className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2"
                />
                <p className="mt-1 text-xs text-gray-500">
                  Ingresá el código de 6 dígitos de tu aplicación autenticadora.
                </p>
              </div>
            ) : (
              <>
                <div>
                  <label htmlFor="email" className="block text-sm font-medium text-gray-700">
                    Email
                  </label>
                  <input
                    id="email"
                    type="email"
                    autoComplete="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2"
                  />
                </div>
                <div>
                  <label htmlFor="password" className="block text-sm font-medium text-gray-700">
                    Contraseña
                  </label>
                  <input
                    id="password"
                    type="password"
                    autoComplete="current-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2"
                  />
                </div>
              </>
            )}

            {error && (
              <p role="alert" className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={submitting}
              className="w-full rounded-lg bg-gray-900 px-4 py-2 font-medium text-white transition hover:bg-gray-700 disabled:opacity-50"
            >
              {submitting ? 'Esperá…' : status === 'mfa' ? 'Verificar' : 'Ingresar'}
            </button>
          </form>
        )}
      </div>
    </main>
  )
}

import { useAuth } from '@/auth/AuthProvider'

export function Perfil() {
  const { me } = useAuth()

  if (!me) {
    return (
      <section aria-busy="true">
        <h1 className="text-lg font-medium">Perfil</h1>
        <p className="mt-2 text-gray-500">Cargando…</p>
      </section>
    )
  }

  return (
    <section>
      <h1 className="text-lg font-medium">Perfil</h1>
      <div className="mt-4 max-w-sm rounded-lg border bg-white p-4">
        <p className="font-medium">{me.user.name}</p>
        <p className="text-sm text-gray-500">{me.user.email}</p>
        <p className="mt-2 text-sm text-gray-700">
          MFA: {me.user.mfa_enabled ? 'habilitado' : 'no habilitado'}
        </p>
      </div>
    </section>
  )
}

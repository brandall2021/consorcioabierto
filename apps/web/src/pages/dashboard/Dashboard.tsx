import { useAuth } from '@/auth/AuthProvider'

export function Dashboard() {
  const { me } = useAuth()

  if (!me) {
    return (
      <section aria-busy="true">
        <h1 className="text-lg font-medium">Inicio</h1>
        <p className="mt-2 text-gray-500">Cargando…</p>
      </section>
    )
  }

  return (
    <section>
      <h1 className="text-lg font-medium">Inicio</h1>
      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border bg-white p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-gray-400">Usuario</p>
          <p className="mt-1 font-medium">{me.user.name}</p>
          <p className="text-sm text-gray-500">{me.user.email}</p>
        </div>
        <div className="rounded-lg border bg-white p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-gray-400">Consorcio</p>
          <p className="mt-1 font-medium">{me.membership.tenant_name}</p>
          <p className="text-sm text-gray-500">{me.membership.roles.join(', ')}</p>
        </div>
      </div>
    </section>
  )
}

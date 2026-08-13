import { NavLink, Outlet } from 'react-router'
import { useAuth } from '@/auth/AuthProvider'
import { isEnabled } from '@/lib/featureFlags'

export function AppLayout() {
  const { me, logout } = useAuth()
  const permissions = me?.permissions ?? []
  const canReadAudit = permissions.includes('auditoria.read')

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    `block rounded-md px-3 py-2 text-sm ${
      isActive ? 'bg-gray-900 text-white' : 'text-gray-700 hover:bg-gray-200'
    }`

  return (
    <div className="flex min-h-screen bg-gray-50">
      <aside className="flex w-60 flex-col border-r bg-white">
        <div className="flex h-14 items-center border-b px-4">
          <span className="font-semibold">ConsorcioAbierto</span>
        </div>
        <nav className="flex-1 space-y-1 overflow-y-auto p-3" aria-label="Navegación principal">
          <NavLink to="/app" end className={navLinkClass}>
            Inicio
          </NavLink>

          {isEnabled('consorcios') && (
            <>
              <p className="px-3 pt-3 text-xs font-medium uppercase tracking-wide text-gray-400">
                Consorcios
              </p>
              <NavLink to="/app/consorcios" className={navLinkClass}>
                Resumen
              </NavLink>
            </>
          )}

          {isEnabled('administracion') && (
            <p className="px-3 pt-3 text-xs font-medium uppercase tracking-wide text-gray-400">
              Administración
            </p>
          )}

          {isEnabled('auditoria') && canReadAudit && (
            <NavLink to="/app/auditoria" className={navLinkClass}>
              Auditoría
            </NavLink>
          )}

          {isEnabled('perfil') && (
            <NavLink to="/app/perfil" className={navLinkClass}>
              Perfil
            </NavLink>
          )}
        </nav>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 items-center justify-between border-b bg-white px-4">
          <span className="text-sm text-gray-600">
            {me?.membership?.tenant_name ?? 'ConsorcioAbierto'}
          </span>
          <button
            type="button"
            onClick={() => void logout()}
            className="rounded-md px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-200"
          >
            Salir
          </button>
        </header>
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

import { NavLink, Outlet, useParams } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import { client } from '@/api/client'
import { useAuth } from '@/auth/AuthProvider'
import { ErrorState, SkeletonRows } from '@/components/ui/primitives'

export function ConsorcioLayout() {
  const { consorcioId = '' } = useParams()
  const { me } = useAuth()

  const { data, error, isLoading, refetch } = useQuery({
    queryKey: ['consorcio', consorcioId],
    queryFn: async () => {
      const res = await client.GET('/consorcios/{id}', { params: { path: { id: consorcioId } } })
      if (!res.data) throw new Error('No se pudo cargar el consorcio')
      return res.data
    },
  })

  const canRead = me?.permissions.includes('consorcios.read')

  const tabClass = ({ isActive }: { isActive: boolean }) =>
    `rounded-md px-3 py-1.5 text-sm font-medium ${
      isActive ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'
    }`

  if (!canRead) {
    return (
      <section>
        <p className="text-sm text-gray-500">No tenés permiso para ver este consorcio.</p>
      </section>
    )
  }

  return (
    <section>
      {isLoading && (
        <div className="rounded-lg border bg-white">
          <SkeletonRows rows={3} />
        </div>
      )}

      {error && (
        <ErrorState
          message={`No se pudo cargar el consorcio: ${error instanceof Error ? error.message : 'error desconocido'}`}
          onRetry={() => void refetch()}
        />
      )}

      {data && (
        <>
          <header>
            <h1 className="text-lg font-medium">{data.nombre}</h1>
            {data.cuit && <p className="mt-1 text-sm text-gray-500">CUIT {data.cuit}</p>}
          </header>

          <nav className="mt-4 flex gap-2 border-b" aria-label="Secciones del consorcio">
            <NavLink
              to={`/app/consorcios/${consorcioId}/unidades`}
              end
              className={tabClass}
            >
              Unidades
            </NavLink>
            <NavLink
              to={`/app/consorcios/${consorcioId}/proveedores`}
              className={tabClass}
            >
              Proveedores
            </NavLink>
          </nav>

          <div className="mt-4">
            <Outlet />
          </div>
        </>
      )}
    </section>
  )
}
import { useQuery } from '@tanstack/react-query'
import { client } from '@/api/client'

export function Dashboard() {
  const { data, error, isLoading } = useQuery({
    queryKey: ['health'],
    queryFn: async () => {
      const res = await client.GET('/health')
      if (!res.data) {
        throw new Error('API no disponible')
      }
      return res.data
    },
  })

  if (isLoading) {
    return (
      <section aria-busy="true">
        <h1 className="text-lg font-medium">Estado del API</h1>
        <p className="mt-2 text-gray-500">Cargando…</p>
      </section>
    )
  }

  if (error) {
    return (
      <section>
        <h1 className="text-lg font-medium">Estado del API</h1>
        <p className="mt-2 text-red-600">No se pudo conectar: {error.message}</p>
      </section>
    )
  }

  return (
    <section>
      <h1 className="text-lg font-medium">Estado del API</h1>
      <p className="mt-2 text-gray-700">
        API {data?.status === 'ok' ? 'operativa' : 'en estado desconocido'}
      </p>
    </section>
  )
}

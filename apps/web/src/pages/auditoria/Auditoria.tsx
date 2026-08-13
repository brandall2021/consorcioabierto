import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { client } from '@/api/client'

interface AuditEvent {
  id: string
  tenant_id: string | null
  actor_id: string | null
  actor_membership: string | null
  accion: string
  recurso_type: string
  recurso_id: string | null
  request_id: string | null
  ip: string | null
  user_agent: string | null
  diff: unknown
  created_at: string
}

const PAGE_SIZE = 50

function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('es-AR', {
    dateStyle: 'short',
    timeStyle: 'medium',
    timeZone: 'America/Argentina/Buenos_Aires',
  }).format(new Date(iso))
}

export function Auditoria() {
  const [cursor, setCursor] = useState<string | null>(null)

  const { data, error, isLoading, isFetching } = useQuery({
    queryKey: ['audit-events', cursor],
    queryFn: async () => {
      const res = await client.GET('/audit-events', {
        params: { query: cursor ? { cursor } : {} },
      })
      if (!res.data) throw new Error('No se pudo cargar la auditoría')
      return res.data
    },
  })

  if (isLoading) {
    return (
      <section aria-busy="true">
        <h1 className="text-lg font-medium">Auditoría</h1>
        <p className="mt-2 text-gray-500">Cargando eventos…</p>
      </section>
    )
  }

  if (error) {
    return (
      <section>
        <h1 className="text-lg font-medium">Auditoría</h1>
        <p className="mt-2 text-red-600">{error.message}</p>
        <button
          type="button"
          onClick={() => setCursor(null)}
          className="mt-3 rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
        >
          Reintentar
        </button>
      </section>
    )
  }

  const events = (data?.data ?? []) as AuditEvent[]
  const nextCursor = data?.meta?.next_cursor ?? null

  return (
    <section>
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-medium">Auditoría</h1>
        {isFetching && <span className="text-sm text-gray-400">Actualizando…</span>}
      </div>

      <div className="mt-4 overflow-x-auto rounded-lg border bg-white">
        <table className="w-full text-left text-sm">
          <thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
            <tr>
              <th className="px-3 py-2">Fecha</th>
              <th className="px-3 py-2">Acción</th>
              <th className="px-3 py-2">Recurso</th>
              <th className="px-3 py-2">IP</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {events.length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-6 text-center text-gray-500">
                  No hay eventos para mostrar.
                </td>
              </tr>
            )}
            {events.map((e) => (
              <tr key={e.id}>
                <td className="whitespace-nowrap px-3 py-2 text-gray-500">{formatDate(e.created_at)}</td>
                <td className="px-3 py-2 font-medium">{e.accion}</td>
                <td className="px-3 py-2">
                  {e.recurso_type}
                  {e.recurso_id ? <span className="text-gray-400"> · {e.recurso_id}</span> : null}
                </td>
                <td className="px-3 py-2 text-gray-500">{e.ip ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mt-3 flex items-center justify-between text-sm">
        <span className="text-gray-500">
          Mostrando {events.length} de hasta {PAGE_SIZE} por página
        </span>
        <div className="space-x-2">
          <button
            type="button"
            disabled={!cursor}
            onClick={() => setCursor(null)}
            className="rounded-md border px-3 py-1.5 disabled:opacity-40"
          >
            Anterior
          </button>
          <button
            type="button"
            disabled={!nextCursor}
            onClick={() => setCursor(nextCursor)}
            className="rounded-md border px-3 py-1.5 disabled:opacity-40"
          >
            Siguiente
          </button>
        </div>
      </div>
    </section>
  )
}

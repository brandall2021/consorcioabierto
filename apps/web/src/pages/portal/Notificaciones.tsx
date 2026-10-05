import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { EmptyState } from '@/components/ui/primitives'

type Notificacion = components['schemas']['Notificacion']
type Bandeja = { data: Notificacion[]; noLeidas: number }

const QUERY_KEY = ['portal-notificaciones']

// El hook queda sin exportar: react-refresh pide que un archivo que exporta
// componentes no exporte otra cosa, y el portal no lo necesita por separado.
function useNotificaciones(soloNoLeidas = false) {
	return useQuery({
		queryKey: [...QUERY_KEY, soloNoLeidas],
		queryFn: async (): Promise<Bandeja> => {
			const res = await client.GET('/portal/notificaciones', {
				params: { query: { solo_no_leidas: soloNoLeidas, limite: 50 } },
			})
			if (!res.data) throw new Error('No se pudieron cargar las notificaciones')
			return {
				data: res.data.data ?? [],
				noLeidas: res.data.meta?.no_leidas ?? 0,
			}
		},
	})
}

export function Notificaciones() {
	const queryClient = useQueryClient()
	const query = useNotificaciones()

	const marcarLeida = useMutation({
		mutationFn: async (id: string) => {
			const res = await client.POST('/portal/notificaciones/{id}/leer', { params: { path: { id } } })
			if (!res.data) throw new Error('No se pudo marcar la notificación como leída')
			return res.data
		},
		// El badge y la lista salen del mismo endpoint: invalidarlo recalcula el
		// contador contra el servidor en vez de restarlo a mano y quedar
		// desincronizado.
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: QUERY_KEY })
		},
	})

	const items = query.data?.data ?? []

	return (
		<section className="rounded-lg border bg-white p-4">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Notificaciones</h2>
				{query.data && query.data.noLeidas > 0 && (
					<span className="rounded-full bg-gray-900 px-2 py-0.5 text-xs font-medium text-white">
						{query.data.noLeidas} sin leer
					</span>
				)}
			</div>

			{query.isLoading && <p className="mt-3 text-sm text-gray-500">Cargando notificaciones…</p>}
			{query.error && (
				<p className="mt-3 text-sm text-red-700">
					No se pudieron cargar las notificaciones:{' '}
					{query.error instanceof Error ? query.error.message : 'error desconocido'}
				</p>
			)}

			{query.data && items.length === 0 && (
				<EmptyState title="Sin notificaciones" description="Cuando se publique un comunicado te va a llegar un aviso acá." />
			)}

			{items.length > 0 && (
				<ul className="mt-3 space-y-2">
					{items.map((n) => (
						<li
							key={n.id}
							className={`rounded-md border px-3 py-2 text-sm ${n.leida_at ? 'bg-white' : 'border-gray-900 bg-gray-50'}`}
						>
							<div className="flex items-start justify-between gap-2">
								<div className="min-w-0">
									<p className="font-medium">{n.titulo}</p>
									{n.cuerpo && <p className="mt-1 whitespace-pre-line text-gray-600">{n.cuerpo}</p>}
									<p className="mt-1 text-xs text-gray-500">
										{formatFecha(n.created_at)}
										{n.leida_at ? ' · leída' : ''}
									</p>
								</div>
								{!n.leida_at && (
									<button
										type="button"
										onClick={() => marcarLeida.mutate(n.id)}
										disabled={marcarLeida.isPending}
										className="shrink-0 rounded-md border px-2 py-1 text-xs text-gray-700 hover:bg-gray-100 disabled:opacity-50"
									>
										Marcar leída
									</button>
								)}
							</div>
						</li>
					))}
				</ul>
			)}
		</section>
	)
}

function formatFecha(iso: string) {
	const d = new Date(iso)
	if (Number.isNaN(d.getTime())) return iso
	return new Intl.DateTimeFormat('es-AR', { dateStyle: 'medium' }).format(d)
}
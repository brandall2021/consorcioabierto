import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { useAuth } from '@/auth/AuthProvider'
import { Field, TextInput } from '@/components/ui/Field'
import { PermissionGate } from '@/components/ui/PermissionGate'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import {
	accionesDisponibles,
	etiquetaEstado,
	toneEstado,
	type Accion,
} from './estados'

type Reclamo = components['schemas']['Reclamo']
type ReclamoDetalle = components['schemas']['ReclamoDetalle']
type Unidad = components['schemas']['Unidad']

// El contrato de reclamos no incluye el nombre de la unidad, así que se resuelve
// con el mismo endpoint que usa la pantalla de unidades (queryKey compartida).
function useNombreUnidad(consorcioId: string): (unidadId: string) => string {
	const unidades = useQuery({
		queryKey: ['unidades', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/unidades', {
				params: { path: { id: consorcioId } },
			})
			if (!res.data) throw new Error('No se pudieron cargar las unidades')
			return res.data.data as Unidad[]
		},
		enabled: Boolean(consorcioId),
	})

	const porId = useMemo(() => {
		const map = new Map<string, string>()
		for (const u of unidades.data ?? []) map.set(u.id, u.codigo)
		return map
	}, [unidades.data])

	return (unidadId: string) => porId.get(unidadId) ?? `unidad ${unidadId.slice(0, 8)}`
}

function formatFecha(iso?: string): string {
	if (!iso) return '—'
	const d = new Date(iso)
	if (Number.isNaN(d.getTime())) return '—'
	return d.toLocaleString('es-AR', { dateStyle: 'short', timeStyle: 'short' })
}

export function Reclamos() {
	return (
		<PermissionGate permission="reclamos.read">
			<ReclamosInterno />
		</PermissionGate>
	)
}

function ReclamosInterno() {
	const { consorcioId = '' } = useParams()
	const [seleccionado, setSeleccionado] = useState<string | null>(null)
	const nombreUnidad = useNombreUnidad(consorcioId)

	const lista = useQuery({
		queryKey: ['reclamos', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/reclamos', {
				params: { path: { id: consorcioId } },
			})
			if (!res.data) throw new Error('No se pudieron cargar los reclamos')
			return res.data
		},
		enabled: Boolean(consorcioId),
	})

	const reclamos = (lista.data?.data ?? []) as Reclamo[]
	const total = lista.data?.meta?.total ?? reclamos.length

	return (
		<section>
			<PageHeader
				title="Reclamos"
				description="Reclamos de las unidades del consorcio y su estado de resolución."
			/>

			{lista.isLoading && (
				<div className="mt-4 rounded-lg border bg-white">
					<SkeletonRows rows={4} />
				</div>
			)}

			{lista.error && (
				<div className="mt-4">
					<ErrorState
						message={`No se pudieron cargar los reclamos: ${lista.error instanceof Error ? lista.error.message : 'error desconocido'}`}
						onRetry={() => void lista.refetch()}
					/>
				</div>
			)}

			{lista.data && reclamos.length === 0 && (
				<div className="mt-4 rounded-lg border bg-white">
					<EmptyState title="Todavía no hay reclamos" description="Cuando un consorcista abra un reclamo, aparece acá." />
				</div>
			)}

			{lista.data && reclamos.length > 0 && (
				<div className="mt-4 space-y-4">
					<p className="text-sm text-gray-500">
						{total} {total === 1 ? 'reclamo' : 'reclamos'}
					</p>

					<ul className="divide-y rounded-lg border bg-white">
						{reclamos.map((r) => (
							<li key={r.id}>
								<button
									type="button"
									onClick={() => setSeleccionado(r.id)}
									aria-current={seleccionado === r.id}
									className={`flex w-full items-center justify-between gap-3 px-4 py-3 text-left hover:bg-gray-50 ${
										seleccionado === r.id ? 'bg-gray-50' : ''
									}`}
								>
									<span className="min-w-0">
										<span className="block truncate font-medium">{r.categoria}</span>
										<span className="mt-0.5 block truncate text-xs text-gray-500">
											{nombreUnidad(r.unidad_id)} · {formatFecha(r.created_at)}
										</span>
									</span>
									<Badge tone={toneEstado(r.estado)}>{etiquetaEstado(r.estado)}</Badge>
								</button>
							</li>
						))}
					</ul>

					{seleccionado && <Detalle reclamoId={seleccionado} nombreUnidad={nombreUnidad} />}
				</div>
			)}
		</section>
	)
}

function Detalle({
	reclamoId,
	nombreUnidad,
}: {
	reclamoId: string
	nombreUnidad: (unidadId: string) => string
}) {
	const { me } = useAuth()
	const queryClient = useQueryClient()
	const puedeGestionar = me?.permissions.includes('reclamos.manage') ?? false

	const [accion, setAccion] = useState<Accion | null>(null)
	const [motivo, setMotivo] = useState('')
	const [mensaje, setMensaje] = useState('')
	const [error, setError] = useState<string | null>(null)

	const detalle = useQuery({
		queryKey: ['reclamo', reclamoId],
		queryFn: async () => {
			const res = await client.GET('/reclamos/{id}', { params: { path: { id: reclamoId } } })
			if (!res.data) throw new Error('No se pudo cargar el reclamo')
			return res.data as ReclamoDetalle
		},
	})

	const refrescar = () => {
		void queryClient.invalidateQueries({ queryKey: ['reclamo', reclamoId] })
		void queryClient.invalidateQueries({ queryKey: ['reclamos'] })
	}

	const transicionar = useMutation({
		mutationFn: async (payload: { accion: Accion; motivo: string }) => {
			const res = await client.POST('/reclamos/{id}/transiciones', {
				params: { path: { id: reclamoId } },
				body: payload,
			})
			if (res.error) throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo transicionar')
			return res.data
		},
		onSuccess: () => {
			setAccion(null)
			setMotivo('')
			setError(null)
			refrescar()
		},
		onError: (e) => setError(e instanceof Error ? e.message : 'No se pudo transicionar'),
	})

	const enviarMensaje = useMutation({
		mutationFn: async (texto: string) => {
			const res = await client.POST('/reclamos/{id}/mensajes', {
				params: { path: { id: reclamoId } },
				body: { texto },
			})
			if (res.error) throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo enviar el mensaje')
			return res.data
		},
		onSuccess: () => {
			setMensaje('')
			setError(null)
			refrescar()
		},
		onError: (e) => setError(e instanceof Error ? e.message : 'No se pudo enviar el mensaje'),
	})

	if (detalle.isLoading) {
		return (
			<div className="rounded-lg border bg-white">
				<SkeletonRows rows={3} />
			</div>
		)
	}

	if (detalle.error || !detalle.data) {
		return (
			<ErrorState
				message={`No se pudo cargar el reclamo: ${detalle.error instanceof Error ? detalle.error.message : 'error desconocido'}`}
				onRetry={() => void detalle.refetch()}
			/>
		)
	}

	const reclamo = detalle.data.reclamo
	const disponibles = accionesDisponibles(reclamo.estado)
	const pendiente = accion ? disponibles.find((a) => a.accion === accion) : undefined
	const faltaMotivo = Boolean(pendiente?.requiereMotivo && motivo.trim().length === 0)

	return (
		<article className="rounded-lg border bg-white">
			<header className="flex items-start justify-between gap-3 border-b px-4 py-3">
				<div>
					<h2 className="font-medium">{reclamo.categoria}</h2>
					<p className="mt-0.5 text-xs text-gray-500">
						{nombreUnidad(reclamo.unidad_id)} · abierto {formatFecha(reclamo.created_at)}
					</p>
				</div>
				<Badge tone={toneEstado(reclamo.estado)}>{etiquetaEstado(reclamo.estado)}</Badge>
			</header>

			<div className="space-y-5 px-4 py-4">
				{error && <p className="text-sm text-red-600">{error}</p>}

				{puedeGestionar && (
					<div>
						<h3 className="text-sm font-medium">Acciones</h3>
						<div className="mt-2 flex flex-wrap gap-2">
							{disponibles.map((a) => (
								<button
									key={a.accion}
									type="button"
									onClick={() => {
										setAccion(a.accion)
										setMotivo('')
										setError(null)
									}}
									className="rounded-md border px-3 py-1.5 text-sm font-medium hover:bg-gray-50"
								>
									{a.etiqueta}
								</button>
							))}
						</div>

						{accion && (
							<div className="mt-3 rounded-md border bg-gray-50 p-3">
								<Field
									label={pendiente?.requiereMotivo ? 'Motivo (obligatorio)' : 'Motivo (opcional)'}
									required={pendiente?.requiereMotivo}
									error={faltaMotivo ? 'Indicá el motivo.' : null}
								>
									{(motivoId) => (
										<TextInput
											id={motivoId}
											value={motivo}
											onChange={(e) => setMotivo(e.target.value)}
											placeholder={pendiente?.requiereMotivo ? 'Por qué se reabre' : undefined}
										/>
									)}
								</Field>
								<div className="mt-3 flex gap-2">
									<button
										type="button"
										disabled={faltaMotivo || transicionar.isPending}
										onClick={() => transicionar.mutate({ accion, motivo: motivo.trim() })}
										className="rounded-md bg-gray-900 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
									>
										Confirmar
									</button>
									<button
										type="button"
										onClick={() => setAccion(null)}
										className="rounded-md border px-3 py-1.5 text-sm font-medium"
									>
										Cancelar
									</button>
								</div>
							</div>
						)}
					</div>
				)}

				{!puedeGestionar && (
					<p className="text-sm text-gray-500">
						Se necesita permiso de gestión para actuar sobre el reclamo.
					</p>
				)}

				<div>
					<h3 className="text-sm font-medium">Mensajes</h3>
					{detalle.data.mensajes.length === 0 ? (
						<p className="mt-2 text-sm text-gray-500">Todavía no hay mensajes.</p>
					) : (
						<ul className="mt-2 space-y-2">
							{detalle.data.mensajes.map((m) => (
								<li key={m.id} className="rounded-md border px-3 py-2">
									<p className="text-sm">{m.texto}</p>
									<p className="mt-1 text-xs text-gray-500">{formatFecha(m.created_at)}</p>
								</li>
							))}
						</ul>
					)}

					{puedeGestionar && (
						<div className="mt-3">
							<Field label="Mensaje">
								{(mensajeId) => (
									<TextInput
										id={mensajeId}
										value={mensaje}
										onChange={(e) => setMensaje(e.target.value)}
										placeholder="Escribí una respuesta"
									/>
								)}
							</Field>
							<button
								type="button"
								disabled={mensaje.trim().length === 0 || enviarMensaje.isPending}
								onClick={() => enviarMensaje.mutate(mensaje.trim())}
								className="mt-2 rounded-md bg-gray-900 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
							>
								Enviar mensaje
							</button>
						</div>
					)}
				</div>

				<div>
					<h3 className="text-sm font-medium">Historial</h3>
					{detalle.data.transiciones.length === 0 ? (
						<p className="mt-2 text-sm text-gray-500">Sin movimientos todavía.</p>
					) : (
						<ol className="mt-2 space-y-1">
							{detalle.data.transiciones.map((t) => (
								<li key={t.id} className="text-sm text-gray-600">
									<span className="font-medium">{t.accion}</span> · {formatFecha(t.created_at)}
									{t.motivo ? ` — ${t.motivo}` : ''}
								</li>
							))}
						</ol>
					)}
				</div>
			</div>
		</article>
	)
}

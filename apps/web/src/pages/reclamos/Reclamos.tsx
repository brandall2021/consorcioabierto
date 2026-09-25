import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'

type Reclamo = components['schemas']['Reclamo'] & {
	texto?: string
	created_at?: string
	updated_at?: string
	responsable_id?: string | null
}

type ReclamoDetalle = components['schemas']['ReclamoDetalle'] & {
	reclamo: Reclamo & { texto?: string }
}
	type ReclamosResponse = { data: Reclamo[] }

export function Reclamos() {
	const { consorcioId = '' } = useParams()
	const queryClient = useQueryClient()
	const [unidadId, setUnidadId] = useState('')
	const [categoria, setCategoria] = useState('general')
	const [texto, setTexto] = useState('')
	const [selectedId, setSelectedId] = useState<string | null>(null)
	const [mensaje, setMensaje] = useState('')
	const [motivo, setMotivo] = useState('')
	const [responsableId, setResponsableId] = useState('')

	const listQuery = useQuery({
		queryKey: ['reclamos', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/reclamos', { params: { path: { id: consorcioId } } })
			if (!res.data) throw new Error('No se pudieron cargar los reclamos')
			return res.data as ReclamosResponse
		},
	})

	const reclamos = useMemo(() => listQuery.data?.data ?? [], [listQuery.data?.data])
	const selected = useMemo(() => selectedId ?? reclamos[0]?.id ?? null, [selectedId, reclamos])

	const detailQuery = useQuery({
		queryKey: ['reclamo', selected],
		queryFn: async () => {
			if (!selected) throw new Error('Sin reclamo seleccionado')
			const res = await client.GET('/reclamos/{id}', { params: { path: { id: selected } } })
			if (!res.data) throw new Error('No se pudo cargar el reclamo')
			return res.data as ReclamoDetalle
		},
		enabled: Boolean(selected),
	})

	useEffect(() => {
		if (!selectedId && reclamos[0]) {
			setSelectedId(reclamos[0].id)
		}
	}, [selectedId, reclamos])

	const create = useMutation({
		mutationFn: async () => {
			const res = await client.POST('/consorcios/{id}/reclamos', {
				params: { path: { id: consorcioId } },
				body: { unidad_id: unidadId, categoria, texto },
			})
			if (!res.data) throw new Error('No se pudo crear el reclamo')
			return res.data
		},
		onSuccess: async () => {
			setUnidadId('')
			setCategoria('general')
			setTexto('')
			await queryClient.invalidateQueries({ queryKey: ['reclamos', consorcioId] })
		},
	})

	const message = useMutation({
		mutationFn: async () => {
			if (!selected) throw new Error('Sin reclamo seleccionado')
			const res = await client.POST('/reclamos/{id}/mensajes', {
				params: { path: { id: selected } },
				body: { texto: mensaje || ' ' },
			})
			if (!res.data) throw new Error('No se pudo agregar el mensaje')
			return res.data
		},
		onSuccess: async () => {
			setMensaje('')
			await queryClient.invalidateQueries({ queryKey: ['reclamo', selected] })
		},
	})

	const transition = useMutation({
		mutationFn: async (accion: 'asignar' | 'en_progreso' | 'resolver' | 'cerrar' | 'reabrir') => {
			if (!selected) throw new Error('Sin reclamo seleccionado')
			const res = await client.POST('/reclamos/{id}/transiciones', {
				params: { path: { id: selected } },
				body: { accion, motivo: motivo || null, responsable_id: responsableId || null },
			})
			if (!res.data) throw new Error('No se pudo transicionar el reclamo')
			return res.data
		},
		onSuccess: async () => {
			setMotivo('')
			setResponsableId('')
			await queryClient.invalidateQueries({ queryKey: ['reclamo', selected] })
			await queryClient.invalidateQueries({ queryKey: ['reclamos', consorcioId] })
		},
	})

	const detail = detailQuery.data

	return (
		<section className="space-y-4">
			<PageHeader title="Reclamos" description="Seguimiento de la conversación y del estado de cada caso." />

			<form
				className="grid gap-3 rounded-2xl border bg-white p-4 shadow-sm md:grid-cols-[1fr_1fr_2fr_auto]"
				onSubmit={(e) => {
					e.preventDefault()
					if (unidadId.trim() && categoria.trim() && texto.trim()) void create.mutateAsync()
				}}
			>
				<label className="text-sm text-gray-600">
					Unidad ID
					<input className="mt-1 w-full rounded-md border px-3 py-2" value={unidadId} onChange={(e) => setUnidadId(e.target.value)} required />
				</label>
				<label className="text-sm text-gray-600">
					Categoría
					<input className="mt-1 w-full rounded-md border px-3 py-2" value={categoria} onChange={(e) => setCategoria(e.target.value)} required />
				</label>
				<label className="text-sm text-gray-600 md:col-span-2">
					Texto
					<textarea className="mt-1 min-h-20 w-full rounded-md border px-3 py-2" value={texto} onChange={(e) => setTexto(e.target.value)} required />
				</label>
				<button type="submit" disabled={create.isPending} className="h-10 rounded-md bg-gray-900 px-4 text-sm font-medium text-white disabled:opacity-40">
					{create.isPending ? 'Creando…' : 'Crear'}
				</button>
			</form>

			<div className="grid gap-4 lg:grid-cols-[0.9fr_1.1fr]">
				<div className="rounded-2xl border bg-white shadow-sm">
					<div className="border-b px-4 py-3">
						<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Casos</h2>
					</div>
					{listQuery.isLoading && <SkeletonRows rows={4} />}
					{listQuery.error && <ErrorState message={`No se pudieron cargar los reclamos: ${listQuery.error instanceof Error ? listQuery.error.message : 'error desconocido'}`} onRetry={() => void listQuery.refetch()} />}
					{!listQuery.isLoading && !listQuery.error && reclamos.length === 0 && <EmptyState title="Sin reclamos" description="Creá el primero para abrir una conversación." />}
					<div className="divide-y">
						{reclamos.map((r) => (
							<button key={r.id} type="button" onClick={() => setSelectedId(r.id)} className={`block w-full px-4 py-3 text-left ${selected === r.id ? 'bg-gray-50' : 'hover:bg-gray-50'}`}>
								<div className="flex items-start justify-between gap-3">
									<div>
										<p className="font-medium text-gray-900">{r.categoria}</p>
										<p className="text-sm text-gray-500">{r.texto}</p>
									</div>
									<Badge tone={r.estado === 'cerrado' ? 'gray' : r.estado === 'resuelto' ? 'green' : 'amber'}>{r.estado}</Badge>
								</div>
							</button>
						))}
					</div>
				</div>

				<div className="rounded-2xl border bg-white shadow-sm">
					<div className="border-b px-4 py-3">
						<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Detalle</h2>
					</div>
					{detailQuery.isLoading && <SkeletonRows rows={5} />}
					{detailQuery.error && <ErrorState message={`No se pudo cargar el reclamo: ${detailQuery.error instanceof Error ? detailQuery.error.message : 'error desconocido'}`} onRetry={() => void detailQuery.refetch()} />}
					{detail && (
						<div className="space-y-4 p-4">
							<div className="grid gap-3 md:grid-cols-2">
								<div>
									<p className="text-xs uppercase tracking-wide text-gray-400">Estado</p>
									<Badge tone={detail.reclamo.estado === 'cerrado' ? 'gray' : detail.reclamo.estado === 'resuelto' ? 'green' : 'amber'}>{detail.reclamo.estado}</Badge>
								</div>
								<div>
									<p className="text-xs uppercase tracking-wide text-gray-400">Categoría</p>
									<p className="font-medium text-gray-900">{detail.reclamo.categoria}</p>
								</div>
							</div>

							<div className="space-y-2">
								<p className="text-sm font-medium text-gray-900">{detail.reclamo.texto}</p>
							</div>

							<form className="space-y-2 rounded-xl border p-3" onSubmit={(e) => { e.preventDefault(); if (mensaje.trim()) void message.mutateAsync() }}>
								<label className="block text-sm text-gray-600">
									Mensaje
									<textarea className="mt-1 min-h-20 w-full rounded-md border px-3 py-2" value={mensaje} onChange={(e) => setMensaje(e.target.value)} />
								</label>
								<div className="flex items-center justify-between gap-3">
									<span className="text-xs text-gray-400">Se guarda en el hilo del reclamo.</span>
									<button type="submit" disabled={message.isPending} className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40">
										{message.isPending ? 'Enviando…' : 'Agregar mensaje'}
									</button>
								</div>
							</form>

							<div className="grid gap-3 md:grid-cols-2">
								<label className="text-sm text-gray-600">
									Responsable ID
									<input className="mt-1 w-full rounded-md border px-3 py-2" value={responsableId} onChange={(e) => setResponsableId(e.target.value)} />
								</label>
								<label className="text-sm text-gray-600">
									Motivo
									<input className="mt-1 w-full rounded-md border px-3 py-2" value={motivo} onChange={(e) => setMotivo(e.target.value)} />
								</label>
							</div>

							<div className="flex flex-wrap gap-2">
								{(['asignar', 'en_progreso', 'resolver', 'cerrar', 'reabrir'] as const).map((accion) => (
									<button key={accion} type="button" onClick={() => transition.mutate(accion)} disabled={transition.isPending} className="rounded-md border px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-40">
										{accion}
									</button>
								))}
							</div>

							<div className="space-y-3">
								<div>
									<p className="text-sm font-medium text-gray-900">Mensajes</p>
									{detail.mensajes.length === 0 ? <p className="text-sm text-gray-500">Sin mensajes todavía.</p> : detail.mensajes.map((m) => <p key={m.id} className="mt-2 rounded-lg bg-gray-50 p-3 text-sm text-gray-700">{m.texto}</p>)}
								</div>
								<div>
									<p className="text-sm font-medium text-gray-900">Historial</p>
									{detail.transiciones.length === 0 ? <p className="text-sm text-gray-500">Sin transiciones todavía.</p> : detail.transiciones.map((t) => <p key={t.id} className="mt-2 text-sm text-gray-600">{t.accion}{t.motivo ? ` · ${t.motivo}` : ''}</p>)}
								</div>
							</div>
						</div>
					)}
				</div>
			</div>
		</section>
	)
}

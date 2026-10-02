import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { useAuth } from '@/auth/AuthProvider'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import { etiquetaEstado, toneEstado, type EstadoReclamo } from '@/pages/reclamos/estados'

type PortalHome = components['schemas']['PortalHome']
type PortalResponse = { data: PortalHome }

export function Portal() {
	const { logout, me } = useAuth()
	const [showForm, setShowForm] = useState(false)
	const queryClient = useQueryClient()
	const portalQuery = useQuery({
		queryKey: ['portal-home'],
		queryFn: async () => {
			const res = await client.GET('/portal')
			if (!res.data) throw new Error('No se pudo cargar el portal')
			return res.data as PortalResponse
		},
	})

	const home = portalQuery.data?.data
	const unidades = home?.unidades ?? []

	return (
		<section className="space-y-6">
			<PageHeader
				title="Portal consorcista"
				description="Tu deuda, recibos recientes y accesos rápidos a tu actividad."
				actions={
					<div className="flex gap-2">
						<button
							type="button"
							onClick={() => void logout()}
							className="rounded-md border px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
						>
							Salir
						</button>
					</div>
				}
			/>

			{portalQuery.isLoading && <SkeletonRows rows={4} />}
			{portalQuery.error && (
				<ErrorState
					message={`No se pudo cargar el portal: ${portalQuery.error instanceof Error ? portalQuery.error.message : 'error desconocido'}`}
					onRetry={() => void portalQuery.refetch()}
				/>
			)}

			{home && (
				<div className="space-y-6">
					<div className="grid gap-4 md:grid-cols-3">
						<div className="rounded-lg border bg-white p-4">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Saldo vencido total</p>
							<p className="mt-2 text-2xl font-semibold">{formatMoney(home.total_saldo_vencido_cents)}</p>
						</div>
						<div className="rounded-lg border bg-white p-4">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Consorcios</p>
							<p className="mt-2 text-2xl font-semibold">{home.consorcios.length}</p>
						</div>
						<div className="rounded-lg border bg-white p-4">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Recibos recientes</p>
							<p className="mt-2 text-2xl font-semibold">{home.recibos_recientes.length}</p>
						</div>
					</div>

					<section className="rounded-lg border bg-white p-4">
						<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Consorcios</h2>
						{home.consorcios.length ? (
							<div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
								{home.consorcios.map((consorcio) => (
									<div key={consorcio.id} className="rounded-md border p-4">
										<div className="flex items-center justify-between gap-2">
											<div>
												<p className="font-medium">{consorcio.nombre}</p>
												<p className="text-sm text-gray-500">{consorcio.cargos_vencidos} cargos vencidos</p>
											</div>
											<Badge tone={consorcio.saldo_vencido_cents > 0 ? 'amber' : 'green'}>
												{consorcio.estado}
											</Badge>
										</div>
										<p className="mt-3 text-sm text-gray-600">Saldo vencido: {formatMoney(consorcio.saldo_vencido_cents)}</p>
										<p className="text-xs text-gray-500">Vencido desde {consorcio.vencido_desde || 'sin deuda vencida'}</p>
									</div>
								))}
							</div>
						) : (
							<EmptyState title="No hay consorcios para mostrar" description="Cuando tengas acceso a un consorcio, va a aparecer acá." />
						)}
					</section>

					<div className="grid gap-6 lg:grid-cols-2">
						<section className="rounded-lg border bg-white p-4">
							<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Recibos recientes</h2>
							{home.recibos_recientes.length ? (
								<ul className="mt-3 space-y-2">
									{home.recibos_recientes.map((r) => (
										<li key={r.cobranza_id} className="rounded-md border px-3 py-2 text-sm">
											<div className="flex items-center justify-between gap-2">
												<span className="font-medium">{r.consorcio_nombre}</span>
												<Badge tone={r.estado === 'acreditado' ? 'green' : 'gray'}>{r.estado}</Badge>
											</div>
											<p className="mt-1 text-gray-600">{r.fecha} · {formatMoney(r.importe_cents)}</p>
											<p className="text-xs text-gray-500">Saldo a favor: {formatMoney(r.saldo_a_favor_cents)}</p>
										</li>
									))}
								</ul>
							) : (
								<EmptyState title="Sin recibos recientes" description="Tus cobros acreditados van a aparecer acá." />
							)}
						</section>

					<section className="rounded-lg border bg-white p-4">
						<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Comunicados</h2>
						{home.comunicados_recientes.length ? (
							<ul className="mt-3 space-y-2">
								{home.comunicados_recientes.map((comunicado) => (
									<li key={comunicado.id} className="rounded-md border px-3 py-2 text-sm">
										<div className="flex items-center justify-between gap-2">
											<span className="font-medium">{comunicado.titulo}</span>
											<Badge tone={comunicado.estado === 'publicado' ? 'green' : 'gray'}>{comunicado.estado}</Badge>
										</div>
										<p className="mt-1 text-gray-600">{comunicado.consorcio_nombre}</p>
										<p className="text-xs text-gray-500">Publicado {new Intl.DateTimeFormat('es-AR', { dateStyle: 'medium' }).format(new Date(comunicado.publicado_at))}</p>
									</li>
								))}
							</ul>
						) : (
							<EmptyState title="Sin comunicados" description="Esta sección se completa en la próxima fase." />
						)}
					</section>
					</div>

					<section className="rounded-lg border bg-white p-4">
						<div className="flex flex-wrap items-center justify-between gap-2">
							<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Reclamos</h2>
							{unidades.length > 0 && (
								<button
									type="button"
									onClick={() => setShowForm((v) => !v)}
									className="rounded-md bg-gray-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-gray-700"
								>
									{showForm ? 'Cancelar' : 'Abrir reclamo'}
								</button>
							)}
						</div>

						{showForm && unidades.length > 0 && (
							<NuevoReclamo
								unidades={unidades}
								onCreated={() => {
									setShowForm(false)
									void queryClient.invalidateQueries({ queryKey: ['portal-home'] })
								}}
							/>
						)}

						{unidades.length === 0 && (
							<p className="mt-3 text-sm text-gray-500">
								No tenés unidades con vínculo vigente, así que todavía no podés abrir reclamos desde acá.
							</p>
						)}

						{home.reclamos_recientes.length ? (
							<ul className="mt-3 space-y-2">
								{home.reclamos_recientes.map((reclamo) => (
									<li key={reclamo.id} className="rounded-md border px-3 py-2 text-sm">
										<div className="flex items-center justify-between gap-2">
											<span className="font-medium">{reclamo.categoria}</span>
											<Badge tone={toneEstado(reclamo.estado as EstadoReclamo)}>
												{etiquetaEstado(reclamo.estado as EstadoReclamo)}
											</Badge>
										</div>
										<p className="mt-1 text-gray-600">{reclamo.texto}</p>
										<p className="text-xs text-gray-500">
											Abierto{' '}
											{new Intl.DateTimeFormat('es-AR', { dateStyle: 'medium' }).format(
												new Date(reclamo.created_at),
											)}
										</p>
									</li>
								))}
							</ul>
						) : (
							<EmptyState
								title="Todavía no abriste ningún reclamo"
								description="Si tenés un problema en tu unidad, abrilo acá y seguí su estado."
							/>
						)}
					</section>
				</div>
			)}

			{!portalQuery.isLoading && !portalQuery.error && !home && (
				<p className="text-sm text-gray-500">{me?.membership?.tenant_name ?? 'Tu portal'}</p>
			)}
		</section>
	)
}

type UnidadResumen = components['schemas']['PortalUnidadResumen']

const CATEGORIAS_SUGERIDAS = ['humedad', 'plomería', 'eléctrica', 'ascensor', 'seguridad', 'ruido']

function NuevoReclamo({
	unidades,
	onCreated,
}: {
	unidades: UnidadResumen[]
	onCreated: () => void
}) {
	const [unidadId, setUnidadId] = useState(unidades[0]?.id ?? '')
	const [categoria, setCategoria] = useState('')
	const [texto, setTexto] = useState('')

	const crear = useMutation({
		mutationFn: async () => {
			const res = await client.POST('/portal/reclamos', {
				body: { unidad_id: unidadId, categoria, texto },
			})
			if (res.error) {
				throw new Error(
					(res.error as { detail?: string }).detail ?? 'No se pudo abrir el reclamo',
				)
			}
			return res.data
		},
		onSuccess: () => onCreated(),
	})

	const puedeEnviar = unidadId !== '' && categoria.trim() !== '' && texto.trim().length >= 3

	return (
		<form
			className="mt-4 space-y-3 rounded-md border bg-gray-50 p-4"
			onSubmit={(e) => {
				e.preventDefault()
				if (puedeEnviar) crear.mutate()
			}}
		>
			<div>
				<label htmlFor="reclamo-unidad" className="block text-sm font-medium">
					Unidad
				</label>
				<select
					id="reclamo-unidad"
					value={unidadId}
					onChange={(e) => setUnidadId(e.target.value)}
					className="mt-1 w-full rounded-md border bg-white px-2 py-1.5 text-sm"
				>
					{unidades.map((u) => (
						<option key={u.id} value={u.id}>
							Unidad {u.codigo}
						</option>
					))}
				</select>
			</div>

			<div>
				<label htmlFor="reclamo-categoria" className="block text-sm font-medium">
					Categoría
				</label>
				<input
					id="reclamo-categoria"
					list="reclamo-categorias"
					value={categoria}
					onChange={(e) => setCategoria(e.target.value)}
					placeholder="humedad"
					className="mt-1 w-full rounded-md border bg-white px-2 py-1.5 text-sm"
				/>
				<datalist id="reclamo-categorias">
					{CATEGORIAS_SUGERIDAS.map((c) => (
						<option key={c} value={c} />
					))}
				</datalist>
			</div>

			<div>
				<label htmlFor="reclamo-texto" className="block text-sm font-medium">
					¿Qué pasa?
				</label>
				<textarea
					id="reclamo-texto"
					value={texto}
					onChange={(e) => setTexto(e.target.value)}
					rows={3}
					placeholder="Contale a la administración qué problema tenés."
					className="mt-1 w-full rounded-md border bg-white px-2 py-1.5 text-sm"
				/>
			</div>

			{crear.error && (
				<p className="text-sm text-red-700" role="alert">
					{crear.error instanceof Error ? crear.error.message : 'No se pudo abrir el reclamo'}
				</p>
			)}

			<button
				type="submit"
				disabled={!puedeEnviar || crear.isPending}
				className="rounded-md bg-gray-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
			>
				{crear.isPending ? 'Enviando…' : 'Enviar reclamo'}
			</button>
		</form>
	)
}

function formatMoney(cents: number) {
	return new Intl.NumberFormat('es-AR', { style: 'currency', currency: 'ARS', maximumFractionDigits: 0 }).format(cents / 100)
}

import { useQuery } from '@tanstack/react-query'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { useAuth } from '@/auth/AuthProvider'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'

type PortalHome = components['schemas']['PortalHome']
type PortalResponse = { data: PortalHome }

export function Portal() {
	const { logout, me } = useAuth()
	const portalQuery = useQuery({
		queryKey: ['portal-home'],
		queryFn: async () => {
			const res = await client.GET('/portal')
			if (!res.data) throw new Error('No se pudo cargar el portal')
			return res.data as PortalResponse
		},
	})

	const home = portalQuery.data?.data

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
						<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Reclamos</h2>
						<EmptyState title="Sin reclamos" description="Los reclamos del portal se habilitan en la siguiente etapa." />
					</section>
				</div>
			)}

			{!portalQuery.isLoading && !portalQuery.error && !home && (
				<p className="text-sm text-gray-500">{me?.membership?.tenant_name ?? 'Tu portal'}</p>
			)}
		</section>
	)
}

function formatMoney(cents: number) {
	return new Intl.NumberFormat('es-AR', { style: 'currency', currency: 'ARS', maximumFractionDigits: 0 }).format(cents / 100)
}

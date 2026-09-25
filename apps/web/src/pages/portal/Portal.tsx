import { useQuery } from '@tanstack/react-query'
import { useAuth } from '@/auth/AuthProvider'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows, EstadoBadge } from '@/components/ui/primitives'
import { formatCents } from '@/lib/format'

type PortalHome = components['schemas']['PortalHome']

export function Portal() {
	const { me, logout } = useAuth()

	const query = useQuery({
		queryKey: ['portal-home'],
		queryFn: async () => {
			const res = await client.GET('/portal')
			if (!res.data) throw new Error('No se pudo cargar el portal')
			return res.data as { data: PortalHome; meta: { request_id: string } }
		},
	})

	const home = query.data?.data

	return (
		<section className="mx-auto flex max-w-6xl flex-col gap-6">
			<PageHeader
				title="Inicio del consorcista"
				description="Vista de transición tenant-level para deuda, recibos y próximos comunicados."
				actions={
					<div className="flex items-center gap-3 text-sm text-gray-500">
						<span className="font-medium text-gray-900">{me?.membership?.tenant_name ?? 'ConsorcioAbierto'}</span>
						<button type="button" onClick={() => void logout()} className="rounded-md border px-3 py-1.5 text-gray-700 hover:bg-gray-50">
							Salir
						</button>
					</div>
				}
			/>

			{query.isLoading && <SkeletonRows rows={4} />}

			{query.error && (
				<ErrorState
					message={`No se pudo cargar el portal: ${query.error instanceof Error ? query.error.message : 'error desconocido'}`}
					onRetry={() => void query.refetch()}
				/>
			)}

			{home && (
				<>
					<div className="grid gap-4 md:grid-cols-3">
						<div className="rounded-2xl border bg-white p-5 shadow-sm">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Saldo vencido total</p>
							<p className="mt-3 text-3xl font-semibold text-gray-900">{formatCents(home.total_saldo_vencido_cents)}</p>
						</div>
						<div className="rounded-2xl border bg-white p-5 shadow-sm">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Consorcios</p>
							<p className="mt-3 text-3xl font-semibold text-gray-900">{home.consorcios.length}</p>
						</div>
						<div className="rounded-2xl border bg-white p-5 shadow-sm">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Recibos recientes</p>
							<p className="mt-3 text-3xl font-semibold text-gray-900">{home.recibos_recientes.length}</p>
						</div>
					</div>

					<div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
						<div className="rounded-2xl border bg-white shadow-sm">
							<div className="border-b px-5 py-4">
								<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Consorcios con mora</h2>
							</div>
							{home.consorcios.length === 0 ? (
								<div className="p-5">
									<EmptyState title="Sin deuda vencida" description="No hay saldos vencidos para mostrar en este momento." />
								</div>
							) : (
								<div className="overflow-x-auto">
									<table className="w-full text-left text-sm">
										<thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
											<tr>
												<th className="px-4 py-3">Consorcio</th>
												<th className="px-4 py-3">Estado</th>
												<th className="px-4 py-3">Saldo</th>
												<th className="px-4 py-3">Cargos</th>
												<th className="px-4 py-3">Desde</th>
											</tr>
										</thead>
										<tbody className="divide-y">
											{home.consorcios.map((c) => (
												<tr key={c.id}>
													<td className="px-4 py-3 font-medium text-gray-900">{c.nombre}</td>
													<td className="px-4 py-3"><EstadoBadge estado={c.estado} /></td>
													<td className="px-4 py-3 text-gray-600">{formatCents(c.saldo_vencido_cents)}</td>
													<td className="px-4 py-3 text-gray-600">{c.cargos_vencidos}</td>
													<td className="px-4 py-3 text-gray-600">{c.vencido_desde || '—'}</td>
												</tr>
											))}
										</tbody>
									</table>
								</div>
							)}
						</div>

						<div className="rounded-2xl border bg-white shadow-sm">
							<div className="border-b px-5 py-4">
								<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Recibos recientes</h2>
							</div>
							{home.recibos_recientes.length === 0 ? (
								<div className="p-5">
									<EmptyState title="Sin recibos" description="Todavía no hay pagos para mostrar." />
								</div>
							) : (
								<div className="space-y-0 divide-y">
									{home.recibos_recientes.map((r) => (
										<div key={r.cobranza_id} className="px-5 py-4">
											<div className="flex items-start justify-between gap-4">
												<div>
													<p className="font-medium text-gray-900">{r.consorcio_nombre}</p>
													<p className="text-sm text-gray-500">{r.fecha}</p>
												</div>
												<div className="text-right">
													<p className="font-medium text-gray-900">{formatCents(r.importe_cents)}</p>
													<Badge tone="green">{r.estado}</Badge>
												</div>
											</div>
											<p className="mt-2 text-xs text-gray-400">Recibo {r.cobranza_id}</p>
										</div>
									))}
								</div>
							)}
						</div>
					</div>

					<div className="grid gap-6 lg:grid-cols-2">
						<div className="rounded-2xl border bg-white shadow-sm">
							<div className="border-b px-5 py-4">
								<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Comunicados recientes</h2>
							</div>
							{home.comunicados_recientes.length === 0 ? (
								<div className="p-5">
									<EmptyState title="Sin comunicados" description="Todavía no hay anuncios publicados para mostrar." />
								</div>
							) : (
								<div className="space-y-0 divide-y">
									{home.comunicados_recientes.map((c) => (
										<div key={c.id} className="px-5 py-4">
											<div className="flex items-start justify-between gap-4">
												<div>
													<p className="font-medium text-gray-900">{c.titulo}</p>
													<p className="text-sm text-gray-500">{c.publicado_at}</p>
												</div>
												<Badge tone="blue">{c.estado}</Badge>
											</div>
											<p className="mt-2 text-xs text-gray-400">{c.consorcio_nombre}</p>
										</div>
									))}
								</div>
							)}
						</div>

						<div className="rounded-2xl border bg-white shadow-sm">
							<div className="border-b px-5 py-4">
								<h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">Reclamos recientes</h2>
							</div>
							{home.reclamos_recientes.length === 0 ? (
								<div className="p-5">
									<EmptyState title="Sin reclamos" description="Todavía no hay reclamos abiertos para mostrar." />
								</div>
							) : (
								<div className="space-y-0 divide-y">
									{home.reclamos_recientes.map((r) => (
										<div key={r.id} className="px-5 py-4">
											<div className="flex items-start justify-between gap-4">
												<div>
													<p className="font-medium text-gray-900">{r.categoria}</p>
													<p className="text-sm text-gray-500">{r.created_at}</p>
												</div>
												<EstadoBadge estado={r.estado} />
											</div>
											<p className="mt-2 text-sm text-gray-600">{r.texto}</p>
										</div>
									))}
								</div>
							)}
						</div>
					</div>
				</>
			)}
		</section>
	)
}

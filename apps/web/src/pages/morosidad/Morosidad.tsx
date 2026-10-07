import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import { formatCents } from '@/lib/format'

type MorosidadItem = {
	unidad_id: string
	unidad_codigo: string
	saldo_vencido_cents: number
	cantidad_cargos: number
	vencido_desde: string
}

type MorosidadResponse = {
	data: MorosidadItem[]
	meta: { request_id: string; total_saldo_cents: number; total_cargos_vencidos: number }
}

export function Morosidad() {
	const { consorcioId = '' } = useParams()

	const morosidadQuery = useQuery({
		queryKey: ['morosidad', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/morosidad', { params: { path: { id: consorcioId } } })
			if (!res.data) throw new Error('No se pudo cargar la morosidad')
			return res.data as MorosidadResponse
		},
	})

	const items = morosidadQuery.data?.data ?? []

	return (
		<section>
			<PageHeader title="Morosidad" description="Saldos vencidos por UF." />

			<div className="mt-4 space-y-4">
				{morosidadQuery.isLoading && (
					<div className="rounded-lg border bg-white">
						<SkeletonRows rows={4} />
					</div>
				)}

				{morosidadQuery.error && (
					<ErrorState
						message={`No se pudo cargar la morosidad: ${morosidadQuery.error instanceof Error ? morosidadQuery.error.message : 'error desconocido'}`}
						onRetry={() => void morosidadQuery.refetch()}
					/>
				)}

				{morosidadQuery.data && (
					<div className="space-y-4">
						<div className="grid gap-3 md:grid-cols-3">
							<div className="rounded-lg border bg-white p-4">
								<p className="text-xs uppercase tracking-wide text-gray-500">UFs con mora</p>
								<p className="mt-1 font-medium">{items.length}</p>
							</div>
							<div className="rounded-lg border bg-white p-4">
								<p className="text-xs uppercase tracking-wide text-gray-500">Saldo vencido</p>
								<p className="mt-1 font-medium">{formatCents(morosidadQuery.data.meta.total_saldo_cents)}</p>
							</div>
							<div className="rounded-lg border bg-white p-4">
								<p className="text-xs uppercase tracking-wide text-gray-500">Cargos vencidos</p>
								<p className="mt-1 font-medium">{morosidadQuery.data.meta.total_cargos_vencidos}</p>
							</div>
						</div>

						{items.length === 0 ? (
							<EmptyState title="Sin mora" description="No hay cargos vencidos con saldo pendiente." />
						) : (
							<div className="overflow-x-auto rounded-lg border bg-white">
								<table className="w-full text-left text-sm">
									<thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
										<tr>
											<th className="px-3 py-2">UF</th>
											<th className="px-3 py-2">Saldo vencido</th>
											<th className="px-3 py-2">Cargos</th>
											<th className="px-3 py-2">Desde</th>
											<th className="px-3 py-2">Alerta</th>
										</tr>
									</thead>
									<tbody className="divide-y">
										{items.map((item) => (
											<tr key={item.unidad_id}>
												<td className="px-3 py-2 font-medium">{item.unidad_codigo}</td>
												<td className="px-3 py-2 text-gray-600">{formatCents(item.saldo_vencido_cents)}</td>
												<td className="px-3 py-2 text-gray-600">{item.cantidad_cargos}</td>
												<td className="px-3 py-2 text-gray-600">{item.vencido_desde}</td>
												<td className="px-3 py-2"><Badge tone={item.saldo_vencido_cents > 50000 ? 'red' : 'amber'}>{item.saldo_vencido_cents > 50000 ? 'Crítica' : 'Seguimiento'}</Badge></td>
											</tr>
										))}
									</tbody>
								</table>
							</div>
						)}
					</div>
				)}
			</div>
		</section>
	)
}

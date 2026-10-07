import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import { Field, Select } from '@/components/ui/Field'
import { formatCents } from '@/lib/format'

type Unidad = components['schemas']['Unidad']

type Movimiento = {
	id: string
	unidad_id: string
	tipo: string
	fecha_efectiva: string
	debit_cents: number
	credit_cents: number
	currency: string
	referencia?: string | null
}

type CuentaCorrienteResponse = {
	data: Movimiento[]
	meta: { request_id: string; saldo_cents: number }
}

const EMPTY_UNIDADES: Unidad[] = []
const EMPTY_MOVIMIENTOS: Movimiento[] = []

export function CuentaCorriente() {
	const { consorcioId = '' } = useParams()
	const [unidadId, setUnidadId] = useState('')

	const unidadesQuery = useQuery({
		queryKey: ['unidades', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/unidades', { params: { path: { id: consorcioId } } })
			if (!res.data) throw new Error('No se pudieron cargar las unidades')
			return res.data as { data: Unidad[] }
		},
	})

	const unidades = (unidadesQuery.data?.data ?? EMPTY_UNIDADES) as Unidad[]

	useEffect(() => {
		if (!unidadId && unidades[0]) {
			setUnidadId(unidades[0].id)
		}
	}, [unidadId, unidades])

	const cuentaQuery = useQuery({
		queryKey: ['cuenta-corriente', consorcioId, unidadId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/unidades/{unidadId}/cuenta-corriente', {
				params: { path: { id: consorcioId, unidadId } },
			})
			if (!res.data) throw new Error('No se pudo cargar la cuenta corriente')
			return res.data as CuentaCorrienteResponse
		},
		enabled: unidadId !== '',
	})

	const movimientos = (cuentaQuery.data?.data ?? EMPTY_MOVIMIENTOS) as Movimiento[]
	const unidadActual = unidades.find((u) => u.id === unidadId)

	return (
		<section>
			<PageHeader
				title="Cuenta corriente"
				description="Movimientos por UF y saldo reconstruido desde el libro de asientos."
			/>

			<div className="mt-4 space-y-4">
				{unidadesQuery.isLoading && (
					<div className="rounded-lg border bg-white">
						<SkeletonRows rows={3} />
					</div>
				)}

				{unidadesQuery.error && (
					<ErrorState
						message={`No se pudieron cargar las unidades: ${unidadesQuery.error instanceof Error ? unidadesQuery.error.message : 'error desconocido'}`}
						onRetry={() => void unidadesQuery.refetch()}
					/>
				)}

				{!unidadesQuery.isLoading && !unidadesQuery.error && unidades.length > 0 && (
					<>
						<Field label="Unidad">
							{(id) => (
								<Select id={id} value={unidadId} onChange={(e) => setUnidadId(e.target.value)}>
									{unidades.map((u) => (
										<option key={u.id} value={u.id}>
											{u.codigo} · {u.tipo}
										</option>
									))}
								</Select>
							)}
						</Field>

						{cuentaQuery.isLoading && <SkeletonRows rows={4} />}

						{cuentaQuery.error && (
							<ErrorState
								message={`No se pudo cargar la cuenta corriente: ${cuentaQuery.error instanceof Error ? cuentaQuery.error.message : 'error desconocido'}`}
								onRetry={() => void cuentaQuery.refetch()}
							/>
						)}

						{cuentaQuery.data && (
							<div className="space-y-4">
								<div className="grid gap-3 md:grid-cols-3">
									<div className="rounded-lg border bg-white p-4">
										<p className="text-xs uppercase tracking-wide text-gray-500">UF</p>
										<p className="mt-1 font-medium">{unidadActual?.codigo ?? unidadId}</p>
									</div>
									<div className="rounded-lg border bg-white p-4">
										<p className="text-xs uppercase tracking-wide text-gray-500">Saldo</p>
										<p className="mt-1 font-medium">{formatCents(cuentaQuery.data.meta.saldo_cents)}</p>
									</div>
									<div className="rounded-lg border bg-white p-4">
										<p className="text-xs uppercase tracking-wide text-gray-500">Movimientos</p>
										<p className="mt-1 font-medium">{movimientos.length}</p>
									</div>
								</div>

								{movimientos.length === 0 ? (
									<EmptyState title="Sin movimientos" description="La UF no tiene cargos ni asientos todavía." />
								) : (
									<div className="overflow-x-auto rounded-lg border bg-white">
										<table className="w-full text-left text-sm">
											<thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
												<tr>
													<th className="px-3 py-2">Fecha</th>
													<th className="px-3 py-2">Tipo</th>
													<th className="px-3 py-2">Débito</th>
													<th className="px-3 py-2">Crédito</th>
													<th className="px-3 py-2">Referencia</th>
												</tr>
											</thead>
											<tbody className="divide-y">
												{movimientos.map((m) => (
													<tr key={m.id}>
														<td className="px-3 py-2 text-gray-600">{m.fecha_efectiva}</td>
														<td className="px-3 py-2"><Badge tone={m.tipo === 'cargo' ? 'amber' : m.tipo === 'credito' ? 'green' : 'gray'}>{m.tipo}</Badge></td>
														<td className="px-3 py-2 text-gray-600">{formatCents(m.debit_cents)}</td>
														<td className="px-3 py-2 text-gray-600">{formatCents(m.credit_cents)}</td>
														<td className="px-3 py-2 text-gray-600">{m.referencia ?? '—'}</td>
													</tr>
												))}
											</tbody>
										</table>
									</div>
								)}
							</div>
						)}
					</>
				)}

				{!unidadesQuery.isLoading && !unidadesQuery.error && unidades.length === 0 && (
					<EmptyState title="Sin unidades" description="No hay unidades cargadas para mostrar cuenta corriente." />
				)}
			</div>
		</section>
	)
}

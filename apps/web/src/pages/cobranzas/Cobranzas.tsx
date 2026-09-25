import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { PermissionGate } from '@/components/ui/PermissionGate'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import { Field, Select, TextInput } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { parseCobranzasCsv } from './csv'

type Cobranza = components['schemas']['Cobranza']
type Unidad = components['schemas']['Unidad']
type CobranzaCanal = 'efectivo' | 'transferencia' | 'deposito' | 'tarjeta' | 'cajero' | 'mercadopago' | 'otros'
type CobranzaDetalle = {
	cobranza: Cobranza
	asignaciones: { charge_id: string; amount_cents: number }[]
	saldo_a_favor_cents: number
}
type MercadoPagoCobranza = {
	cobranza: Cobranza
	checkout_url: string
	provider: string
	preference_id: string
}

type CobranzasResponse = { data: Cobranza[] }
type UnidadesResponse = { data: Unidad[] }

const EMPTY_UNIDADES: Unidad[] = []
const EMPTY_COBRANZAS: Cobranza[] = []

const estadoTone: Record<string, 'gray' | 'blue' | 'amber' | 'green' | 'red'> = {
	pendiente_revision: 'amber',
	acreditado: 'green',
	rechazado: 'red',
	revertido: 'gray',
}

const canalLabels: Record<CobranzaCanal, string> = {
	efectivo: 'Efectivo',
	transferencia: 'Transferencia',
	deposito: 'Depósito',
	tarjeta: 'Tarjeta',
	cajero: 'Cajero',
	mercadopago: 'Mercado Pago',
	otros: 'Otros',
}

const validCanales = new Set<CobranzaCanal>(Object.keys(canalLabels) as CobranzaCanal[])

export function Cobranzas() {
	const { consorcioId = '' } = useParams()
	const queryClient = useQueryClient()
	const [showCreate, setShowCreate] = useState(false)
	const [showMercadoPago, setShowMercadoPago] = useState(false)
	const [showImport, setShowImport] = useState(false)
	const [selectedCobranzaId, setSelectedCobranzaId] = useState<string | null>(null)

	const unidadesQuery = useQuery({
		queryKey: ['unidades', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/unidades', {
				params: { path: { id: consorcioId } },
			})
			if (!res.data) throw new Error('No se pudieron cargar las unidades')
			return res.data as UnidadesResponse
		},
	})

	const cobranzasQuery = useQuery({
		queryKey: ['cobranzas', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/cobranzas', {
				params: { path: { id: consorcioId } },
			})
			if (!res.data) throw new Error('No se pudieron cargar las cobranzas')
			return res.data as CobranzasResponse
		},
	})

	const unidades = (unidadesQuery.data?.data ?? EMPTY_UNIDADES) as Unidad[]
	const unitByCode = new Map(unidades.map((u) => [u.codigo, u]))

	const createCobranza = useMutation({
		mutationFn: async (input: {
			unidad_id: string
			fecha: string
			canal?: CobranzaCanal
			importe: { amount_cents: number; currency: string }
			referencia?: string
		}) => {
					const res = await client.POST('/consorcios/{id}/cobranzas', {
				params: { path: { id: consorcioId } },
				headers: { 'Idempotency-Key': crypto.randomUUID() },
				body: input,
			})
			if (res.error) {
				throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo crear la cobranza')
			}
			return res.data
		},
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['cobranzas', consorcioId] })
			setShowCreate(false)
		},
	})

	const createMercadoPagoCobranza = useMutation({
		mutationFn: async (input: {
			unidad_id: string
			fecha: string
			importe: { amount_cents: number; currency: string }
			referencia?: string
			checkoutWindow: Window | null
		}) => {
			const res = await client.POST('/consorcios/{id}/cobranzas/mercado-pago', {
				params: { path: { id: consorcioId } },
				headers: { 'Idempotency-Key': crypto.randomUUID() },
				body: {
					unidad_id: input.unidad_id,
					fecha: input.fecha,
					importe: input.importe,
					...(input.referencia ? { referencia: input.referencia } : {}),
				},
			})
			if (res.error) {
				throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo iniciar el cobro con Mercado Pago')
			}
			return res.data as MercadoPagoCobranza
		},
		onSuccess: async (result) => {
			await queryClient.invalidateQueries({ queryKey: ['cobranzas', consorcioId] })
			setShowMercadoPago(false)
			if (createMercadoPagoCobranza.variables?.checkoutWindow) {
				createMercadoPagoCobranza.variables.checkoutWindow.location.href = result.checkout_url
			} else {
				window.location.assign(result.checkout_url)
			}
		},
		onError: async () => {
			createMercadoPagoCobranza.variables?.checkoutWindow?.close()
		},
	})

	const closeCreate = () => {
		createCobranza.reset()
		setShowCreate(false)
	}

	const closeImport = () => {
		importCobranza.reset()
		setShowImport(false)
	}

	const closeMercadoPago = () => {
		createMercadoPagoCobranza.reset()
		setShowMercadoPago(false)
	}

	const importCobranza = useMutation({
		mutationFn: async (file: File) => {
			const rows = parseCobranzasCsv(await file.text())
			const summary = { creadas: 0, duplicadas: 0, errores: [] as string[] }
			for (const [index, row] of rows.entries()) {
				const unidad = unitByCode.get(row.codigo_unidad)
				if (!unidad) {
					summary.errores.push(`Fila ${index + 2}: unidad ${row.codigo_unidad} no existe`)
					continue
				}
				const canal = row.canal?.trim()
				if (canal && !validCanales.has(canal as CobranzaCanal)) {
					summary.errores.push(`Fila ${index + 2}: canal ${canal} inválido`)
					continue
				}
				const res = await client.POST('/consorcios/{id}/cobranzas', {
					params: { path: { id: consorcioId } },
					headers: { 'Idempotency-Key': crypto.randomUUID() },
					body: {
						unidad_id: unidad.id,
						fecha: row.fecha,
						...(canal ? { canal: canal as CobranzaCanal } : {}),
						importe: { amount_cents: row.amount_cents, currency: 'ARS' },
						...(row.referencia ? { referencia: row.referencia } : {}),
					},
				})
				if (res.error) {
					if ((res.error as { status?: number }).status === 409) {
						summary.duplicadas += 1
						continue
					}
					summary.errores.push(`Fila ${index + 2}: ${(res.error as { detail?: string }).detail ?? 'error'}`)
					continue
				}
				summary.creadas += 1
			}
			return summary
		},
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['cobranzas', consorcioId] })
		},
	})

	const revertCobranza = useMutation({
		mutationFn: async (cobranzaId: string) => {
			const res = await client.POST('/cobranzas/{id}/revertir', {
				params: { path: { id: cobranzaId } },
				headers: { 'Idempotency-Key': crypto.randomUUID() },
			})
			if (res.error) {
				throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo revertir la cobranza')
			}
			return res.data
		},
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['cobranzas', consorcioId] })
			await queryClient.invalidateQueries({ queryKey: ['cobranza', consorcioId] })
		},
	})

	const detalleCobranzaQuery = useQuery({
		queryKey: ['cobranza', consorcioId, selectedCobranzaId],
		queryFn: async () => {
			if (!selectedCobranzaId) throw new Error('No se seleccionó una cobranza')
			const res = await client.GET('/cobranzas/{id}', {
				params: { path: { id: selectedCobranzaId } },
			})
			if (!res.data) throw new Error('No se pudo cargar el detalle de la cobranza')
			return res.data as CobranzaDetalle
		},
		enabled: selectedCobranzaId !== null,
	})

	const cobranzas = (cobranzasQuery.data?.data ?? EMPTY_COBRANZAS) as Cobranza[]
	const unitById = new Map(unidades.map((u) => [u.id, u]))
	const loadError = unidadesQuery.error ?? cobranzasQuery.error
	const detalleCobranza = detalleCobranzaQuery.data?.cobranza

	return (
		<section>
			<PageHeader
				title="Cobranzas"
				description="Cobros manuales e importación CSV con detección de referencia duplicada."
				actions={
					<PermissionGate permission="cobranzas.manage">
						<div className="flex gap-2">
							<button
								type="button"
								onClick={() => setShowImport(true)}
								className="rounded-md border px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
							>
								Importar CSV
							</button>
							<button
								type="button"
								onClick={() => setShowMercadoPago(true)}
								className="rounded-md border border-gray-900 px-3 py-1.5 text-sm text-gray-900 hover:bg-gray-900 hover:text-white"
							>
								Cobrar con Mercado Pago
							</button>
							<button
								type="button"
								onClick={() => setShowCreate(true)}
								className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
							>
								Nueva cobranza
							</button>
						</div>
					</PermissionGate>
				}
			/>

			<div className="mt-4">
				{(unidadesQuery.isLoading || cobranzasQuery.isLoading) && (
					<div className="rounded-lg border bg-white">
						<SkeletonRows rows={5} />
					</div>
				)}

				{loadError && (
					<ErrorState
						message={`No se pudieron cargar las cobranzas: ${loadError instanceof Error ? loadError.message : 'error desconocido'}`}
						onRetry={() => {
							void unidadesQuery.refetch()
							void cobranzasQuery.refetch()
						}}
					/>
				)}

				{!unidadesQuery.isLoading && !cobranzasQuery.isLoading && !unidadesQuery.error && !cobranzasQuery.error && cobranzas.length === 0 && (
					<EmptyState
						title="No hay cobranzas todavía"
						description="Cargá el primer cobro manual o importá un CSV simple."
						action={
							<PermissionGate permission="cobranzas.manage">
								<button
									type="button"
									onClick={() => setShowCreate(true)}
									className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
								>
									Nueva cobranza
								</button>
							</PermissionGate>
						}
					/>
				)}

				{cobranzas.length > 0 && (
					<div className="overflow-x-auto rounded-lg border bg-white">
						<table className="w-full text-left text-sm">
							<thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
								<tr>
									<th className="px-3 py-2">Fecha</th>
									<th className="px-3 py-2">UF</th>
							<th className="px-3 py-2">Importe</th>
							<th className="px-3 py-2">Saldo a favor</th>
							<th className="px-3 py-2">Canal</th>
							<th className="px-3 py-2">Referencia</th>
							<th className="px-3 py-2">Estado</th>
							<th className="px-3 py-2"></th>
						</tr>
					</thead>
					<tbody className="divide-y">
						{cobranzas.map((c) => (
							<tr key={c.id}>
								<td className="px-3 py-2 text-gray-600">{c.fecha}</td>
								<td className="px-3 py-2 font-medium">{unitById.get(c.unidad_id)?.codigo ?? c.unidad_id}</td>
								<td className="px-3 py-2 text-gray-600">{formatMoney(c.importe.amount_cents)}</td>
								<td className="px-3 py-2 text-gray-600">{formatMoney(c.saldo_a_favor_cents ?? 0)}</td>
								<td className="px-3 py-2 text-gray-600">{canalLabels[(c.canal ?? 'otros') as CobranzaCanal]}</td>
								<td className="px-3 py-2 text-gray-600">{c.referencia ?? '—'}</td>
								<td className="px-3 py-2"><Badge tone={estadoTone[c.estado] ?? 'gray'}>{c.estado}</Badge></td>
							<td className="px-3 py-2 text-right">
								<div className="flex justify-end gap-2">
									<button
										type="button"
										onClick={() => setSelectedCobranzaId(c.id)}
										className="rounded-md border px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-100"
									>
										Detalle
									</button>
									{c.estado === 'acreditado' && (
										<button
											type="button"
											disabled={revertCobranza.isPending}
											onClick={() => {
											if (window.confirm('Revertir esta cobranza restaurará los saldos de los cargos asociados.')) {
												revertCobranza.mutate(c.id)
											}
										}}
										className="rounded-md border px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50"
										>
											Revertir
										</button>
									)}
								</div>
								</td>
							</tr>
						))}
					</tbody>
						</table>
					</div>
				)}
			</div>

			<Modal title="Nueva cobranza" open={showCreate} onClose={closeCreate}>
				<NewCobranzaForm
					unidades={unidades}
					pending={createCobranza.isPending}
					error={createCobranza.error instanceof Error ? createCobranza.error.message : null}
					onSubmit={(input) => createCobranza.mutate(input)}
					onCancel={closeCreate}
				/>
			</Modal>

			<Modal title="Importar cobranzas CSV" open={showImport} onClose={closeImport}>
				<ImportCobranzaForm
					pending={importCobranza.isPending}
					error={importCobranza.error instanceof Error ? importCobranza.error.message : null}
					result={importCobranza.data ?? null}
					onSubmit={(file) => importCobranza.mutate(file)}
					onCancel={closeImport}
				/>
			</Modal>

			<Modal title="Cobrar con Mercado Pago" open={showMercadoPago} onClose={closeMercadoPago}>
				<MercadoPagoCobranzaForm
					unidades={unidades}
					pending={createMercadoPagoCobranza.isPending}
					error={createMercadoPagoCobranza.error instanceof Error ? createMercadoPagoCobranza.error.message : null}
					onSubmit={(input, checkoutWindow) => createMercadoPagoCobranza.mutate({ ...input, checkoutWindow })}
					onCancel={closeMercadoPago}
				/>
			</Modal>

			<Modal
				title="Detalle de cobranza"
				open={selectedCobranzaId !== null}
				onClose={() => setSelectedCobranzaId(null)}
			>
				{detalleCobranzaQuery.isLoading && <p className="text-sm text-gray-600">Cargando detalle…</p>}
				{detalleCobranzaQuery.error && (
					<p className="text-sm text-red-600" role="alert">
						{detalleCobranzaQuery.error instanceof Error ? detalleCobranzaQuery.error.message : 'No se pudo cargar el detalle'}
					</p>
				)}
				{detalleCobranza && (
					<div className="space-y-4 text-sm">
						<div className="grid gap-3 md:grid-cols-2">
							<div>
								<p className="text-xs uppercase tracking-wide text-gray-500">Unidad</p>
								<p className="font-medium">{unitById.get(detalleCobranza.unidad_id)?.codigo ?? detalleCobranza.unidad_id}</p>
							</div>
							<div>
								<p className="text-xs uppercase tracking-wide text-gray-500">Estado</p>
								<p><Badge tone={estadoTone[detalleCobranza.estado] ?? 'gray'}>{detalleCobranza.estado}</Badge></p>
							</div>
							<div>
								<p className="text-xs uppercase tracking-wide text-gray-500">Importe</p>
								<p className="font-medium">{formatMoney(detalleCobranza.importe.amount_cents)}</p>
							</div>
							<div>
								<p className="text-xs uppercase tracking-wide text-gray-500">Saldo a favor</p>
								<p className="font-medium">{formatMoney(detalleCobranza.saldo_a_favor_cents ?? 0)}</p>
							</div>
						</div>

						<div>
							<p className="text-xs uppercase tracking-wide text-gray-500">Asignaciones</p>
							{detalleCobranzaQuery.data?.asignaciones.length ? (
								<ul className="mt-2 space-y-2">
									{detalleCobranzaQuery.data.asignaciones.map((a) => (
										<li key={`${a.charge_id}-${a.amount_cents}`} className="flex items-center justify-between rounded-md border px-3 py-2">
											<span className="text-gray-600">{a.charge_id}</span>
											<span className="font-medium">{formatMoney(a.amount_cents)}</span>
										</li>
									))}
								</ul>
							) : (
								<p className="mt-2 text-gray-600">Sin asignaciones.</p>
							)}
						</div>

						{detalleCobranza.estado === 'acreditado' && (
							<div className="flex justify-end gap-2">
								<button
									type="button"
									onClick={() => window.open(`/api/v1/cobranzas/${detalleCobranza.id}/recibo.pdf`, '_blank', 'noopener,noreferrer')}
									className="rounded-md border px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
								>
									Descargar recibo
								</button>
								<button
									type="button"
									onClick={() => {
									if (window.confirm('Revertir esta cobranza restaurará los saldos de los cargos asociados.')) {
										revertCobranza.mutate(detalleCobranza.id)
									}
								}}
									disabled={revertCobranza.isPending}
									className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
								>
									Revertir
								</button>
							</div>
						)}
					</div>
				)}
			</Modal>
		</section>
	)
}

function NewCobranzaForm({
	unidades,
	pending,
	error,
	onSubmit,
	onCancel,
}: {
	unidades: Unidad[]
	pending: boolean
	error: string | null
			onSubmit: (input: {
			unidad_id: string
			fecha: string
			canal?: CobranzaCanal
			importe: { amount_cents: number; currency: string }
			referencia?: string
		}) => void
	onCancel: () => void
}) {
	const [unidadId, setUnidadId] = useState(unidades[0]?.id ?? '')
	const [fecha, setFecha] = useState('')
	const [canal, setCanal] = useState<CobranzaCanal>('otros')
	const [importe, setImporte] = useState('')
	const [referencia, setReferencia] = useState('')

	useEffect(() => {
		if (!unidadId && unidades[0]) {
			setUnidadId(unidades[0].id)
		}
	}, [unidadId, unidades])

	const importeCents = Number.parseInt(importe, 10)
	const canSubmit = unidadId && fecha && Number.isFinite(importeCents) && importeCents > 0

	return (
		<form
			className="space-y-4"
			onSubmit={(e) => {
				e.preventDefault()
				if (!canSubmit) return
				onSubmit({
					unidad_id: unidadId,
					fecha,
					canal: canal || undefined,
					importe: { amount_cents: importeCents, currency: 'ARS' },
					...(referencia.trim() ? { referencia: referencia.trim() } : {}),
				})
			}}
		>
			<Field label="Unidad" required>
				{(id) => (
					<Select id={id} value={unidadId} onChange={(e) => setUnidadId(e.target.value)} required>
						{unidades.map((u) => (
							<option key={u.id} value={u.id}>
								{u.codigo} · {u.tipo}
							</option>
						))}
					</Select>
				)}
			</Field>

			<div className="grid gap-3 md:grid-cols-3">
				<Field label="Fecha" required>
					{(id) => <TextInput id={id} type="date" value={fecha} onChange={(e) => setFecha(e.target.value)} required />}
				</Field>
				<Field label="Canal">
					{(id) => (
							<Select id={id} value={canal} onChange={(e) => setCanal(e.target.value as CobranzaCanal)}>
								<option value="otros">Otros</option>
								<option value="efectivo">Efectivo</option>
								<option value="transferencia">Transferencia</option>
								<option value="deposito">Depósito</option>
								<option value="tarjeta">Tarjeta</option>
								<option value="cajero">Cajero</option>
								<option value="mercadopago">Mercado Pago</option>
							</Select>
						)}
					</Field>
				<Field label="Importe (centavos)" required hint="Se guarda como Money.amount_cents.">
					{(id) => (
						<TextInput
							id={id}
							type="number"
							min="1"
							step="1"
							value={importe}
							onChange={(e) => setImporte(e.target.value)}
							required
						/>
					)}
				</Field>
			</div>

			<Field label="Referencia">
				{(id) => <TextInput id={id} value={referencia} onChange={(e) => setReferencia(e.target.value)} />}
			</Field>

			{error && <p className="text-sm text-red-600" role="alert">{error}</p>}

			<div className="flex justify-end gap-2">
				<button type="button" onClick={onCancel} className="rounded-md border px-3 py-1.5 text-sm">
					Cancelar
				</button>
				<button type="submit" disabled={pending || !canSubmit} className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40">
					{pending ? 'Creando…' : 'Crear cobranza'}
				</button>
			</div>
		</form>
	)
}

function ImportCobranzaForm({
	pending,
	error,
	result,
	onSubmit,
	onCancel,
}: {
	pending: boolean
	error: string | null
	result: { creadas: number; duplicadas: number; errores: string[] } | null
	onSubmit: (file: File) => void
	onCancel: () => void
}) {
	const [file, setFile] = useState<File | null>(null)
	return (
		<form
			className="space-y-4"
			onSubmit={(e) => {
				e.preventDefault()
				if (!file) return
				onSubmit(file)
			}}
		>
			<p className="text-sm text-gray-600">
				Encabezado esperado: <code className="rounded bg-gray-100 px-1 py-0.5 text-xs">codigo_unidad,fecha,canal,amount_cents,referencia</code>
			</p>
			<input
				type="file"
				accept=".csv,text/csv"
				className="block w-full text-sm text-gray-600 file:mr-3 file:rounded-md file:border-0 file:bg-gray-100 file:px-3 file:py-1.5 file:text-sm file:text-gray-700"
				onChange={(e) => setFile(e.target.files?.[0] ?? null)}
			/>
			{result && (
				<p className="text-sm text-gray-600">
					Creó {result.creadas}, duplicó {result.duplicadas}
					{result.errores.length > 0 ? `, ${result.errores.length} errores` : ''}.
				</p>
			)}
			{error && <p className="text-sm text-red-600" role="alert">{error}</p>}
			<div className="flex justify-end gap-2">
				<button type="button" onClick={onCancel} className="rounded-md border px-3 py-1.5 text-sm">
					Cancelar
				</button>
				<button type="submit" disabled={pending || !file} className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40">
					{pending ? 'Importando…' : 'Importar'}
				</button>
			</div>
		</form>
	)
}

function MercadoPagoCobranzaForm({
	unidades,
	pending,
	error,
	onSubmit,
	onCancel,
}: {
	unidades: Unidad[]
	pending: boolean
	error: string | null
	onSubmit: (input: {
		unidad_id: string
		fecha: string
		importe: { amount_cents: number; currency: string }
		referencia?: string
	}, checkoutWindow: Window | null) => void
	onCancel: () => void
}) {
	const [unidadId, setUnidadId] = useState(unidades[0]?.id ?? '')
	const [fecha, setFecha] = useState('')
	const [importe, setImporte] = useState('')
	const [referencia, setReferencia] = useState('')

	useEffect(() => {
		if (!unidadId && unidades[0]) {
			setUnidadId(unidades[0].id)
		}
	}, [unidadId, unidades])

	const importeCents = Number.parseInt(importe, 10)
	const canSubmit = unidadId && fecha && Number.isFinite(importeCents) && importeCents > 0

	return (
		<form
			className="space-y-4"
			onSubmit={(e) => {
				e.preventDefault()
				if (!canSubmit) return
				const checkoutWindow = window.open('', '_blank', 'noopener,noreferrer')
				onSubmit({
					unidad_id: unidadId,
					fecha,
					importe: { amount_cents: importeCents, currency: 'ARS' },
					...(referencia.trim() ? { referencia: referencia.trim() } : {}),
				}, checkoutWindow)
			}}
		>
			<p className="text-sm text-gray-600">
				Se crea la cobranza y se abre el checkout de Mercado Pago en una pestaña nueva.
			</p>
			<Field label="Unidad" required>
				{(id) => (
					<Select id={id} value={unidadId} onChange={(e) => setUnidadId(e.target.value)} required>
						{unidades.map((u) => (
							<option key={u.id} value={u.id}>
								{u.codigo} · {u.tipo}
							</option>
						))}
					</Select>
				)}
			</Field>

			<div className="grid gap-3 md:grid-cols-2">
				<Field label="Fecha" required>
					{(id) => <TextInput id={id} type="date" value={fecha} onChange={(e) => setFecha(e.target.value)} required />}
				</Field>
				<Field label="Importe (centavos)" required hint="Se guarda como Money.amount_cents.">
					{(id) => (
						<TextInput
							id={id}
							type="number"
							min="1"
							step="1"
							value={importe}
							onChange={(e) => setImporte(e.target.value)}
							required
						/>
					)}
				</Field>
			</div>

			<Field label="Referencia">
				{(id) => <TextInput id={id} value={referencia} onChange={(e) => setReferencia(e.target.value)} />}
			</Field>

			{error && <p className="text-sm text-red-600" role="alert">{error}</p>}

			<div className="flex justify-end gap-2">
				<button type="button" onClick={onCancel} className="rounded-md border px-3 py-1.5 text-sm">
					Cancelar
				</button>
				<button type="submit" disabled={pending || !canSubmit} className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40">
					{pending ? 'Abriendo checkout…' : 'Cobrar con Mercado Pago'}
				</button>
			</div>
		</form>
	)
}

function formatMoney(cents: number): string {
	return new Intl.NumberFormat('es-AR', { style: 'currency', currency: 'ARS' }).format(cents / 100)
}

import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { client } from '@/api/client'
import { Badge, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import { readMetricValue } from './metrics'

type HealthResponse = { status: string }

type MetricCard = {
	label: string
	metric: string
	description: string
	tone: 'gray' | 'green' | 'amber' | 'red' | 'blue'
}

const metricCards: MetricCard[] = [
	{ label: 'Requests totales', metric: 'http_requests_total', description: 'Requests servidas por el API.', tone: 'blue' },
	{ label: 'Fallos de login', metric: 'login_failures_total', description: 'Intentos rechazados y errores de autenticación.', tone: 'amber' },
	{ label: 'Outbox procesado', metric: 'outbox_processed_total', description: 'Eventos entregados por el worker.', tone: 'green' },
	{ label: 'Outbox fallido', metric: 'outbox_failed_total', description: 'Eventos que quedaron en estado fallido.', tone: 'red' },
]

function formatMetric(value: number | null): string {
	if (value === null) return '—'
	if (Number.isInteger(value)) return value.toLocaleString('es-AR')
	return value.toFixed(2)
}

export function Observabilidad() {
	const query = useQuery({
		queryKey: ['observabilidad'],
		refetchInterval: 30_000,
		queryFn: async () => {
			const [health, metricsResponse] = await Promise.all([
				client.GET('/health'),
				fetch('/metrics'),
			])
			if (!health.data) throw new Error('No se pudo cargar la salud del API')
			if (!metricsResponse.ok) throw new Error('No se pudieron cargar las métricas')
			return {
				health: health.data as HealthResponse,
				metricsText: await metricsResponse.text(),
				syncedAt: new Date().toISOString(),
			}
		},
	})

	const summary = useMemo(() => {
		const metricsText = query.data?.metricsText ?? ''
		return metricCards.map((card) => ({
			...card,
			value: readMetricValue(metricsText, card.metric),
		}))
	}, [query.data?.metricsText])

	const health = query.data?.health
	const apiOk = health?.status === 'ok'
	const syncedAt = query.data?.syncedAt ? new Date(query.data.syncedAt) : null
	const syncedLabel = syncedAt
		? new Intl.DateTimeFormat('es-AR', {
			dateStyle: 'short',
			timeStyle: 'medium',
			timeZone: 'America/Argentina/Buenos_Aires',
		}).format(syncedAt)
		: '—'

	return (
		<section className="mx-auto flex max-w-6xl flex-col gap-6">
			<PageHeader
				title="Observabilidad"
				description="Sala de control para salud del API, métricas Prometheus y trazas del worker."
				actions={
					<div className="flex items-center gap-2 text-sm">
						{query.data && <span className="text-xs uppercase tracking-[0.2em] text-gray-400">Sincronizado {syncedLabel}</span>}
						<Badge tone={apiOk ? 'green' : 'red'}>{apiOk ? 'API healthy' : 'API degraded'}</Badge>
						<button className="rounded-md border px-3 py-1.5 text-gray-700 hover:bg-gray-50" type="button" onClick={() => void query.refetch()}>
							Actualizar
						</button>
						<a className="rounded-md border px-3 py-1.5 text-gray-700 hover:bg-gray-50" href="/healthz" target="_blank" rel="noreferrer">
							/healthz
						</a>
						<a className="rounded-md border px-3 py-1.5 text-gray-700 hover:bg-gray-50" href="/metrics" target="_blank" rel="noreferrer">
							/metrics
						</a>
					</div>
				}
			/>

			{query.isLoading && <SkeletonRows rows={5} />}

			{query.error && (
				<ErrorState
					message={`No se pudo cargar la observabilidad: ${query.error instanceof Error ? query.error.message : 'error desconocido'}`}
					onRetry={() => void query.refetch()}
				/>
			)}

			{query.data && (
				<>
					<div className="grid gap-4 xl:grid-cols-[1.4fr_0.6fr]">
						<div className="rounded-3xl border border-gray-200 bg-slate-950 p-5 text-slate-100 shadow-sm">
							<div className="flex items-center justify-between gap-3 border-b border-white/10 pb-4">
								<div>
									<p className="text-xs uppercase tracking-[0.3em] text-slate-400">Control strip</p>
									<p className="mt-1 text-lg font-medium text-white">Señales vivas del sistema</p>
									<p className="mt-2 text-sm text-slate-300">La sala se refresca cada 30 segundos.</p>
								</div>
								<Badge tone={apiOk ? 'green' : 'red'}>{query.data.health.status}</Badge>
							</div>
							<div className="mt-4 grid gap-3 sm:grid-cols-3">
								<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
									<p className="text-xs uppercase tracking-wide text-slate-400">API</p>
									<p className="mt-2 text-2xl font-semibold text-white">{query.data.health.status}</p>
									<p className="mt-1 text-sm text-slate-300">`GET /health`</p>
								</div>
								<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
									<p className="text-xs uppercase tracking-wide text-slate-400">Trace</p>
									<p className="mt-2 text-2xl font-semibold text-white">stdout</p>
									<p className="mt-1 text-sm text-slate-300">OTel export local</p>
								</div>
								<div className="rounded-2xl border border-white/10 bg-white/5 p-4">
									<p className="text-xs uppercase tracking-wide text-slate-400">Logs</p>
									<p className="mt-2 text-2xl font-semibold text-white">slog</p>
									<p className="mt-1 text-sm text-slate-300">JSON con request_id</p>
								</div>
							</div>
						</div>

						<div className="rounded-3xl border bg-white p-5 shadow-sm">
							<p className="text-xs uppercase tracking-[0.3em] text-gray-400">Notas operativas</p>
							<ul className="mt-4 space-y-3 text-sm text-gray-600">
								<li>• `request_id`, `trace_id` y `span_id` ya viajan en logs del API.</li>
								<li>• `/metrics` expone requests, login y outbox.</li>
								<li>• El worker exporta spans y cuenta eventos procesados.</li>
							</ul>
						</div>
					</div>

					<div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
						{summary.map((card) => (
							<div key={card.metric} className="rounded-2xl border bg-white p-5 shadow-sm">
								<div className="flex items-center justify-between gap-3">
									<p className="text-xs font-medium uppercase tracking-wide text-gray-400">{card.label}</p>
									<Badge tone={card.tone}>{card.metric}</Badge>
								</div>
								<p className="mt-3 text-3xl font-semibold text-gray-900">{formatMetric(card.value)}</p>
								<p className="mt-2 text-sm text-gray-500">{card.description}</p>
							</div>
						))}
					</div>

					<div className="grid gap-4 lg:grid-cols-2">
						<div className="rounded-2xl border bg-white p-5 shadow-sm">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Worker / outbox</p>
							<div className="mt-3 grid gap-3 sm:grid-cols-2">
								<div className="rounded-xl bg-gray-50 p-4">
									<p className="text-xs uppercase tracking-wide text-gray-400">Procesados</p>
									<p className="mt-2 text-2xl font-semibold text-gray-900">{formatMetric(readMetricValue(query.data.metricsText, 'outbox_processed_total'))}</p>
								</div>
								<div className="rounded-xl bg-gray-50 p-4">
									<p className="text-xs uppercase tracking-wide text-gray-400">Fallidos</p>
									<p className="mt-2 text-2xl font-semibold text-gray-900">{formatMetric(readMetricValue(query.data.metricsText, 'outbox_failed_total'))}</p>
								</div>
							</div>
							<p className="mt-3 text-sm text-gray-500">El worker escribe spans de batch, evento y envío de comunicados.</p>
						</div>

						<div className="rounded-2xl border bg-white p-5 shadow-sm">
							<p className="text-xs font-medium uppercase tracking-wide text-gray-400">Formatos y consultas</p>
							<pre className="mt-3 overflow-x-auto rounded-xl bg-gray-950 p-4 text-xs leading-6 text-gray-100">
{`Log JSON esperado
{
  "level": "INFO",
  "msg": "http",
  "request_id": "...",
  "trace_id": "...",
  "span_id": "..."
}

PromQL útil
rate(http_requests_total{status=~"5.."}[5m])
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, route))`}
							</pre>
						</div>
					</div>
				</>
			)}
		</section>
	)
}

import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { PermissionGate } from '@/components/ui/PermissionGate'
import { Badge, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import { Field, TextInput } from '@/components/ui/Field'

type Liquidacion = components['schemas']['Liquidacion']

const estadoTone: Record<string, 'gray' | 'blue' | 'amber' | 'green' | 'red'> = {
  borrador: 'gray',
  calculada: 'blue',
  confirmada: 'amber',
  publicada: 'green',
  cerrada: 'gray',
  anulada: 'red',
}

export function LiquidacionWizard() {
  const { consorcioId = '', liquidacionId = '' } = useParams()
  const queryClient = useQueryClient()
  const [vencimiento1, setVencimiento1] = useState('')
  const [vencimiento2, setVencimiento2] = useState('')
  const [motivo, setMotivo] = useState('')

  const queryKey = ['liquidacion', consorcioId, liquidacionId]

  const { data, error, isLoading, refetch } = useQuery({
    queryKey,
    queryFn: async () => {
      const res = await client.GET('/consorcios/{consorcioId}/liquidaciones/{liquidacionId}', {
        params: { path: { consorcioId, liquidacionId } },
      })
      if (!res.data) throw new Error('No se pudo cargar la liquidación')
      return res.data as Liquidacion
    },
  })

  useEffect(() => {
    if (!data) return
    setVencimiento1(data.vencimiento_1)
    setVencimiento2(data.vencimiento_2 ?? '')
  }, [data])

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['liquidaciones', consorcioId] }),
      queryClient.invalidateQueries({ queryKey }),
    ])
  }

  const patch = useMutation({
    mutationFn: async () => {
      if (!data) throw new Error('No hay liquidación cargada')
      const res = await client.PATCH('/consorcios/{consorcioId}/liquidaciones/{liquidacionId}', {
        params: {
          path: { consorcioId, liquidacionId },
          header: { 'If-Match': String(data.version) },
        },
        body: {
          vencimiento_1: vencimiento1,
          vencimiento_2: vencimiento2 ? vencimiento2 : null,
        },
      })
      if (res.error) throw new Error(res.error.detail ?? 'No se pudieron actualizar los vencimientos')
      return res.data as Liquidacion
    },
    onSuccess: refresh,
  })

  const calcular = useMutation({
    mutationFn: async () => runTransition('/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/calcular'),
    onSuccess: refresh,
  })

  const confirmar = useMutation({
    mutationFn: async () => runTransition('/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/confirmar'),
    onSuccess: refresh,
  })

  const publicar = useMutation({
    mutationFn: async () => runTransition('/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/publicar'),
    onSuccess: refresh,
  })

  const anular = useMutation({
    mutationFn: async () => {
      if (!data) throw new Error('No hay liquidación cargada')
      const res = await client.POST('/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/anular', {
        params: {
          path: { consorcioId, liquidacionId },
          header: { 'If-Match': String(data.version) },
        },
        body: motivo ? { motivo } : {},
      })
      if (res.error) throw new Error(res.error.detail ?? 'No se pudo anular la liquidación')
      return res.data as Liquidacion
    },
    onSuccess: refresh,
  })

  const isBusy = patch.isPending || calcular.isPending || confirmar.isPending || publicar.isPending || anular.isPending

  const actions = useMemo(() => {
    if (!data) return null
    const canCalculate = data.estado === 'borrador' || data.estado === 'calculada'
    const canConfirm = data.estado === 'calculada'
    const canPublish = data.estado === 'confirmada'
    const canAnnul = data.estado === 'confirmada' || data.estado === 'publicada'
    return { canCalculate, canConfirm, canPublish, canAnnul }
  }, [data])

  async function runTransition(
    path:
      | '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/calcular'
      | '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/confirmar'
      | '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/publicar',
  ) {
    if (!data) throw new Error('No hay liquidación cargada')
    const res = await client.POST(path, {
      params: {
        path: { consorcioId, liquidacionId },
        header: {
          'If-Match': String(data.version),
          'Idempotency-Key': crypto.randomUUID(),
        },
      },
    })
    if (res.error) throw new Error(res.error.detail ?? 'No se pudo ejecutar la transición')
    return res.data as Liquidacion
  }

  if (isLoading) {
    return (
      <section aria-busy="true">
        <PageHeader title="Liquidación" description="Cargando detalle…" />
        <div className="mt-4 rounded-lg border bg-white">
          <SkeletonRows rows={4} />
        </div>
      </section>
    )
  }

  if (error) {
    return (
      <section>
        <PageHeader title="Liquidación" description="Detalle y acciones" />
        <ErrorState
          message={`No se pudo cargar la liquidación: ${error instanceof Error ? error.message : 'error desconocido'}`}
          onRetry={() => void refetch()}
        />
      </section>
    )
  }

  if (!data) return null

  return (
    <section>
      <PageHeader
        title={`Liquidación ${data.periodo}`}
        description={`Detalle de estado, vencimientos y acciones.`}
      />

      <div className="mt-4 flex flex-wrap gap-2">
        <Badge tone={estadoTone[data.estado] ?? 'gray'}>{data.estado}</Badge>
        <Badge tone="gray">v{data.version}</Badge>
        <Badge tone="blue">UFs {data.unidades_alcanzadas}</Badge>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-3">
        <StatCard label="Total gastos" value={formatCents(data.total_gastos_cents)} />
        <StatCard label="Total distribuido" value={formatCents(data.total_distribuido_cents)} />
        <StatCard label="Vencimiento 1" value={data.vencimiento_1} />
      </div>

      <div className="mt-6 rounded-lg border bg-white p-4">
        <h2 className="text-sm font-medium text-gray-900">Vencimientos</h2>
        <form
          className="mt-4 grid gap-3 md:grid-cols-3"
          onSubmit={(e) => {
            e.preventDefault()
            patch.mutate()
          }}
        >
          <Field label="Vencimiento 1" required>
            {(id) => (
              <TextInput id={id} type="date" value={vencimiento1} onChange={(e) => setVencimiento1(e.target.value)} required />
            )}
          </Field>
          <Field label="Vencimiento 2">
            {(id) => (
              <TextInput id={id} type="date" value={vencimiento2} onChange={(e) => setVencimiento2(e.target.value)} />
            )}
          </Field>
          <div className="flex items-end gap-2">
            <button
              type="submit"
              disabled={isBusy || data.estado !== 'borrador'}
              className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
            >
              {patch.isPending ? 'Guardando…' : 'Guardar cambios'}
            </button>
            <span className="text-xs text-gray-400">Solo editable en borrador.</span>
          </div>
        </form>
        {patch.error && <p className="mt-3 text-sm text-red-600">{patch.error.message}</p>}
      </div>

      <div className="mt-6 grid gap-4 lg:grid-cols-2">
        <div className="rounded-lg border bg-white p-4">
          <h2 className="text-sm font-medium text-gray-900">Acciones</h2>
          <div className="mt-4 flex flex-wrap gap-2">
            <PermissionGate permission="expensas.create">
              <button
                type="button"
                disabled={!actions?.canCalculate || isBusy}
                onClick={() => calcular.mutate()}
                className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
              >
                {calcular.isPending ? 'Calculando…' : 'Calcular'}
              </button>
            </PermissionGate>
            <PermissionGate permission="expensas.confirm">
              <button
                type="button"
                disabled={!actions?.canConfirm || isBusy}
                onClick={() => confirmar.mutate()}
                className="rounded-md border px-3 py-1.5 text-sm disabled:opacity-40"
              >
                {confirmar.isPending ? 'Confirmando…' : 'Confirmar'}
              </button>
            </PermissionGate>
            <PermissionGate permission="expensas.publish">
              <button
                type="button"
                disabled={!actions?.canPublish || isBusy}
                onClick={() => publicar.mutate()}
                className="rounded-md border px-3 py-1.5 text-sm disabled:opacity-40"
              >
                {publicar.isPending ? 'Publicando…' : 'Publicar'}
              </button>
            </PermissionGate>
          </div>
          {(calcular.error || confirmar.error || publicar.error) && (
            <p className="mt-3 text-sm text-red-600">
              {(calcular.error || confirmar.error || publicar.error)?.message}
            </p>
          )}
        </div>

        <div className="rounded-lg border bg-white p-4">
          <h2 className="text-sm font-medium text-gray-900">Anulación</h2>
          <p className="mt-1 text-sm text-gray-500">
            Disponible cuando la liquidación todavía tiene cargos reversibles.
          </p>
          <form
            className="mt-4 space-y-3"
            onSubmit={(e) => {
              e.preventDefault()
              anular.mutate()
            }}
          >
            <Field label="Motivo" required hint="Se audita en el evento de consorcio.">
              {(id) => (
                <TextInput id={id} value={motivo} onChange={(e) => setMotivo(e.target.value)} placeholder="Motivo de la anulación" />
              )}
            </Field>
            <button
              type="submit"
              disabled={!actions?.canAnnul || isBusy || motivo.trim().length === 0}
              className="rounded-md border border-red-300 px-3 py-1.5 text-sm text-red-700 disabled:opacity-40"
            >
              {anular.isPending ? 'Anulando…' : 'Anular liquidación'}
            </button>
            {anular.error && <p className="text-sm text-red-600">{anular.error.message}</p>}
          </form>
        </div>
      </div>
    </section>
  )
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-xs font-medium uppercase tracking-wide text-gray-400">{label}</p>
      <p className="mt-1 text-lg font-medium text-gray-900">{value}</p>
    </div>
  )
}

function formatCents(cents: number) {
  return new Intl.NumberFormat('es-AR', {
    style: 'currency',
    currency: 'ARS',
    maximumFractionDigits: 2,
  }).format(cents / 100)
}

import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { PermissionGate } from '@/components/ui/PermissionGate'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'
import { Field, Select, TextInput } from '@/components/ui/Field'

type Liquidacion = components['schemas']['Liquidacion']
type LiquidacionInput = components['schemas']['LiquidacionInput']

type LiquidacionListResponse = {
  data: Liquidacion[]
}

const estadoTone: Record<string, 'gray' | 'blue' | 'amber' | 'green' | 'red'> = {
  borrador: 'gray',
  calculada: 'blue',
  confirmada: 'amber',
  publicada: 'green',
  cerrada: 'gray',
  anulada: 'red',
}

export function Liquidaciones() {
  const { consorcioId = '' } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [periodo, setPeriodo] = useState('')
  const [estado, setEstado] = useState('')
  const [showCreate, setShowCreate] = useState(false)

  const { data, error, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['liquidaciones', consorcioId, periodo, estado],
    queryFn: async () => {
      const res = await client.GET('/consorcios/{id}/liquidaciones', {
        params: {
          path: { id: consorcioId },
          query: {
            periodo: periodo || undefined,
            estado: estado || undefined,
            page: 1,
            per_page: 50,
          },
        },
      })
      if (!res.data) throw new Error('No se pudieron cargar las liquidaciones')
      return res.data as LiquidacionListResponse
    },
  })

  const create = useMutation({
    mutationFn: async (input: LiquidacionInput) => {
      const res = await client.POST('/consorcios/{id}/liquidaciones', {
        params: { path: { id: consorcioId } },
        body: input,
      })
      if (res.error) throw new Error(res.error.detail ?? 'No se pudo crear la liquidación')
      return res.data as Liquidacion
    },
    onSuccess: async (liq) => {
      await queryClient.invalidateQueries({ queryKey: ['liquidaciones', consorcioId] })
      setShowCreate(false)
      navigate(`/app/consorcios/${consorcioId}/liquidaciones/${liq.id}`)
    },
  })

  const liquidaciones = data?.data ?? []

  return (
    <section>
      <PageHeader
        title="Liquidaciones"
        description="Historial, cálculo y publicación de expensas del consorcio."
      />

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <label className="text-sm text-gray-500">
          Período
          <TextInput
            type="search"
            value={periodo}
            onChange={(e) => setPeriodo(e.target.value)}
            placeholder="YYYYMM"
            className="ml-2 inline-block w-auto"
          />
        </label>
        <label className="text-sm text-gray-500">
          Estado
          <Select value={estado} onChange={(e) => setEstado(e.target.value)} className="ml-2 inline-block w-auto">
            <option value="">Todos</option>
            <option value="borrador">Borrador</option>
            <option value="calculada">Calculada</option>
            <option value="confirmada">Confirmada</option>
            <option value="publicada">Publicada</option>
            <option value="cerrada">Cerrada</option>
            <option value="anulada">Anulada</option>
          </Select>
        </label>
        {isFetching && <span className="text-sm text-gray-400">Actualizando…</span>}
        <PermissionGate permission="expensas.create">
          <button
            type="button"
            onClick={() => setShowCreate((v) => !v)}
            className="rounded-md border px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
          >
            {showCreate ? 'Cerrar' : 'Nueva liquidación'}
          </button>
        </PermissionGate>
      </div>

      <div className="mt-4">
        {showCreate && (
          <PermissionGate permission="expensas.create">
            <CreateLiquidacionForm
              pending={create.isPending}
              error={create.error instanceof Error ? create.error.message : null}
              onSubmit={(input) => create.mutate(input)}
              onCancel={() => setShowCreate(false)}
            />
          </PermissionGate>
        )}

        {isLoading && (
          <div className="rounded-lg border bg-white">
            <SkeletonRows rows={5} />
          </div>
        )}

        {error && (
          <ErrorState
            message={`No se pudieron cargar las liquidaciones: ${error instanceof Error ? error.message : 'error desconocido'}`}
            onRetry={() => void refetch()}
          />
        )}

        {!isLoading && !error && liquidaciones.length === 0 && (
          <EmptyState
            title="No hay liquidaciones todavía"
            description="Creá la primera y empezá a calcular expensas para este consorcio."
            action={
              <PermissionGate permission="expensas.create">
                <button
                  type="button"
                  onClick={() => setShowCreate(true)}
                  className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
                >
                  Nueva liquidación
                </button>
              </PermissionGate>
            }
          />
        )}

        {!isLoading && !error && liquidaciones.length > 0 && (
          <div className="overflow-x-auto rounded-lg border bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="px-3 py-2">Período</th>
                  <th className="px-3 py-2">Estado</th>
                  <th className="px-3 py-2">Vencimiento 1</th>
                  <th className="px-3 py-2">Gastos</th>
                  <th className="px-3 py-2">Distribuido</th>
                  <th className="px-3 py-2">UFs</th>
                  <th className="px-3 py-2">Versión</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {liquidaciones.map((liq) => (
                  <tr key={liq.id} className="hover:bg-gray-50">
                    <td className="px-3 py-2 font-medium">
                      <a
                        href={`/app/consorcios/${consorcioId}/liquidaciones/${liq.id}`}
                        className="text-gray-900 underline-offset-2 hover:underline"
                      >
                        {liq.periodo}
                      </a>
                    </td>
                    <td className="px-3 py-2">
                      <Badge tone={estadoTone[liq.estado] ?? 'gray'}>{liq.estado}</Badge>
                    </td>
                    <td className="px-3 py-2 text-gray-600">{liq.vencimiento_1}</td>
                    <td className="px-3 py-2 text-gray-600">{formatCents(liq.total_gastos_cents)}</td>
                    <td className="px-3 py-2 text-gray-600">{formatCents(liq.total_distribuido_cents)}</td>
                    <td className="px-3 py-2 text-gray-600">{liq.unidades_alcanzadas}</td>
                    <td className="px-3 py-2 text-gray-600">v{liq.version}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </section>
  )
}

function CreateLiquidacionForm({
  pending,
  error,
  onSubmit,
  onCancel,
}: {
  pending: boolean
  error: string | null
  onSubmit: (input: LiquidacionInput) => void
  onCancel: () => void
}) {
  const [periodo, setPeriodo] = useState('')
  const [vencimiento1, setVencimiento1] = useState('')
  const [vencimiento2, setVencimiento2] = useState('')

  return (
    <form
      className="mb-4 space-y-4 rounded-lg border bg-white p-4"
      onSubmit={(e) => {
        e.preventDefault()
        onSubmit({
          periodo: periodo.trim(),
          vencimiento_1: vencimiento1,
          ...(vencimiento2 ? { vencimiento_2: vencimiento2 } : {}),
        })
      }}
    >
      <div className="grid gap-3 md:grid-cols-3">
        <Field label="Período" required hint="YYYYMM">
          {(id) => (
            <TextInput
              id={id}
              value={periodo}
              onChange={(e) => setPeriodo(e.target.value)}
              placeholder="202609"
              required
            />
          )}
        </Field>
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
      </div>

      {error && <p className="text-sm text-red-600" role="alert">{error}</p>}

      <div className="flex items-center justify-end gap-2">
        <button type="button" onClick={onCancel} className="rounded-md border px-3 py-1.5 text-sm">
          Cancelar
        </button>
        <button
          type="submit"
          disabled={pending || !periodo.trim() || !vencimiento1}
          className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
        >
          {pending ? 'Creando…' : 'Crear liquidación'}
        </button>
      </div>
    </form>
  )
}

function formatCents(cents: number) {
  return new Intl.NumberFormat('es-AR', {
    style: 'currency',
    currency: 'ARS',
    maximumFractionDigits: 2,
  }).format(cents / 100)
}

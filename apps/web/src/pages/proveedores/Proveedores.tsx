import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { PermissionGate } from '@/components/ui/PermissionGate'
import { Modal } from '@/components/ui/Modal'
import { Field, TextInput } from '@/components/ui/Field'
import { EmptyState, ErrorState, EstadoBadge, PageHeader, SkeletonRows } from '@/components/ui/primitives'

type Proveedor = components['schemas']['Proveedor']

const CUIT_RE = /^[0-9]{11}$/

export function Proveedores() {
  const { consorcioId = '' } = useParams()
  const queryClient = useQueryClient()
  const [q, setQ] = useState('')
  const [showCreate, setShowCreate] = useState(false)

  const { data, error, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['proveedores', consorcioId],
    queryFn: async () => {
      const res = await client.GET('/consorcios/{id}/proveedores', {
        params: { path: { id: consorcioId } },
      })
      if (!res.data) throw new Error('No se pudieron cargar los proveedores')
      return res.data
    },
  })

  const proveedores = useMemo(() => {
    const all = (data?.data ?? []) as Proveedor[]
    if (!q.trim()) return all
    const needle = q.trim().toLowerCase()
    return all.filter(
      (p) => p.razon_social.toLowerCase().includes(needle) || p.cuit.includes(needle),
    )
  }, [data, q])

  const create = useMutation({
    mutationFn: async (input: { cuit: string; razon_social: string }) => {
      const res = await client.POST('/consorcios/{id}/proveedores', {
        params: { path: { id: consorcioId } },
        body: input,
      })
      if (res.error) {
        throw new Error(
          (res.error as { detail?: string }).detail ?? 'No se pudo crear el proveedor',
        )
      }
      return res.data
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['proveedores'] })
      setShowCreate(false)
    },
  })

  return (
    <section>
      <PageHeader
        title="Proveedores"
        description="Proveedores del consorcio para gastos y comprobantes."
        actions={
          <PermissionGate permission="proveedores.manage">
            <button
              type="button"
              onClick={() => setShowCreate(true)}
              className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
            >
              Nuevo proveedor
            </button>
          </PermissionGate>
        }
      />

      <div className="mt-4 flex items-center gap-3">
        <label className="text-sm text-gray-500">
          Buscar
          <TextInput
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Razón social o CUIT"
            className="ml-2 inline-block w-auto"
          />
        </label>
        {isFetching && <span className="text-sm text-gray-400">Actualizando…</span>}
      </div>

      <div className="mt-4">
        {isLoading && (
          <div className="rounded-lg border bg-white">
            <SkeletonRows rows={5} />
          </div>
        )}

        {error && (
          <ErrorState
            message={`No se pudieron cargar los proveedores: ${error instanceof Error ? error.message : 'error desconocido'}`}
            onRetry={() => void refetch()}
          />
        )}

        {!isLoading && !error && proveedores.length === 0 && q.trim() && (
          <EmptyState title="Sin resultados" description="Probá con otro término de búsqueda." />
        )}

        {!isLoading && !error && proveedores.length === 0 && !q.trim() && (
          <EmptyState
            title="No hay proveedores todavía"
            description="Cargá el primer proveedor del consorcio."
            action={
              <PermissionGate permission="proveedores.manage">
                <button
                  type="button"
                  onClick={() => setShowCreate(true)}
                  className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
                >
                  Nuevo proveedor
                </button>
              </PermissionGate>
            }
          />
        )}

        {!isLoading && !error && proveedores.length > 0 && (
          <div className="overflow-x-auto rounded-lg border bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="px-3 py-2">Razón social</th>
                  <th className="px-3 py-2">CUIT</th>
                  <th className="px-3 py-2">Contacto</th>
                  <th className="px-3 py-2">Estado</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {proveedores.map((p) => (
                  <tr key={p.id}>
                    <td className="px-3 py-2 font-medium">{p.razon_social}</td>
                    <td className="px-3 py-2 text-gray-600">{p.cuit}</td>
                    <td className="px-3 py-2 text-gray-600">
                      {p.contacto_nombre ?? '—'}
                      {p.contacto_email && (
                        <span className="text-gray-400"> · {p.contacto_email}</span>
                      )}
                    </td>
                    <td className="px-3 py-2">
                      <EstadoBadge estado={p.estado} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <Modal title="Nuevo proveedor" open={showCreate} onClose={() => setShowCreate(false)}>
        <NewProveedorForm
          pending={create.isPending}
          error={create.error instanceof Error ? create.error.message : null}
          onSubmit={(input) => create.mutate(input)}
          onCancel={() => setShowCreate(false)}
        />
      </Modal>
    </section>
  )
}

function NewProveedorForm({
  pending,
  error,
  onSubmit,
  onCancel,
}: {
  pending: boolean
  error: string | null
  onSubmit: (input: { cuit: string; razon_social: string }) => void
  onCancel: () => void
}) {
  const [cuit, setCuit] = useState('')
  const [razonSocial, setRazonSocial] = useState('')
  const cuitValido = CUIT_RE.test(cuit.trim())

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        if (!cuitValido) return
        onSubmit({ cuit: cuit.trim(), razon_social: razonSocial.trim() })
      }}
    >
      <Field
        label="CUIT"
        required
        hint="11 dígitos, sin guiones ni espacios."
        error={cuit && !cuitValido ? 'El CUIT debe tener exactamente 11 dígitos.' : null}
      >
        {(id) => (
          <TextInput
            id={id}
            value={cuit}
            onChange={(e) => setCuit(e.target.value.replace(/\D/g, '').slice(0, 11))}
            inputMode="numeric"
            autoComplete="off"
            required
          />
        )}
      </Field>

      <Field label="Razón social" required>
        {(id) => (
          <TextInput
            id={id}
            value={razonSocial}
            onChange={(e) => setRazonSocial(e.target.value)}
            required
          />
        )}
      </Field>

      {error && <p className="text-sm text-red-600" role="alert">{error}</p>}

      <div className="flex justify-end gap-2">
        <button type="button" onClick={onCancel} className="rounded-md border px-3 py-1.5 text-sm">
          Cancelar
        </button>
        <button
          type="submit"
          disabled={pending || !razonSocial.trim() || !cuitValido}
          className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
        >
          {pending ? 'Creando…' : 'Crear proveedor'}
        </button>
      </div>
    </form>
  )
}
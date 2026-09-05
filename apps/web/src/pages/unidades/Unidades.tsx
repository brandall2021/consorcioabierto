import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { PermissionGate } from '@/components/ui/PermissionGate'
import { Modal } from '@/components/ui/Modal'
import { Field, Select, TextInput } from '@/components/ui/Field'
import { DocumentUpload } from '@/components/documentos/DocumentUpload'
import { ImportWizard } from './ImportWizard'
import { EmptyState, ErrorState, EstadoBadge, PageHeader, SkeletonRows } from '@/components/ui/primitives'

type Unidad = components['schemas']['Unidad']
type UnidadInput = components['schemas']['UnidadInput']

const tipoUnidad: Record<string, string> = {
  departamento: 'Departamento',
  cochera: 'Cochera',
  local: 'Local',
  unidad_edificio: 'Unidad de edificio',
  otros: 'Otros',
}

const vinculoLabel: Record<string, string> = {
  propietario: 'Propietario',
  inquilino: 'Inquilino',
  apoderado: 'Apoderado',
}

export function Unidades() {
  const { consorcioId = '' } = useParams()
  const queryClient = useQueryClient()
  const [estado, setEstado] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [showImport, setShowImport] = useState(false)

  const { data, error, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['unidades', consorcioId],
    queryFn: async () => {
      const res = await client.GET('/consorcios/{id}/unidades', {
        params: { path: { id: consorcioId } },
      })
      if (!res.data) throw new Error('No se pudieron cargar las unidades')
      return res.data
    },
  })

  const unidades = useMemo(() => {
    const all = (data?.data ?? []) as Unidad[]
    return estado ? all.filter((u) => u.estado === estado) : all
  }, [data, estado])

  const create = useMutation({
    mutationFn: async (input: UnidadInput) => {
      const res = await client.POST('/consorcios/{id}/unidades', {
        params: { path: { id: consorcioId } },
        body: input,
      })
      if (res.error) {
        throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo crear la unidad')
      }
      return res.data
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['unidades'] })
      setShowCreate(false)
    },
  })

  return (
    <section>
      <div className="flex items-center justify-between">
        <PageHeader
          title="Unidades funcionales"
          description="UFs del consorcio, sus vínculos y la importación desde CSV."
          actions={
            <PermissionGate permission="unidades.manage">
              <button
                type="button"
                onClick={() => setShowImport(true)}
                className="rounded-md border px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
              >
                Importar CSV
              </button>
              <button
                type="button"
                onClick={() => setShowCreate(true)}
                className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
              >
                Nueva unidad
              </button>
            </PermissionGate>
          }
        />
      </div>

      <div className="mt-4 flex items-center gap-3">
        <label className="text-sm text-gray-500">
          Estado
          <Select value={estado} onChange={(e) => setEstado(e.target.value)}>
            <option value="">Todos</option>
            <option value="activa">Activa</option>
            <option value="inactiva">Inactiva</option>
          </Select>
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
            message={`No se pudieron cargar las unidades: ${error instanceof Error ? error.message : 'error desconocido'}`}
            onRetry={() => void refetch()}
          />
        )}

        {!isLoading && !error && unidades.length === 0 && (
          <EmptyState
            title="No hay unidades todavía"
            description="Creá la primera unidad funcional o importalas desde un CSV."
          />
        )}

        {!isLoading && !error && unidades.length > 0 && (
          <div className="overflow-x-auto rounded-lg border bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="px-3 py-2">Código</th>
                  <th className="px-3 py-2">Tipo</th>
                  <th className="px-3 py-2">Superficie</th>
                  <th className="px-3 py-2">Coeficiente</th>
                  <th className="px-3 py-2">Estado</th>
                  <th className="px-3 py-2">Personas</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {unidades.map((u) => (
                  <tr key={u.id}>
                    <td className="px-3 py-2 font-medium">{u.codigo}</td>
                    <td className="px-3 py-2 text-gray-600">{tipoUnidad[u.tipo] ?? u.tipo}</td>
                    <td className="px-3 py-2 text-gray-600">
                      {u.superficie != null ? `${u.superficie} m²` : '—'}
                    </td>
                    <td className="px-3 py-2 text-gray-600">{u.coeficiente}</td>
                    <td className="px-3 py-2">
                      <EstadoBadge estado={u.estado} />
                    </td>
                    <td className="px-3 py-2 text-gray-600">
                      {u.personas?.length ? (
                        <ul className="space-y-0.5">
                          {u.personas.map((p, i) => (
                            <li key={`${p.persona.id}-${i}`} className="text-xs">
                              {p.persona.nombre}{' '}
                              <span className="text-gray-400">
                                · {vinculoLabel[p.vinculo] ?? p.vinculo}
                                {p.porcentaje ? ` · ${p.porcentaje}%` : ''}
                              </span>
                            </li>
                          ))}
                        </ul>
                      ) : (
                        '—'
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <Modal title="Nueva unidad funcional" open={showCreate} onClose={() => setShowCreate(false)}>
        <NewUnidadForm
          pending={create.isPending}
          error={create.error instanceof Error ? create.error.message : null}
          onSubmit={(input) => create.mutate(input)}
          onCancel={() => setShowCreate(false)}
        />
      </Modal>

      <ImportWizard open={showImport} onClose={() => setShowImport(false)} />

      <div className="mt-6">
        <DocumentUpload tipo="documento" />
      </div>
    </section>
  )
}

function NewUnidadForm({
  pending,
  error,
  onSubmit,
  onCancel,
}: {
  pending: boolean
  error: string | null
  onSubmit: (input: UnidadInput) => void
  onCancel: () => void
}) {
  const [codigo, setCodigo] = useState('')
  const [tipo, setTipo] = useState<UnidadInput['tipo']>('departamento')
  const [superficie, setSuperficie] = useState('')
  const [coeficiente, setCoeficiente] = useState('')
  const [vinculos, setVinculos] = useState<Array<components['schemas']['PersonaVinculoInput']>>([])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({
      codigo: codigo.trim(),
      tipo,
      ...(superficie ? { superficie: Number(superficie) } : {}),
      ...(coeficiente ? { coeficiente } : {}),
      ...(vinculos.length ? { personas: vinculos } : {}),
    })
  }

  return (
    <form className="space-y-4" onSubmit={handleSubmit}>
      <Field label="Código" required>
        {(id) => (
          <TextInput
            id={id}
            value={codigo}
            onChange={(e) => setCodigo(e.target.value)}
            placeholder="Ej. 1A"
            required
          />
        )}
      </Field>

      <Field label="Tipo" required>
        {(id) => (
          <Select id={id} value={tipo} onChange={(e) => setTipo(e.target.value as UnidadInput['tipo'])}>
            {Object.entries(tipoUnidad).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </Select>
        )}
      </Field>

      <div className="grid grid-cols-2 gap-3">
        <Field label="Superficie (m²)">
          {(id) => (
            <TextInput
              id={id}
              type="number"
              min="0"
              step="0.01"
              value={superficie}
              onChange={(e) => setSuperficie(e.target.value)}
              placeholder="Ej. 45.5"
            />
          )}
        </Field>
        <Field label="Coeficiente" hint="Ocho decimales, del 0 al total del consorcio.">
          {(id) => (
            <TextInput
              id={id}
              type="text"
              inputMode="decimal"
              value={coeficiente}
              onChange={(e) => setCoeficiente(e.target.value)}
              placeholder="Ej. 1.00000000"
            />
          )}
        </Field>
      </div>

      <VinculosEditor vinculos={vinculos} onChange={setVinculos} />

      {error && <p className="text-sm text-red-600" role="alert">{error}</p>}

      <div className="flex justify-end gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md border px-3 py-1.5 text-sm text-gray-700"
        >
          Cancelar
        </button>
        <button
          type="submit"
          disabled={pending || !codigo.trim()}
          className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
        >
          {pending ? 'Creando…' : 'Crear unidad'}
        </button>
      </div>
    </form>
  )
}

function VinculosEditor({
  vinculos,
  onChange,
}: {
  vinculos: Array<components['schemas']['PersonaVinculoInput']>
  onChange: (v: Array<components['schemas']['PersonaVinculoInput']>) => void
}) {
  const add = () =>
    onChange([...vinculos, { persona: { nombre: '' }, vinculo: 'propietario' }])

  const update = (i: number, patch: Partial<components['schemas']['PersonaVinculoInput']>) =>
    onChange(vinculos.map((v, idx) => (idx === i ? { ...v, ...patch } : v)))

  const remove = (i: number) => onChange(vinculos.filter((_, idx) => idx !== i))

  return (
    <div>
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600">Vínculos</p>
        <button
          type="button"
          onClick={add}
          className="rounded-md border px-2 py-1 text-xs text-gray-700 hover:bg-gray-100"
        >
          + Persona
        </button>
      </div>

      {vinculos.length === 0 && (
        <p className="mt-1 text-xs text-gray-400">Sin vínculos. Agregá propietarios, inquilinos o apoderados.</p>
      )}

      <div className="mt-2 space-y-3">
        {vinculos.map((v, i) => (
          <div key={i} className="rounded-md border border-gray-200 p-3">
            <div className="grid grid-cols-2 gap-2">
              <Field label="Nombre" required>
                {(id) => (
                  <TextInput
                    id={id}
                    value={v.persona.nombre}
                    onChange={(e) => update(i, { persona: { ...v.persona, nombre: e.target.value } })}
                    required
                  />
                )}
              </Field>
              <Field label="Documento">
                {(id) => (
                  <TextInput
                    id={id}
                    value={v.persona.documento ?? ''}
                    onChange={(e) => update(i, { persona: { ...v.persona, documento: e.target.value || null } })}
                  />
                )}
              </Field>
              <Field label="Vínculo">
                {(id) => (
                  <Select
                    id={id}
                    value={v.vinculo}
                    onChange={(e) => update(i, { vinculo: e.target.value as typeof v.vinculo })}
                  >
                    {Object.entries(vinculoLabel).map(([k, name]) => (
                      <option key={k} value={k}>
                        {name}
                      </option>
                    ))}
                  </Select>
                )}
              </Field>
              <Field label="Porcentaje (%)">
                {(id) => (
                  <TextInput
                    id={id}
                    value={v.porcentaje ?? ''}
                    onChange={(e) => update(i, { porcentaje: e.target.value || null })}
                    placeholder="Ej. 100"
                  />
                )}
              </Field>
            </div>
            <button
              type="button"
              onClick={() => remove(i)}
              className="mt-2 text-xs text-red-600 hover:underline"
            >
              Quitar vínculo
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
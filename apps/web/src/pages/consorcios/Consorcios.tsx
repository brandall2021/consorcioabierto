import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'

type Consorcio = components['schemas']['Consorcio']

const tipoLabel: Record<string, string> = {
  edificio: 'Edificio',
  barrio: 'Barrio',
  complejo: 'Complejo',
  otros: 'Otros',
}

export function Consorcios() {
  const queryClient = useQueryClient()
  const [q, setQ] = useState('')
  const [estado, setEstado] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const { data, error, isLoading, isFetching } = useQuery({
    queryKey: ['consorcios', q, estado],
    queryFn: async () => {
      const res = await client.GET('/consorcios', {
        params: {
          query: {
            q: q || undefined,
            estado: estado || undefined,
          },
        },
      })
      if (!res.data) throw new Error('No se pudieron cargar los consorcios')
      return res.data
    },
  })

  const create = useMutation({
    mutationFn: async (nombre: string) => {
      const res = await client.POST('/consorcios', {
        body: { nombre },
      })
      if (res.error) throw new Error(res.error.detail ?? 'No se pudo crear el consorcio')
      return res.data
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['consorcios'] })
      setShowForm(false)
      setFormError(null)
    },
    onError: (e: Error) => setFormError(e.message),
  })

  const consorcios = (data?.data ?? []) as Consorcio[]

  if (isLoading) {
    return (
      <section aria-busy="true">
        <h1 className="text-lg font-medium">Consorcios</h1>
        <p className="mt-2 text-gray-500">Cargando consorcios…</p>
      </section>
    )
  }

  if (error) {
    return (
      <section>
        <h1 className="text-lg font-medium">Consorcios</h1>
        <p className="mt-2 text-red-600">{error.message}</p>
        <button
          type="button"
          onClick={() => void queryClient.invalidateQueries({ queryKey: ['consorcios'] })}
          className="mt-3 rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
        >
          Reintentar
        </button>
      </section>
    )
  }

  return (
    <section>
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-medium">Consorcios</h1>
        <button
          type="button"
          onClick={() => setShowForm((v) => !v)}
          className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
        >
          {showForm ? 'Cancelar' : 'Nuevo consorcio'}
        </button>
      </div>

      {showForm && (
        <NewConsorcioForm
          onSubmit={(nombre) => create.mutate(nombre)}
          error={formError}
          pending={create.isPending}
        />
      )}

      <div className="mt-4 flex items-center gap-3">
        <label className="text-sm text-gray-500">
          Buscar
          <input
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Nombre o CUIT"
            className="ml-2 rounded-md border px-3 py-1.5 text-sm"
          />
        </label>
        <label className="text-sm text-gray-500">
          Estado
          <select
            value={estado}
            onChange={(e) => setEstado(e.target.value)}
            className="ml-2 rounded-md border px-3 py-1.5 text-sm"
          >
            <option value="">Todos</option>
            <option value="activo">Activo</option>
            <option value="inactivo">Inactivo</option>
          </select>
        </label>
        {isFetching && <span className="text-sm text-gray-400">Actualizando…</span>}
      </div>

      <div className="mt-4 overflow-x-auto rounded-lg border bg-white">
        <table className="w-full text-left text-sm">
          <thead className="border-b bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
            <tr>
              <th className="px-3 py-2">Nombre</th>
              <th className="px-3 py-2">CUIT</th>
              <th className="px-3 py-2">Domicilio</th>
              <th className="px-3 py-2">Tipo</th>
              <th className="px-3 py-2">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {consorcios.length === 0 && (
              <tr>
                <td colSpan={5} className="px-3 py-6 text-center text-gray-500">
                  No hay consorcios. Creá el primero con «Nuevo consorcio».
                </td>
              </tr>
            )}
            {consorcios.map((c) => (
              <tr key={c.id}>
                <td className="px-3 py-2 font-medium">{c.nombre}</td>
                <td className="px-3 py-2 text-gray-500">{c.cuit ?? '—'}</td>
                <td className="px-3 py-2 text-gray-500">{c.domicilio ?? '—'}</td>
                <td className="px-3 py-2 text-gray-500">
                  {(c.tipo && tipoLabel[c.tipo]) || '—'}
                </td>
                <td className="px-3 py-2">
                  <span
                    className={
                      c.estado === 'activo'
                        ? 'rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-700'
                        : 'rounded-full bg-gray-200 px-2 py-0.5 text-xs text-gray-600'
                    }
                  >
                    {c.estado === 'activo' ? 'Activo' : 'Inactivo'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function NewConsorcioForm({
  onSubmit,
  error,
  pending,
}: {
  onSubmit: (nombre: string) => void
  error: string | null
  pending: boolean
}) {
  const [nombre, setNombre] = useState('')

  return (
    <form
      className="mt-4 flex max-w-lg items-end gap-3 rounded-lg border bg-white p-4"
      onSubmit={(e) => {
        e.preventDefault()
        if (nombre.trim()) onSubmit(nombre.trim())
      }}
    >
      <label className="flex-1 text-sm text-gray-600">
        Nombre
        <input
          type="text"
          value={nombre}
          onChange={(e) => setNombre(e.target.value)}
          placeholder="Ej. Torres del Sol"
          required
          className="mt-1 w-full rounded-md border px-3 py-1.5 text-sm"
        />
      </label>
      <button
        type="submit"
        disabled={pending || !nombre.trim()}
        className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
      >
        {pending ? 'Creando…' : 'Crear'}
      </button>
      {error && <p className="w-full text-sm text-red-600">{error}</p>}
    </form>
  )
}

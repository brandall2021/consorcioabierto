import { useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { Modal } from '@/components/ui/Modal'

type ImportJob = components['schemas']['ImportJob']
type ImportModo = 'crear' | 'actualizar' | 'crear_y_actualizar'

// Clave idempotente conservada entre reintentos del mismo intento de confirmar.
let idempotencyKey: string | null = null
function freshKey(): string {
  idempotencyKey = crypto.randomUUID()
  return idempotencyKey
}

export function ImportWizard({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { consorcioId = '' } = useParams()
  const queryClient = useQueryClient()
  const fileInput = useRef<HTMLInputElement>(null)
  const [archivo, setArchivo] = useState<File | null>(null)
  const [modo, setModo] = useState<ImportModo>('crear_y_actualizar')
  const [jobId, setJobId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const parseError = (e: unknown): string => {
    const err = e as { detail?: string; title?: string } | undefined
    return err?.detail ?? err?.title ?? 'Ocurrió un error inesperado.'
  }

  const crearJob = useMutation({
    mutationFn: async () => {
      if (!archivo) throw new Error('Elegí un archivo CSV primero.')
      const form = new FormData()
      form.append('archivo', archivo)
      form.append('modo', modo)
      const res = await client.POST('/consorcios/{id}/unidades/import-jobs', {
        params: { path: { id: consorcioId } },
        body: form as unknown as { archivo: string; modo: ImportModo },
      })
      if (res.error) throw res.error
      if (!res.data) throw new Error('No se pudo crear el job de importación.')
      return res.data
    },
    onMutate: () => setError(null),
    onSuccess: (job) => {
      setJobId(job.id)
      if (job.estado === 'listo' || job.estado === 'procesado') return
      // Si sigue validando, se mostrará el progreso con el query por id.
    },
    onError: (e) => setError(parseError(e)),
  })

  const { data: job, isLoading: jobLoading } = useQuery({
    queryKey: ['import-job', jobId],
    queryFn: async () => {
      if (!jobId) return null
      const res = await client.GET('/import-jobs/{id}', {
        params: { path: { id: jobId } },
      })
      if (!res.data) throw new Error('El job no devolvió datos.')
      return res.data
    },
    enabled: !!jobId && crearJob.isSuccess,
    refetchInterval: (query) =>
      // Polling mientras se valida; detiene cuando llega a un estado terminal.
      query.state.data?.estado === 'validando' ? 1500 : false,
  })

  const confirmar = useMutation({
    mutationFn: async () => {
      if (!jobId) throw new Error('Falta el job.')
      const key = idempotencyKey ?? freshKey()
      const res = await client.POST('/import-jobs/{id}/confirm', {
        params: { path: { id: jobId } },
        headers: { 'Idempotency-Key': key },
      })
      if (res.error) throw new Error('No se pudo confirmar la importación.')
      if (!res.data) throw new Error('La confirmación no devolvió datos.')
      return res.data
    },
    onMutate: () => setError(null),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['unidades'] })
    },
    onError: (e) => setError(parseError(e)),
  })

  const jobConfirmado = confirmar.data as ImportJob | undefined
  const jobActual = (jobConfirmado ?? job ?? (crearJob.data as ImportJob | undefined)) ?? null
  const estadoTerminal = jobActual?.estado === 'listo'
  const estadoProcesado = jobActual?.estado === 'procesado'
  const esperandoValidacion = jobActual?.estado === 'validando'

  const validaCsv = (): string | null => {
    if (!archivo) return 'Elegí un archivo CSV.'
    if (!/\.csv$/i.test(archivo.name)) return 'El archivo debe ser .csv.'
    if (archivo.size > 10 * 1024 * 1024) return 'El archivo supera los 10 MiB.'
    return null
  }

  const reset = () => {
    setArchivo(null)
    setJobId(null)
    setError(null)
    idempotencyKey = null
    if (fileInput.current) fileInput.current.value = ''
    queryClient.removeQueries({ queryKey: ['import-job'] })
  }

  const close = () => {
    reset()
    onClose()
  }

  const paso = !jobActual ? 1 : estadoProcesado ? 3 : 2

  return (
    <Modal title="Importar unidades funcionales (CSV)" open={open} onClose={close}>
      {/* Paso 1: elegir archivo */}
      {paso === 1 && (
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            const v = validaCsv()
            if (v) {
              setError(v)
              return
            }
            crearJob.mutate()
          }}
        >
          <p className="text-sm text-gray-600">
            Plantilla{' '}
            <code className="rounded bg-gray-100 px-1 py-0.5 text-xs">ufs-v1</code>
            : la primera fila fija los encabezados esperados. Cada error de fila se informa en el
            preview antes de confirmar.
          </p>

          <input
            ref={fileInput}
            type="file"
            accept=".csv,text/csv"
            className="block w-full text-sm text-gray-600 file:mr-3 file:rounded-md file:border-0 file:bg-gray-100 file:px-3 file:py-1.5 file:text-sm file:text-gray-700"
            onChange={(e) => {
              setArchivo(e.target.files?.[0] ?? null)
              setError(null)
            }}
          />

          <label className="block text-sm text-gray-600">
            Modo de importación
            <select
              value={modo}
              onChange={(e) => setModo(e.target.value as ImportModo)}
              className="mt-1 w-full rounded-md border px-3 py-1.5 text-sm"
            >
              <option value="crear">Solo crear</option>
              <option value="actualizar">Solo actualizar</option>
              <option value="crear_y_actualizar">Crear y actualizar</option>
            </select>
          </label>

          {error && (
            <p className="text-sm text-red-600" role="alert">
              {error}
            </p>
          )}

          <div className="flex justify-end gap-2">
            <button type="button" onClick={close} className="rounded-md border px-3 py-1.5 text-sm">
              Cancelar
            </button>
            <button
              type="submit"
              disabled={crearJob.isPending}
              className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
            >
              {crearJob.isPending ? 'Validando…' : 'Subir y previsualizar'}
            </button>
          </div>
        </form>
      )}

      {/* Paso 2: preview con errores por fila + confirmar */}
      {paso === 2 && (
        <div className="space-y-4">
          {jobLoading && <p className="text-sm text-gray-500">Cargando preview…</p>}

          {!jobLoading && jobActual && (
            <>
              {esperandoValidacion && (
                <p className="text-sm text-amber-700" role="status">
                  Validando el archivo… esperá un momento.
                </p>
              )}

              <dl className="grid grid-cols-2 gap-3 text-sm">
                <div className="rounded-md bg-gray-50 p-3">
                  <dt className="text-xs uppercase tracking-wide text-gray-400">Filas</dt>
                  <dd className="mt-1 font-medium">{jobActual.total_filas ?? '—'}</dd>
                </div>
                <div className="rounded-md bg-gray-50 p-3">
                  <dt className="text-xs uppercase tracking-wide text-gray-400">Errores</dt>
                  <dd className="mt-1 font-medium">{jobActual.errores?.length ?? 0}</dd>
                </div>
              </dl>

              {jobActual.errores && jobActual.errores.length > 0 && (
                <div className="max-h-56 overflow-y-auto rounded-md border" role="table">
                  <table className="w-full text-left text-xs">
                    <thead className="sticky top-0 border-b bg-gray-50 uppercase tracking-wide text-gray-500">
                      <tr>
                        <th className="px-2 py-1">Fila</th>
                        <th className="px-2 py-1">Campo</th>
                        <th className="px-2 py-1">Error</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {jobActual.errores.map((e, i) => (
                        <tr key={i}>
                          <td className="px-2 py-1 text-gray-500">{e.fila}</td>
                          <td className="px-2 py-1">
                            <ul className="space-y-1">
                              {e.campos.map((c, j) => (
                                <li key={j} className="text-gray-600">
                                  {c.campo}
                                </li>
                              ))}
                            </ul>
                          </td>
                          <td className="px-2 py-1 text-red-600">
                            <ul className="space-y-1">
                              {e.campos.map((c, j) => (
                                <li key={j}>{c.error}</li>
                              ))}
                            </ul>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {jobActual.estado !== 'listo' && !esperandoValidacion && (
                <p className="text-sm text-amber-700">
                  El archivo tiene errores. Corregilos y volvé a subir; no se importa nada sin
                  confirmar.
                </p>
              )}

              {error && (
                <p className="text-sm text-red-600" role="alert">
                  {error}
                </p>
              )}

              <div className="flex justify-end gap-2">
                <button type="button" onClick={close} className="rounded-md border px-3 py-1.5 text-sm">
                  Cancelar
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setJobId(null)
                    setArchivo(null)
                    if (fileInput.current) fileInput.current.value = ''
                  }}
                  className="rounded-md border px-3 py-1.5 text-sm"
                >
                  Otro archivo
                </button>
                <button
                  type="button"
                  disabled={!estadoTerminal || confirmar.isPending || esperandoValidacion}
                  onClick={() => {
                    idempotencyKey ??= freshKey()
                    confirmar.mutate()
                  }}
                  className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
                >
                  {confirmar.isPending ? 'Confirmando…' : 'Confirmar importación'}
                </button>
              </div>
            </>
          )}
        </div>
      )}

      {/* Paso 3: resultado */}
      {paso === 3 && (
        <div className="space-y-4">
          <div role="status">
            <p className="text-sm font-medium text-green-700">Importación completada.</p>
            <dl className="mt-3 grid grid-cols-3 gap-3 text-sm">
              <div className="rounded-md bg-gray-50 p-3">
                <dt className="text-xs uppercase tracking-wide text-gray-400">Creadas</dt>
                <dd className="mt-1 font-medium">{jobActual?.creados ?? 0}</dd>
              </div>
              <div className="rounded-md bg-gray-50 p-3">
                <dt className="text-xs uppercase tracking-wide text-gray-400">Actualizadas</dt>
                <dd className="mt-1 font-medium">{jobActual?.actualizados ?? 0}</dd>
              </div>
              <div className="rounded-md bg-gray-50 p-3">
                <dt className="text-xs uppercase tracking-wide text-gray-400">Rechazadas</dt>
                <dd className="mt-1 font-medium">{jobActual?.rechazados ?? 0}</dd>
              </div>
            </dl>
          </div>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={close}
              className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white"
            >
              Cerrar
            </button>
          </div>
        </div>
      )}
    </Modal>
  )
}
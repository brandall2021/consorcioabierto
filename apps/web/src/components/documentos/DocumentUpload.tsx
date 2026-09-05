import { useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { client } from '@/api/client'

const MAX_UPLOAD = 10 * 1024 * 1024 // 10 MiB, coincide con MAX_UPLOAD_BYTES del backend.

type Estado = 'idle' | 'hash' | 'subiendo' | 'escaneando' | 'listo' | 'cuarentena' | 'error'

export function DocumentUpload({ tipo }: { tipo: string }) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [archivo, setArchivo] = useState<File | null>(null)
  const [estado, setEstado] = useState<Estado>('idle')
  const [detalle, setDetalle] = useState<string | null>(null)
  const [urlDescarga, setUrlDescarga] = useState<string | null>(null)

  const subir = useMutation({
    mutationFn: async () => {
      if (!archivo) return
      if (archivo.size > MAX_UPLOAD) {
        throw new Error('El archivo supera el máximo permitido (10 MiB).')
      }
      setEstado('hash')
      const sha256 = await hashSha256(archivo)

      setEstado('subiendo')
      const intent = await client.POST('/document-upload-intents', {
        body: { tipo, nombre: archivo.name, size_bytes: archivo.size, sha256 },
      })
      if (!intent.data) {
        const err = intent.error as { detail?: string; status?: number } | undefined
        if (err?.status === 413) throw new Error('El archivo supera el máximo permitido (10 MiB).')
        throw new Error(err?.detail ?? 'No se pudo iniciar la subida')
      }
      const { documento_id: documentoId, upload_url } = intent.data
      if (!upload_url) throw new Error('El servidor no devolvió una URL de subida.')
      if (!documentoId) throw new Error('El servidor no devolvió el identificador del documento.')

      setEstado('escaneando')
      const upload = await fetch(upload_url, {
        method: 'PUT',
        body: archivo,
        headers: { 'Content-Type': 'application/octet-stream' },
      })
      if (!upload.ok) {
        throw new Error(`La subida al almacenamiento falló (HTTP ${upload.status}).`)
      }

      // La descarga dispara el escaneo antivirus perezoso y la URL firmada.
      const download = await client.GET('/documentos/{id}/download-url', {
        params: { path: { id: documentoId } },
      })
      if (download.data) {
        setUrlDescarga(download.data.url)
        setEstado('listo')
        return
      }
      const derr = download.error as { status?: number } | undefined
      if (derr?.status === 410) {
        setEstado('cuarentena')
        setDetalle('El antivirus rechazó el archivo. No se habilita la descarga.')
        return
      }
      throw new Error('El antivirus no pudo verificarse. Reintentá en unos segundos.')
    },
    onError: (e: Error) => {
      setEstado('error')
      setDetalle(e.message)
    },
  })

  const statusMessage = ((): { text: string; tone: 'gray' | 'green' | 'red' | 'amber' } | null => {
    switch (estado) {
      case 'hash':
        return { text: 'Calculando hash del archivo…', tone: 'gray' }
      case 'subiendo':
        return { text: 'Subiendo el archivo…', tone: 'gray' }
      case 'escaneando':
        return { text: 'Escaneando con antivirus…', tone: 'amber' }
      case 'listo':
        return { text: 'Antivirus limpio. Subida completa.', tone: 'green' }
      case 'cuarentena':
        return { text: 'Archivo en cuarentena por antivirus.', tone: 'red' }
      case 'error':
        return { text: detalle ?? 'Ocurrió un error.', tone: 'red' }
      default:
        return null
    }
  })()

  const reset = () => {
    setArchivo(null)
    setEstado('idle')
    setDetalle(null)
    setUrlDescarga(null)
    if (inputRef.current) inputRef.current.value = ''
  }

  return (
    <div className="rounded-lg border bg-white p-4">
      <h2 className="text-sm font-medium">Subir documento</h2>
      <p className="mt-1 text-xs text-gray-500">
        El archivo se envía directo al almacenamiento con URL firmada y se escanea con antivirus.
        Máximo 10 MiB.
      </p>

      <form
        className="mt-3 flex flex-wrap items-center gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          subir.mutate()
        }}
      >
        <input
          ref={inputRef}
          type="file"
          className="block text-sm text-gray-600 file:mr-3 file:rounded-md file:border-0 file:bg-gray-100 file:px-3 file:py-1.5 file:text-sm file:text-gray-700"
          onChange={(e) => {
            const f = e.target.files?.[0] ?? null
            setArchivo(f)
            setEstado('idle')
            setDetalle(null)
            setUrlDescarga(null)
          }}
        />
        <button
          type="submit"
          disabled={!archivo || subir.isPending || estado === 'escaneando'}
          className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
        >
          {subir.isPending ? 'Procesando…' : 'Subir y verificar'}
        </button>
        {estado !== 'idle' && (
          <button
            type="button"
            onClick={reset}
            className="rounded-md border px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
          >
            Reiniciar
          </button>
        )}
      </form>

      {statusMessage && (
        <p className={`mt-3 text-sm ${colorFor(statusMessage.tone)}`} role="status">
          {statusMessage.text}
        </p>
      )}

      {estado === 'listo' && urlDescarga && (
        <a
          href={urlDescarga}
          target="_blank"
          rel="noreferrer"
          className="mt-2 inline-block rounded-md border px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
        >
          Descargar documento
        </a>
      )}
    </div>
  )
}

function colorFor(tone: 'gray' | 'green' | 'red' | 'amber'): string {
  switch (tone) {
    case 'green':
      return 'text-green-700'
    case 'red':
      return 'text-red-600'
    case 'amber':
      return 'text-amber-700'
    default:
      return 'text-gray-500'
  }
}

async function hashSha256(file: File): Promise<string> {
  const buf = await file.arrayBuffer()
  const digest = await crypto.subtle.digest('SHA-256', buf)
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}
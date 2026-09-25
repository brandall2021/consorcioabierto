import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { Badge, EmptyState, ErrorState, PageHeader, SkeletonRows } from '@/components/ui/primitives'

type Comunicado = components['schemas']['Comunicado'] & {
	cuerpo?: string
	destinatarios?: string
	publicado_at?: string | null
	created_at?: string
}

type ComunicadosResponse = { data: Comunicado[] }

export function Comunicados() {
	const { consorcioId = '' } = useParams()
	const queryClient = useQueryClient()
	const [titulo, setTitulo] = useState('')
	const [cuerpo, setCuerpo] = useState('')
	const [destinatarios, setDestinatarios] = useState<'todos' | 'unidades'>('todos')

	const query = useQuery({
		queryKey: ['comunicados', consorcioId],
		queryFn: async () => {
			const res = await client.GET('/consorcios/{id}/comunicados', { params: { path: { id: consorcioId } } })
			if (!res.data) throw new Error('No se pudieron cargar los comunicados')
			return res.data as ComunicadosResponse
		},
	})

	const create = useMutation({
		mutationFn: async () => {
			const res = await client.POST('/consorcios/{id}/comunicados', {
				params: { path: { id: consorcioId } },
				body: { titulo, cuerpo, destinatarios },
			})
			if (!res.data) throw new Error('No se pudo crear el comunicado')
			return res.data
		},
		onSuccess: async () => {
			setTitulo('')
			setCuerpo('')
			setDestinatarios('todos')
			await queryClient.invalidateQueries({ queryKey: ['comunicados', consorcioId] })
		},
	})

	const publish = useMutation({
		mutationFn: async (id: string) => {
			const res = await client.POST('/comunicados/{id}/publicar', {
				params: { path: { id } },
				headers: { 'Idempotency-Key': crypto.randomUUID() },
			})
			if (!res.data) throw new Error('No se pudo publicar el comunicado')
			return res.data
		},
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ['comunicados', consorcioId] })
		},
	})

	const comunicados = query.data?.data ?? []

	return (
		<section>
			<PageHeader
				title="Comunicados"
				description="Redactá un comunicado y encolalo para envío a las unidades del consorcio."
			/>

			<form
				className="mt-4 grid gap-3 rounded-2xl border bg-white p-4 shadow-sm md:grid-cols-[1fr_1fr_auto]"
				onSubmit={(e) => {
					e.preventDefault()
					if (titulo.trim() && cuerpo.trim()) void create.mutateAsync()
				}}
			>
				<label className="text-sm text-gray-600">
					Título
					<input className="mt-1 w-full rounded-md border px-3 py-2" value={titulo} onChange={(e) => setTitulo(e.target.value)} required />
				</label>
				<label className="text-sm text-gray-600 md:col-span-2">
					Cuerpo
					<textarea className="mt-1 min-h-24 w-full rounded-md border px-3 py-2" value={cuerpo} onChange={(e) => setCuerpo(e.target.value)} required />
				</label>
				<label className="text-sm text-gray-600">
					Destinatarios
					<select className="mt-1 w-full rounded-md border px-3 py-2" value={destinatarios} onChange={(e) => setDestinatarios(e.target.value as 'todos' | 'unidades')}>
						<option value="todos">Todos</option>
						<option value="unidades">Solo unidades</option>
					</select>
				</label>
				<button type="submit" disabled={create.isPending} className="h-10 rounded-md bg-gray-900 px-4 text-sm font-medium text-white disabled:opacity-40">
					{create.isPending ? 'Creando…' : 'Crear'}
				</button>
			</form>

			<div className="mt-4">
				{query.isLoading && <SkeletonRows rows={4} />}
				{query.error && <ErrorState message={`No se pudieron cargar los comunicados: ${query.error instanceof Error ? query.error.message : 'error desconocido'}`} onRetry={() => void query.refetch()} />}
				{!query.isLoading && !query.error && comunicados.length === 0 && <EmptyState title="Sin comunicados" description="Creá el primero desde el formulario de arriba." />}
				<div className="space-y-3">
					{comunicados.map((c) => (
						<article key={c.id} className="rounded-2xl border bg-white p-4 shadow-sm">
							<div className="flex items-start justify-between gap-4">
								<div>
									<h3 className="font-medium text-gray-900">{c.titulo}</h3>
									<p className="mt-1 text-sm text-gray-500">{c.cuerpo}</p>
								</div>
								<div className="text-right">
									<Badge tone={c.estado === 'publicado' ? 'green' : 'amber'}>{c.estado}</Badge>
									<p className="mt-2 text-xs text-gray-400">{c.destinatarios ?? 'todos'}</p>
								</div>
							</div>
							{c.estado === 'borrador' && (
								<button type="button" onClick={() => publish.mutate(c.id)} disabled={publish.isPending} className="mt-4 rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40">
									{publish.isPending ? 'Publicando…' : 'Publicar'}
								</button>
							)}
						</article>
					))}
				</div>
			</div>
		</section>
	)
}

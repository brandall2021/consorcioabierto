import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { client } from '@/api/client'
import type { components } from '@/api/generated.d'
import { Modal } from '@/components/ui/Modal'
import { Select } from '@/components/ui/Field'

type Persona = components['schemas']['Persona']
type Miembro = components['schemas']['TenantMember']

// VincularUsuario asocia una persona de una UF con el usuario de la plataforma
// que la representa. Sin este vinculo el portal del consorcista ve cero
// unidades, porque el scope es user -> persona -> unidad_personas.
export function VincularUsuario({ persona }: { persona: Persona }) {
	const queryClient = useQueryClient()
	const [open, setOpen] = useState(false)
	const [usuarioID, setUsuarioID] = useState('')

	const { data, isLoading } = useQuery({
		queryKey: ['tenant-members'],
		queryFn: async () => {
			const res = await client.GET('/tenant/members')
			if (!res.data) throw new Error('No se pudieron cargar los miembros')
			return res.data
		},
		enabled: open,
	})

	const vincular = useMutation({
		mutationFn: async (id: string) => {
			const res = await client.PUT('/tenant/personas/{personaId}/usuario', {
				params: { path: { personaId: persona.id } },
				body: { usuario_id: id },
			})
			if (res.error) {
				throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo vincular')
			}
			return res.data
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ['unidades'] })
			setOpen(false)
			setUsuarioID('')
		},
	})

	const desvincular = useMutation({
		mutationFn: async () => {
			const res = await client.DELETE('/tenant/personas/{personaId}/usuario', {
				params: { path: { personaId: persona.id } },
			})
			if (res.error) {
				throw new Error((res.error as { detail?: string }).detail ?? 'No se pudo desvincular')
			}
			return res.data
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ['unidades'] })
		},
	})

	const miembros = (data?.data ?? []) as Miembro[]
	const error =
		(vincular.error instanceof Error ? vincular.error.message : null) ??
		(desvincular.error instanceof Error ? desvincular.error.message : null)

	return (
		<>
			<span className="text-xs">
				{persona.user_id ? (
					<span className="text-green-700">Vinculada</span>
				) : (
					<span className="text-gray-400">Sin vincular</span>
				)}
			</span>

			<button
				type="button"
				onClick={() => setOpen(true)}
				className="ml-2 text-xs text-gray-500 underline hover:text-gray-800"
			>
				{persona.user_id ? 'Cambiar usuario' : 'Vincular usuario'}
			</button>

			{persona.user_id && (
				<button
					type="button"
					onClick={() => desvincular.mutate()}
					disabled={desvincular.isPending}
					className="ml-2 text-xs text-red-600 underline disabled:opacity-40"
				>
					{desvincular.isPending ? 'Desvinculando…' : 'Desvincular'}
				</button>
			)}

			{error && <span className="mt-1 block text-xs text-red-600">{error}</span>}

			<Modal title={`Vincular usuario a ${persona.nombre}`} open={open} onClose={() => setOpen(false)}>
				{isLoading && <p className="text-sm text-gray-500">Cargando miembros…</p>}

				{!isLoading && miembros.length === 0 && (
					<p className="text-sm text-gray-500">Este tenant todavía no tiene miembros para vincular.</p>
				)}

				{miembros.length > 0 && (
					<form
						onSubmit={(e) => {
							e.preventDefault()
							if (usuarioID) vincular.mutate(usuarioID)
						}}
						className="space-y-3"
					>
						<label className="block text-sm text-gray-600">
							Usuario
							<Select value={usuarioID} onChange={(e) => setUsuarioID(e.target.value)}>
								<option value="">Elegí un miembro</option>
								{miembros.map((m) => (
									<option key={m.user_id} value={m.user_id}>
										{m.nombre} · {m.email}
									</option>
								))}
							</Select>
						</label>

						<div className="flex justify-end gap-2">
							<button
								type="button"
								onClick={() => setOpen(false)}
								className="rounded-md border px-3 py-1.5 text-sm"
							>
								Cerrar
							</button>
							<button
								type="submit"
								disabled={!usuarioID || vincular.isPending}
								className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-40"
							>
								{vincular.isPending ? 'Vinculando…' : 'Vincular'}
							</button>
						</div>
					</form>
				)}
			</Modal>
		</>
	)
}
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Reclamos } from './Reclamos'

const listGet = vi.fn()
const unidadesGet = vi.fn()
const detailGet = vi.fn()
const transicionPost = vi.fn()
const mensajePost = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/consorcios/{id}/reclamos') return listGet()
			if (path === '/consorcios/{id}/unidades') return unidadesGet()
			if (path === '/reclamos/{id}') return detailGet()
			throw new Error(`GET sin mock: ${path}`)
		},
		POST: (path: string, args: unknown) => {
			if (path === '/reclamos/{id}/transiciones') return transicionPost(args)
			if (path === '/reclamos/{id}/mensajes') return mensajePost(args)
			throw new Error(`POST sin mock: ${path}`)
		},
	},
}))

const permisos = { read: true, manage: true }
vi.mock('@/auth/AuthProvider', () => ({
	useAuth: () => ({
		me: {
			permissions: [
				...(permisos.read ? ['reclamos.read'] : []),
				...(permisos.manage ? ['reclamos.manage'] : []),
			],
		},
	}),
}))

const RECLAMO = {
	id: '11111111-1111-1111-1111-111111111111',
	unidad_id: '22222222-2222-2222-2222-222222222222',
	categoria: 'humedad',
	estado: 'abierto',
	responsable_id: null,
	created_at: '2026-09-01T10:00:00Z',
}

function page(data: unknown[], total = data.length) {
	return { data, meta: { request_id: 'req-1', total } }
}

function renderReclamos() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter initialEntries={['/app/consorcios/cons-1/reclamos']}>
				<Routes>
					<Route path="/app/consorcios/:consorcioId/reclamos" element={<Reclamos />} />
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>,
	)
}

describe('Reclamos', () => {
	beforeEach(() => {
		listGet.mockReset()
		unidadesGet.mockReset()
		detailGet.mockReset()
		transicionPost.mockReset()
		mensajePost.mockReset()
		permisos.read = true
		permisos.manage = true
		unidadesGet.mockResolvedValue({
			data: { data: [{ id: RECLAMO.unidad_id, codigo: '3B', consorcio_id: 'cons-1' }], meta: {} },
		})
	})

	it('muestra el codigo de la unidad en vez del identificador interno', async () => {
		listGet.mockResolvedValue({ data: page([RECLAMO]) })

		renderReclamos()

		expect(await screen.findByText(/3B/)).toBeInTheDocument()
		expect(screen.queryByText(new RegExp(RECLAMO.unidad_id.slice(0, 8)))).not.toBeInTheDocument()
	})

	it('cae al identificador cuando la unidad no esta en la lista', async () => {
		unidadesGet.mockResolvedValue({ data: { data: [], meta: {} } })
		listGet.mockResolvedValue({ data: page([RECLAMO]) })

		renderReclamos()

		expect(await screen.findByText(new RegExp(RECLAMO.unidad_id.slice(0, 8)))).toBeInTheDocument()
	})

	it('lista los reclamos del consorcio con su estado', async () => {
		listGet.mockResolvedValue({ data: page([RECLAMO]) })

		renderReclamos()

		expect(await screen.findByText('humedad')).toBeInTheDocument()
		expect(screen.getByText('Abierto')).toBeInTheDocument()
		expect(screen.getByText('1 reclamo')).toBeInTheDocument()
	})

	it('muestra estado vacio cuando el consorcio no tiene reclamos', async () => {
		listGet.mockResolvedValue({ data: page([]) })

		renderReclamos()

		expect(await screen.findByText(/todavía no hay reclamos/i)).toBeInTheDocument()
	})

	it('abre el detalle y muestra mensajes e historial', async () => {
		listGet.mockResolvedValue({ data: page([RECLAMO]) })
		detailGet.mockResolvedValue({
			data: {
				reclamo: RECLAMO,
				mensajes: [
					{ id: 'm1', reclamo_id: RECLAMO.id, texto: 'Se moja la pared', created_at: '2026-09-01T10:05:00Z' },
				],
				transiciones: [
					{
						id: 't1',
						reclamo_id: RECLAMO.id,
						accion: 'asignar',
						motivo: null,
						responsable_id: null,
						created_at: '2026-09-01T11:00:00Z',
					},
				],
			},
		})

		renderReclamos()
		await userEvent.click(await screen.findByRole('button', { name: /humedad/ }))

		expect(await screen.findByText('Se moja la pared')).toBeInTheDocument()
		expect(screen.getByText('Historial')).toBeInTheDocument()
	})

	it('solo ofrece transiciones validas para el estado actual', async () => {
		listGet.mockResolvedValue({ data: page([RECLAMO]) })
		detailGet.mockResolvedValue({ data: { reclamo: RECLAMO, mensajes: [], transiciones: [] } })

		renderReclamos()
		await userEvent.click(await screen.findByRole('button', { name: /humedad/ }))

		expect(await screen.findByRole('button', { name: 'Marcar en progreso' })).toBeInTheDocument()
		expect(screen.queryByRole('button', { name: 'Reabrir' })).not.toBeInTheDocument()
	})

	it('pide motivo al reabrir un reclamo cerrado', async () => {
		listGet.mockResolvedValue({
			data: page([{ ...RECLAMO, estado: 'cerrado' }]),
		})
		detailGet.mockResolvedValue({
			data: { reclamo: { ...RECLAMO, estado: 'cerrado' }, mensajes: [], transiciones: [] },
		})

		renderReclamos()
		await userEvent.click(await screen.findByRole('button', { name: /humedad/ }))

		await userEvent.click(await screen.findByRole('button', { name: 'Reabrir' }))
		expect(screen.getByLabelText(/motivo/i)).toBeInTheDocument()
	})

	it('envia la transicion con motivo y refresca la lista', async () => {
		listGet.mockResolvedValue({ data: page([{ ...RECLAMO, estado: 'cerrado' }]) })
		detailGet.mockResolvedValue({
			data: { reclamo: { ...RECLAMO, estado: 'cerrado' }, mensajes: [], transiciones: [] },
		})
		transicionPost.mockResolvedValue({
			data: { ...RECLAMO, estado: 'abierto' },
		})

		renderReclamos()
		await userEvent.click(await screen.findByRole('button', { name: /humedad/ }))
		await userEvent.click(await screen.findByRole('button', { name: 'Reabrir' }))
		await userEvent.type(screen.getByLabelText(/motivo/i), 'Volvio a humedecerse')
		await userEvent.click(screen.getByRole('button', { name: 'Confirmar' }))

		await waitFor(() => expect(transicionPost).toHaveBeenCalled())
		const [args] = transicionPost.mock.calls[0]
		expect(args.body).toMatchObject({
			accion: 'reabrir',
			motivo: 'Volvio a humedecerse',
		})
		await waitFor(() => expect(listGet.mock.calls.length).toBeGreaterThan(1))
	})

	it('muestra el error del backend cuando la transicion es invalida', async () => {
		listGet.mockResolvedValue({ data: page([RECLAMO]) })
		detailGet.mockResolvedValue({ data: { reclamo: RECLAMO, mensajes: [], transiciones: [] } })
		transicionPost.mockResolvedValue({
			error: { detail: 'transicion invalida', status: 409 },
		})

		renderReclamos()
		await userEvent.click(await screen.findByRole('button', { name: /humedad/ }))
		await userEvent.click(await screen.findByRole('button', { name: 'Resolver' }))
		await userEvent.click(screen.getByRole('button', { name: 'Confirmar' }))

		expect(await screen.findByText(/transicion invalida/i)).toBeInTheDocument()
	})

	it('agrega un mensaje al reclamo', async () => {
		listGet.mockResolvedValue({ data: page([RECLAMO]) })
		detailGet.mockResolvedValue({ data: { reclamo: RECLAMO, mensajes: [], transiciones: [] } })
		mensajePost.mockResolvedValue({ data: {} })

		renderReclamos()
		await userEvent.click(await screen.findByRole('button', { name: /humedad/ }))

		const campo = await screen.findByLabelText(/mensaje/i)
		await userEvent.type(campo, 'Adjunto fotos')
		await userEvent.click(screen.getByRole('button', { name: 'Enviar mensaje' }))

		await waitFor(() => expect(mensajePost).toHaveBeenCalled())
		const [args] = mensajePost.mock.calls[0]
		expect(args.body).toMatchObject({ texto: 'Adjunto fotos' })
	})

	it('oculta las acciones de gestion sin permiso reclamos.manage', async () => {
		permisos.manage = false
		listGet.mockResolvedValue({ data: page([RECLAMO]) })
		detailGet.mockResolvedValue({ data: { reclamo: RECLAMO, mensajes: [], transiciones: [] } })

		renderReclamos()
		await userEvent.click(await screen.findByRole('button', { name: /humedad/ }))

		await screen.findByText('Se necesita permiso de gestión para actuar sobre el reclamo.')
		expect(screen.queryByRole('button', { name: 'Resolver' })).not.toBeInTheDocument()
	})

	it('no muestra la seccion sin permiso reclamos.read', async () => {
		permisos.read = false

		renderReclamos()

		expect(await screen.findByText(/no tenés permiso/i)).toBeInTheDocument()
		expect(listGet).not.toHaveBeenCalled()
	})
})

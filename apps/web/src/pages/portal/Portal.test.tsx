import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Portal } from './Portal'

const portalGet = vi.fn()
const portalPost = vi.fn()
const logout = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/portal') return portalGet()
			throw new Error(`GET sin mock: ${path}`)
		},
		POST: (path: string, options?: unknown) => {
			if (path === '/portal/reclamos') return portalPost(options)
			throw new Error(`POST sin mock: ${path}`)
		},
	},
}))

vi.mock('@/auth/AuthProvider', () => ({
	useAuth: () => ({
		logout,
		me: { membership: { tenant_name: 'Torre A' } },
	}),
}))

function renderPortal() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<Portal />
		</QueryClientProvider>,
	)
}

describe('Portal', () => {
	beforeEach(() => {
		portalGet.mockReset()
		portalPost.mockReset()
		logout.mockReset()
	})

	it('muestra el saldo vencido total y los recibos recientes', async () => {
		portalGet.mockResolvedValue({
			data: {
				data: {
					consorcios: [
						{ id: 'c1', nombre: 'Consorcio Norte', estado: 'activo', saldo_vencido_cents: 15000, cargos_vencidos: 1, vencido_desde: '2026-09-01' },
					],
					recibos_recientes: [
						{ consorcio_id: 'c1', consorcio_nombre: 'Consorcio Norte', cobranza_id: 'p1', fecha: '2026-09-10', importe_cents: 5000, estado: 'acreditado', saldo_a_favor_cents: 0, created_at: '2026-09-10T12:00:00Z' },
					],
					comunicados_recientes: [],
					reclamos_recientes: [],
					unidades: [],
					total_saldo_vencido_cents: 15000,
				},
			},
			error: undefined,
		})

		renderPortal()

		await waitFor(() => expect(screen.getByText('Saldo vencido total')).toBeInTheDocument())
		expect(screen.getAllByText(/150/).length).toBeGreaterThan(0)
		expect(screen.getAllByText('Consorcio Norte').length).toBeGreaterThan(0)
		expect(screen.getAllByText('Recibos recientes').length).toBeGreaterThan(0)
	})

	function portalHome(reclamos: unknown[], unidades: unknown[] = []) {
		portalGet.mockResolvedValue({
			data: {
				data: {
					consorcios: [],
					unidades,
					recibos_recientes: [],
					comunicados_recientes: [],
					reclamos_recientes: reclamos,
					total_saldo_vencido_cents: 0,
				},
			},
			error: undefined,
		})
	}

	it('lista los reclamos del consorcista con texto y estado', async () => {
		portalHome([
			{
				id: 'r1',
				consorcio_id: 'c1',
				unidad_id: 'u1',
				categoria: 'humedad',
				estado: 'en_progreso',
				texto: 'Se moja la pared del pasillo',
				created_at: '2026-09-20T10:00:00Z',
			},
		])

		renderPortal()

		expect(await screen.findByText('humedad')).toBeInTheDocument()
		expect(screen.getByText('Se moja la pared del pasillo')).toBeInTheDocument()
		expect(screen.getByText('En progreso')).toBeInTheDocument()
	})

	it('traduce el estado del reclamo y no muestra el id interno de la unidad', async () => {
		portalHome([
			{
				id: 'r1',
				consorcio_id: 'c1',
				unidad_id: 'unidad-uuid-12345678',
				categoria: 'ruido',
				estado: 'cerrado',
				texto: 'Musica muy fuerte de noche',
				created_at: '2026-09-01T10:00:00Z',
			},
		])

		renderPortal()

		expect(await screen.findByText('Cerrado')).toBeInTheDocument()
		expect(screen.queryByText(/unidad-uuid/)).not.toBeInTheDocument()
	})

	it('invita a abrir un reclamo cuando no hay ninguno', async () => {
		portalHome([])

		renderPortal()

		expect(await screen.findByText('Todavía no abriste ningún reclamo')).toBeInTheDocument()
	})
	function portalConUnidad() {
		portalGet.mockResolvedValue({
			data: {
				data: {
					consorcios: [],
					unidades: [{ id: 'u1', consorcio_id: 'c1', codigo: '1A' }],
					recibos_recientes: [],
					comunicados_recientes: [],
					reclamos_recientes: [],
					total_saldo_vencido_cents: 0,
				},
			},
			error: undefined,
		})
	}

	it('abre el formulario de reclamo cuando el usuario tiene unidades', async () => {
		portalConUnidad()

		renderPortal()

		expect(await screen.findByRole('button', { name: 'Abrir reclamo' })).toBeInTheDocument()
	})

	it('envía el reclamo con la unidad resuelta y recarga el portal', async () => {
		portalConUnidad()
		portalPost.mockResolvedValue({
			data: { id: 'r9', estado: 'abierto', categoria: 'humedad' },
			error: undefined,
		})

		renderPortal()

		fireEvent.click(await screen.findByRole('button', { name: 'Abrir reclamo' }))
		fireEvent.change(await screen.findByLabelText('Categoría'), {
			target: { value: 'humedad' },
		})
		fireEvent.change(screen.getByLabelText('¿Qué pasa?'), {
			target: { value: 'Se moja la pared' },
		})
		fireEvent.click(screen.getByRole('button', { name: 'Enviar reclamo' }))

		await waitFor(() => expect(portalPost).toHaveBeenCalledTimes(1))
		expect(portalPost).toHaveBeenCalledWith({
			body: { unidad_id: 'u1', categoria: 'humedad', texto: 'Se moja la pared' },
		})
	})

	it('muestra el error del servidor cuando el vínculo no alcanza', async () => {
		portalConUnidad()
		portalPost.mockResolvedValue({
			data: undefined,
			error: { detail: 'no tenés un vínculo vigente con esa unidad' },
		})

		renderPortal()

		fireEvent.click(await screen.findByRole('button', { name: 'Abrir reclamo' }))
		fireEvent.change(await screen.findByLabelText('Categoría'), { target: { value: 'humedad' } })
		fireEvent.change(screen.getByLabelText('¿Qué pasa?'), { target: { value: 'Prueba' } })
		fireEvent.click(screen.getByRole('button', { name: 'Enviar reclamo' }))

		expect(await screen.findByRole('alert')).toHaveTextContent('vínculo vigente')
	})

	it('no ofrece abrir reclamo sin unidades con vínculo', async () => {
		portalHome([])

		renderPortal()

		expect(await screen.findByText(/No tenés unidades con vínculo vigente/)).toBeInTheDocument()
		expect(screen.queryByRole('button', { name: 'Abrir reclamo' })).not.toBeInTheDocument()
	})
})

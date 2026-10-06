import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatCents } from '@/lib/format'
import { LiquidacionWizard } from './LiquidacionWizard'

// Intl es-AR emite nbsp entre $ y el número; el matcher de testing-library no lo normaliza.
const money = (cents: number) => formatCents(cents).replace(/\u00a0/g, ' ')

const detailGet = vi.fn()
const patchPut = vi.fn()
const confirmarPost = vi.fn()
const publicarPost = vi.fn()
const anularPost = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}') return detailGet()
			throw new Error(`GET sin mock: ${path}`)
		},
		PATCH: (path: string, args: unknown) => {
			if (path === '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}') return patchPut(args)
			throw new Error(`PATCH sin mock: ${path}`)
		},
		POST: (path: string, args: unknown) => {
			if (path === '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/confirmar') return confirmarPost(args)
			if (path === '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/publicar') return publicarPost(args)
			if (path === '/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/anular') return anularPost(args)
			throw new Error(`POST sin mock: ${path}`)
		},
	},
}))

vi.mock('@/auth/AuthProvider', () => ({
	useAuth: () => ({
		me: {
			permissions: ['expensas.create', 'expensas.confirm', 'expensas.publish'],
		},
	}),
}))

const LIQ = {
	id: 'liq-1',
	periodo: '202608',
	estado: 'calculada',
	vencimiento_1: '2026-08-10',
	vencimiento_2: null,
	total_gastos_cents: 150000,
	total_distribuido_cents: 145000,
	unidades_alcanzadas: 12,
	version: 2,
}

function renderWizard() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter initialEntries={['/app/consorcios/cons-1/liquidaciones/liq-1']}>
				<Routes>
					<Route
						path="/app/consorcios/:consorcioId/liquidaciones/:liquidacionId"
						element={<LiquidacionWizard />}
					/>
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>,
	)
}

describe('LiquidacionWizard', () => {
	beforeEach(() => {
		detailGet.mockReset()
		patchPut.mockReset()
		confirmarPost.mockReset()
		publicarPost.mockReset()
		anularPost.mockReset()
	})

	it('carga el detalle de la liquidacion', async () => {
		detailGet.mockResolvedValue({ data: LIQ })
		renderWizard()
		expect(await screen.findByText('Liquidación 202608')).toBeInTheDocument()
		expect(screen.getByText('calculada')).toBeInTheDocument()
		expect(screen.getByText('Total gastos')).toBeInTheDocument()
		expect(screen.getByText(money(150000))).toBeInTheDocument()
	})

	it('deshabilita anular salvo estado reversible', async () => {
		detailGet.mockResolvedValue({ data: LIQ })
		renderWizard()
		const boton = await screen.findByRole('button', { name: 'Anular liquidación' })
		expect(boton).toBeDisabled()
	})

	it('anula con motivo cuando esta publicada', async () => {
		detailGet.mockResolvedValue({ data: { ...LIQ, estado: 'publicada', version: 4 } })
		anularPost.mockResolvedValue({ data: { ...LIQ, estado: 'anulada' } })
		renderWizard()
		const boton = await screen.findByRole('button', { name: 'Anular liquidación' })
		expect(boton).toBeDisabled()
		fireEvent.change(screen.getByLabelText(/motivo/i), { target: { value: 'Error de cálculo' } })
		expect(boton).not.toBeDisabled()
		fireEvent.click(boton)
		await waitFor(() => expect(anularPost).toHaveBeenCalledTimes(1))
		expect(anularPost.mock.calls[0][0]).toEqual({
			params: {
				path: { consorcioId: 'cons-1', liquidacionId: 'liq-1' },
				header: { 'If-Match': '4' },
			},
			body: { motivo: 'Error de cálculo' },
		})
	})

	it('confirmar envia POST con If-Match e Idempotency-Key', async () => {
		detailGet.mockResolvedValue({ data: LIQ })
		confirmarPost.mockResolvedValue({ data: { ...LIQ, estado: 'confirmada' } })
		renderWizard()
		fireEvent.click(await screen.findByRole('button', { name: 'Confirmar' }))
		await waitFor(() => expect(confirmarPost).toHaveBeenCalledTimes(1))
		const [args] = confirmarPost.mock.calls[0]
		expect(args.params.path).toEqual({ consorcioId: 'cons-1', liquidacionId: 'liq-1' })
		expect(args.params.header['If-Match']).toBe('2')
		expect(args.params.header['Idempotency-Key']).toEqual(expect.any(String))
	})

	it('publicar envia POST cuando esta confirmada', async () => {
		detailGet.mockResolvedValue({ data: { ...LIQ, estado: 'confirmada', version: 3 } })
		publicarPost.mockResolvedValue({ data: { ...LIQ, estado: 'publicada' } })
		renderWizard()
		fireEvent.click(await screen.findByRole('button', { name: 'Publicar' }))
		await waitFor(() => expect(publicarPost).toHaveBeenCalledTimes(1))
		const [args] = publicarPost.mock.calls[0]
		expect(args.params.header['If-Match']).toBe('3')
	})

	it('guardar vencimientos hace PATCH en borrador', async () => {
		detailGet.mockResolvedValue({ data: { ...LIQ, estado: 'borrador', version: 1 } })
		patchPut.mockResolvedValue({ data: LIQ })
		renderWizard()
		const boton = await screen.findByRole('button', { name: 'Guardar cambios' })
		expect(boton).not.toBeDisabled()
		const venc1 = screen.getByLabelText(/vencimiento 1/i)
		await waitFor(() => expect(venc1).toHaveValue('2026-08-10'))
		fireEvent.submit(boton.closest('form')!)
		await waitFor(() => expect(patchPut).toHaveBeenCalledTimes(1))
		expect(patchPut.mock.calls[0][0]).toEqual({
			params: {
				path: { consorcioId: 'cons-1', liquidacionId: 'liq-1' },
				header: { 'If-Match': '1' },
			},
			body: { vencimiento_1: '2026-08-10', vencimiento_2: null },
		})
	})
})

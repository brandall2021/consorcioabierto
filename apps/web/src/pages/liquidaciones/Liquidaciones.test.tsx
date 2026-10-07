import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatCents } from '@/lib/format'
import { Liquidaciones } from './Liquidaciones'

// Intl es-AR emite nbsp entre $ y el número; el matcher de testing-library no lo normaliza.
const money = (cents: number) => formatCents(cents).replace(/\u00a0/g, ' ')

const listGet = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/consorcios/{id}/liquidaciones') return listGet()
			throw new Error(`GET sin mock: ${path}`)
		},
	},
}))

const permisos = { create: true }
vi.mock('@/auth/AuthProvider', () => ({
	useAuth: () => ({
		me: {
			permissions: [...(permisos.create ? ['expensas.create'] : [])],
		},
	}),
}))

const LIQ_CALCULADA = {
	id: '11111111-1111-4111-8111-111111111111',
	periodo: '202608',
	estado: 'calculada',
	vencimiento_1: '2026-08-10',
	total_gastos_cents: 150000,
	total_distribuido_cents: 145000,
	unidades_alcanzadas: 12,
	version: 2,
}

const LIQ_BORRADOR = {
	id: '22222222-2222-4222-8222-222222222222',
	periodo: '202609',
	estado: 'borrador',
	vencimiento_1: '2026-09-10',
	total_gastos_cents: 0,
	total_distribuido_cents: 0,
	unidades_alcanzadas: 0,
	version: 1,
}

function renderLiquidaciones() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter initialEntries={['/app/consorcios/cons-1/liquidaciones']}>
				<Routes>
					<Route path="/app/consorcios/:consorcioId/liquidaciones" element={<Liquidaciones />} />
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>,
	)
}

describe('Liquidaciones', () => {
	beforeEach(() => {
		listGet.mockReset()
		permisos.create = true
	})

	it('lista liquidaciones con periodo, estado e importes', async () => {
		listGet.mockResolvedValue({ data: { data: [LIQ_CALCULADA, LIQ_BORRADOR] } })
		renderLiquidaciones()
		expect(await screen.findByText('202608')).toBeInTheDocument()
		expect(screen.getByText('calculada')).toBeInTheDocument()
		expect(screen.getByText('202609')).toBeInTheDocument()
		expect(screen.getByText('borrador')).toBeInTheDocument()
		expect(screen.getByText(money(150000))).toBeInTheDocument()
	})

	it('linkea cada liquidacion al wizard', async () => {
		listGet.mockResolvedValue({ data: { data: [LIQ_CALCULADA] } })
		renderLiquidaciones()
		const link = await screen.findByRole('link', { name: '202608' })
		expect(link).toHaveAttribute('href', `/app/consorcios/cons-1/liquidaciones/${LIQ_CALCULADA.id}`)
	})

	it('ofrece crear liquidacion con permiso expensas.create', async () => {
		listGet.mockResolvedValue({ data: { data: [LIQ_CALCULADA] } })
		renderLiquidaciones()
		expect(await screen.findByRole('button', { name: 'Nueva liquidación' })).toBeInTheDocument()
	})

	it('oculta la creacion sin permiso expensas.create', async () => {
		permisos.create = false
		listGet.mockResolvedValue({ data: { data: [LIQ_CALCULADA] } })
		renderLiquidaciones()
		await screen.findByText('202608')
		expect(screen.queryByRole('button', { name: 'Nueva liquidación' })).toBeNull()
		expect(screen.getByText(/no tenés permiso/i)).toBeInTheDocument()
	})
})

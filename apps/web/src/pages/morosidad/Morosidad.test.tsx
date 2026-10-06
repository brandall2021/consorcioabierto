import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatCents } from '@/lib/format'
import { Morosidad } from './Morosidad'

// Intl es-AR emite nbsp entre $ y el número; el matcher de testing-library no lo normaliza.
const money = (cents: number) => formatCents(cents).replace(/\u00a0/g, ' ')

const morosidadGet = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/consorcios/{id}/morosidad') return morosidadGet()
			throw new Error(`GET sin mock: ${path}`)
		},
	},
}))

const ITEMS = [
	{
		unidad_id: 'uf-1',
		unidad_codigo: '3A',
		saldo_vencido_cents: 62000,
		cantidad_cargos: 3,
		vencido_desde: '2026-07-10',
	},
	{
		unidad_id: 'uf-2',
		unidad_codigo: '3B',
		saldo_vencido_cents: 13000,
		cantidad_cargos: 1,
		vencido_desde: '2026-09-01',
	},
]

const META = { request_id: 'req-1', total_saldo_cents: 75000, total_cargos_vencidos: 4 }

function renderMorosidad() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter initialEntries={['/app/consorcios/cons-1/morosidad']}>
				<Routes>
					<Route path="/app/consorcios/:consorcioId/morosidad" element={<Morosidad />} />
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>,
	)
}

describe('Morosidad', () => {
	beforeEach(() => {
		morosidadGet.mockReset()
	})

	it('lista morosos con saldo vencido y vencido_desde formateados', async () => {
		morosidadGet.mockResolvedValue({
			data: { data: ITEMS, meta: META },
		})
		renderMorosidad()
		expect(await screen.findByText('3A')).toBeInTheDocument()
		expect(screen.getByText(money(62000))).toBeInTheDocument()
		expect(screen.getByText('2026-07-10')).toBeInTheDocument()
		expect(screen.getByText('Crítica')).toBeInTheDocument()
		expect(screen.getByText(money(13000))).toBeInTheDocument()
		expect(screen.getByText('Seguimiento')).toBeInTheDocument()
	})

	it('muestra los totales del meta', async () => {
		morosidadGet.mockResolvedValue({
			data: { data: ITEMS, meta: META },
		})
		renderMorosidad()
		expect(await screen.findByText('UFs con mora')).toBeInTheDocument()
		expect(screen.getByText('2')).toBeInTheDocument()
		expect(screen.getByText(money(75000))).toBeInTheDocument()
		expect(screen.getByText('Cargos vencidos')).toBeInTheDocument()
		expect(screen.getByText('4')).toBeInTheDocument()
	})

	it('muestra sin mora cuando no hay vencidos', async () => {
		morosidadGet.mockResolvedValue({
			data: { data: [], meta: { request_id: 'req-1', total_saldo_cents: 0, total_cargos_vencidos: 0 } },
		})
		renderMorosidad()
		expect(await screen.findByText('Sin mora')).toBeInTheDocument()
	})
})

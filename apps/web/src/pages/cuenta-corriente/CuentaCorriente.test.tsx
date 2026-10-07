import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatCents } from '@/lib/format'
import { CuentaCorriente } from './CuentaCorriente'

// Intl es-AR emite nbsp entre $ y el número; el matcher de testing-library no lo normaliza.
const money = (cents: number) => formatCents(cents).replace(/\u00a0/g, ' ')

const unidadesGet = vi.fn()
const cuentaGet = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string, args?: unknown) => {
			if (path === '/consorcios/{id}/unidades') return unidadesGet(args)
			if (path === '/consorcios/{id}/unidades/{unidadId}/cuenta-corriente') return cuentaGet(args)
			throw new Error(`GET sin mock: ${path}`)
		},
	},
}))

const UNIDADES = [
	{ id: 'uf-1', codigo: '3A', tipo: 'departamento' },
	{ id: 'uf-2', codigo: '3B', tipo: 'departamento' },
]

const CUENTA = {
	data: [
		{
			id: 'm1',
			unidad_id: 'uf-1',
			tipo: 'cargo',
			fecha_efectiva: '2026-09-01',
			debit_cents: 50000,
			credit_cents: 0,
			currency: 'ARS',
			referencia: 'EXP-202609',
		},
		{
			id: 'm2',
			unidad_id: 'uf-1',
			tipo: 'credito',
			fecha_efectiva: '2026-09-15',
			debit_cents: 0,
			credit_cents: 20000,
			currency: 'ARS',
			referencia: null,
		},
	],
	meta: { request_id: 'req-1', saldo_cents: 30000 },
}

function renderCuentaCorriente() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter initialEntries={['/app/consorcios/cons-1/cuenta-corriente']}>
				<Routes>
					<Route path="/app/consorcios/:consorcioId/cuenta-corriente" element={<CuentaCorriente />} />
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>,
	)
}

describe('CuentaCorriente', () => {
	beforeEach(() => {
		unidadesGet.mockReset()
		cuentaGet.mockReset()
		unidadesGet.mockResolvedValue({ data: { data: UNIDADES } })
		cuentaGet.mockResolvedValue({ data: CUENTA })
	})

	it('selecciona la primera UF y muestra el saldo del meta', async () => {
		renderCuentaCorriente()
		expect(await screen.findByRole('combobox', { name: /unidad/i })).toBeInTheDocument()
		expect(await screen.findByText(money(30000))).toBeInTheDocument()
		const [args] = cuentaGet.mock.calls[0]
		expect(args.params.path).toEqual({ id: 'cons-1', unidadId: 'uf-1' })
	})

	it('lista movimientos con importes formateados', async () => {
		renderCuentaCorriente()
		expect(await screen.findByText(money(50000))).toBeInTheDocument()
		expect(screen.getByText(money(20000))).toBeInTheDocument()
		expect(screen.getByText('EXP-202609')).toBeInTheDocument()
		expect(screen.getByText('cargo')).toBeInTheDocument()
	})

	it('recarga la cuenta al cambiar de UF', async () => {
		renderCuentaCorriente()
		const select = await screen.findByRole('combobox', { name: /unidad/i })
		await screen.findByText(money(30000))
		fireEvent.change(select, { target: { value: 'uf-2' } })
		await waitFor(() => expect(cuentaGet).toHaveBeenCalledTimes(2))
		const [args] = cuentaGet.mock.calls[1]
		expect(args.params.path).toEqual({ id: 'cons-1', unidadId: 'uf-2' })
	})
})

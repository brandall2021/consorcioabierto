import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Portal } from './Portal'

const portalGet = vi.fn()
const logout = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/portal') return portalGet()
			throw new Error(`GET sin mock: ${path}`)
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
})

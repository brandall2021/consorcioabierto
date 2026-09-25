import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Observabilidad } from './Observabilidad'

const healthGet = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/health') return healthGet()
			throw new Error(`GET sin mock: ${path}`)
		},
	},
}))

function renderPage() {
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	})
	return render(
		<QueryClientProvider client={queryClient}>
			<Observabilidad />
		</QueryClientProvider>,
	)
}

describe('Observabilidad', () => {
	beforeEach(() => {
		healthGet.mockReset()
		healthGet.mockResolvedValue({ data: { status: 'ok' }, error: undefined })
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue(
				new Response(`# TYPE http_requests_total counter\nhttp_requests_total{method="GET",route="/portal",status="200"} 7\n`),
			),
		)
	})

	it('muestra la salud y métricas del sistema', async () => {
		renderPage()

		await waitFor(() => expect(screen.getByText('Señales vivas del sistema')).toBeInTheDocument())
		expect(screen.getByText('API healthy')).toBeInTheDocument()
		expect(screen.getByRole('button', { name: 'Actualizar' })).toBeInTheDocument()
		expect(screen.getByText(/Sincronizado/)).toBeInTheDocument()
		expect(screen.getByText('Requests totales')).toBeInTheDocument()
		expect(screen.getByText('7')).toBeInTheDocument()
	})
})

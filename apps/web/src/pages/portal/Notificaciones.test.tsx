import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Notificaciones } from './Notificaciones'

const notificacionesGet = vi.fn()
const notificacionesPost = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/portal/notificaciones') return notificacionesGet()
			throw new Error(`GET sin mock: ${path}`)
		},
		POST: (path: string, options?: unknown) => {
			if (path === '/portal/notificaciones/{id}/leer') return notificacionesPost(options)
			throw new Error(`POST sin mock: ${path}`)
		},
	},
}))

function renderNotificaciones() {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<Notificaciones />
		</QueryClientProvider>,
	)
}

function notificacion(overrides: Record<string, unknown> = {}) {
	return {
		id: '11111111-1111-4111-8111-111111111111',
		tipo: 'comunicado' as const,
		titulo: 'Corte de agua',
		cuerpo: 'De 9 a 13 hs.',
		recurso_type: 'comunicado',
		recurso_id: '22222222-2222-4222-8222-222222222222',
		leida_at: null,
		created_at: '2026-10-02T12:00:00Z',
		...overrides,
	}
}

describe('Notificaciones', () => {
	beforeEach(() => {
		notificacionesGet.mockReset()
		notificacionesPost.mockReset()
	})

	it('lista las notificaciones con su cuerpo', async () => {
		notificacionesGet.mockResolvedValue({
			data: { data: [notificacion()], meta: { request_id: 'r', no_leidas: 1 } },
		})
		renderNotificaciones()
		expect(await screen.findByText('Corte de agua')).toBeTruthy()
		expect(screen.getByText('De 9 a 13 hs.')).toBeTruthy()
	})

	it('muestra el contador de no leidas del meta', async () => {
		notificacionesGet.mockResolvedValue({
			data: { data: [notificacion()], meta: { request_id: 'r', no_leidas: 3 } },
		})
		renderNotificaciones()
		expect(await screen.findByText('3 sin leer')).toBeTruthy()
	})

	it('no muestra contador cuando no hay pendientes', async () => {
		notificacionesGet.mockResolvedValue({
			data: { data: [notificacion({ leida_at: '2026-10-03T09:00:00Z' })], meta: { request_id: 'r', no_leidas: 0 } },
		})
		renderNotificaciones()
		expect(await screen.findByText('Corte de agua')).toBeTruthy()
		expect(screen.queryByText(/sin leer/)).toBeNull()
	})

	it('no ofrece marcar leida una notificacion ya leida', async () => {
		notificacionesGet.mockResolvedValue({
			data: { data: [notificacion({ leida_at: '2026-10-03T09:00:00Z' })], meta: { request_id: 'r', no_leidas: 0 } },
		})
		renderNotificaciones()
		await screen.findByText('Corte de agua')
		expect(screen.queryByRole('button', { name: /marcar le/i })).toBeNull()
	})

	it('marca como leida y refresca la bandeja', async () => {
		notificacionesGet
			.mockResolvedValueOnce({
				data: { data: [notificacion()], meta: { request_id: 'r', no_leidas: 1 } },
			})
			.mockResolvedValue({
				data: { data: [notificacion({ leida_at: '2026-10-03T09:00:00Z' })], meta: { request_id: 'r', no_leidas: 0 } },
			})
		notificacionesPost.mockResolvedValue({ data: notificacion({ leida_at: '2026-10-03T09:00:00Z' }) })

		renderNotificaciones()
		const boton = await screen.findByRole('button', { name: /marcar le/i })
		fireEvent.click(boton)

		await waitFor(() => expect(notificacionesPost).toHaveBeenCalledTimes(1))
		// El mock recibe (path, options) y delega solo options al vi.fn().
		expect(notificacionesPost.mock.calls[0][0]).toEqual({
			params: { path: { id: '11111111-1111-4111-8111-111111111111' } },
		})
		// El contador se recalcula contra el servidor, no se resta a mano.
		await waitFor(() => expect(screen.queryByText(/sin leer/)).toBeNull())
	})

	it('muestra estado vacio sin notificaciones', async () => {
		notificacionesGet.mockResolvedValue({ data: { data: [], meta: { request_id: 'r', no_leidas: 0 } } })
		renderNotificaciones()
		expect(await screen.findByText('Sin notificaciones')).toBeTruthy()
	})

	it('muestra el error sin romper el portal', async () => {
		notificacionesGet.mockResolvedValue({ data: undefined, error: { message: 'boom' } })
		renderNotificaciones()
		expect(await screen.findByText(/No se pudieron cargar las notificaciones/)).toBeTruthy()
	})
})
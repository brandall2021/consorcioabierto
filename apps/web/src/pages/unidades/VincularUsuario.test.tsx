import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { VincularUsuario } from './VincularUsuario'
import type { components } from '@/api/generated.d'

type Persona = components['schemas']['Persona']

// jsdom no implementa el dialogo nativo que usa Modal.
beforeAll(() => {
	HTMLDialogElement.prototype.showModal = function showModal() {
		this.open = true
	}
	HTMLDialogElement.prototype.close = function close() {
		this.open = false
	}
})

const membersGet = vi.fn()
const personaPut = vi.fn()
const personaDelete = vi.fn()

vi.mock('@/api/client', () => ({
	client: {
		GET: (path: string) => {
			if (path === '/tenant/members') return membersGet()
			throw new Error(`GET sin mock: ${path}`)
		},
		PUT: (path: string, options?: unknown) => {
			if (path === '/tenant/personas/{personaId}/usuario') return personaPut(options)
			throw new Error(`PUT sin mock: ${path}`)
		},
		DELETE: (path: string, options?: unknown) => {
			if (path === '/tenant/personas/{personaId}/usuario') return personaDelete(options)
			throw new Error(`DELETE sin mock: ${path}`)
		},
	},
}))

const persona: Persona = {
	id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
	nombre: 'María Test',
	documento: '30111222',
	email: 'maria@ejemplo.com',
	telefono: null,
	user_id: null,
}

const personaVinculada = { ...persona, user_id: '11111111-1111-4111-8111-111111111111' }

const miembros = [
	{
		user_id: '11111111-1111-4111-8111-111111111111',
		email: 'a@tenant-a.com',
		nombre: 'Ana',
		membresia: 'active',
		estado: 'active',
		vinculado: null,
	},
	{
		user_id: '22222222-2222-4222-8222-222222222222',
		email: 'b@tenant-a.com',
		nombre: 'Beto',
		membresia: 'active',
		estado: 'active',
		vinculado: null,
	},
]

function renderUI(p = persona) {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
	return render(
		<QueryClientProvider client={queryClient}>
			<VincularUsuario persona={p} />
		</QueryClientProvider>,
	)
}

describe('VincularUsuario', () => {
	beforeEach(() => {
		vi.clearAllMocks()
		membersGet.mockResolvedValue({ data: { data: miembros, meta: { request_id: 'r1' } } })
		personaPut.mockResolvedValue({ data: { ...persona, user_id: miembros[0].user_id } })
		personaDelete.mockResolvedValue({ data: persona })
	})

	it('muestra el estado sin vínculo y abre el selector de miembros', async () => {
		renderUI()

		expect(screen.getByText('Sin vincular')).toBeInTheDocument()
		fireEvent.click(screen.getByRole('button', { name: 'Vincular usuario' }))

		await waitFor(() => expect(membersGet).toHaveBeenCalledTimes(1))
		expect(await screen.findByRole('option', { name: /Ana · a@tenant-a.com/ })).toBeInTheDocument()
		expect(screen.getByRole('option', { name: /Beto · b@tenant-a.com/ })).toBeInTheDocument()
	})

	it('envía el usuario elegido y no elige el tenant', async () => {
		renderUI()
		fireEvent.click(screen.getByRole('button', { name: 'Vincular usuario' }))

		const select = await screen.findByRole('combobox')
		fireEvent.change(select, { target: { value: miembros[1].user_id } })
		fireEvent.click(screen.getByRole('button', { name: 'Vincular' }))

		await waitFor(() => expect(personaPut).toHaveBeenCalledTimes(1))
		expect(personaPut.mock.calls[0][0]).toMatchObject({
			params: { path: { personaId: persona.id } },
			body: { usuario_id: miembros[1].user_id },
		})
	})

	it('no habilita el botón hasta que se elige un miembro', async () => {
		renderUI()
		fireEvent.click(screen.getByRole('button', { name: 'Vincular usuario' }))

		const submit = await screen.findByRole('button', { name: 'Vincular' })
		expect(submit).toBeDisabled()

		fireEvent.change(screen.getByRole('combobox'), { target: { value: miembros[0].user_id } })
		expect(submit).toBeEnabled()
	})

	it('muestra el error del servidor sin cerrarse', async () => {
		personaPut.mockResolvedValue({
			error: { detail: 'el usuario ya está vinculado a otra persona' },
		})
		renderUI()
		fireEvent.click(screen.getByRole('button', { name: 'Vincular usuario' }))

		fireEvent.change(await screen.findByRole('combobox'), { target: { value: miembros[0].user_id } })
		fireEvent.click(screen.getByRole('button', { name: 'Vincular' }))

		expect(
			await screen.findByText('el usuario ya está vinculado a otra persona'),
		).toBeInTheDocument()
	})

	it('una persona ya vinculada muestra el estado y permite desvincular', async () => {
		renderUI(personaVinculada)

		expect(screen.getByText('Vinculada')).toBeInTheDocument()
		fireEvent.click(screen.getByRole('button', { name: 'Desvincular' }))

		await waitFor(() => expect(personaDelete).toHaveBeenCalledTimes(1))
		expect(personaDelete.mock.calls[0][0]).toMatchObject({
			params: { path: { personaId: persona.id } },
		})
	})

	it('avisa cuando el tenant no tiene miembros', async () => {
		membersGet.mockResolvedValue({ data: { data: [], meta: { request_id: 'r1' } } })
		renderUI()
		fireEvent.click(screen.getByRole('button', { name: 'Vincular usuario' }))

		expect(
			await screen.findByText('Este tenant todavía no tiene miembros para vincular.'),
		).toBeInTheDocument()
	})
})
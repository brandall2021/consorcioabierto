import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import { AppLayout } from './AppLayout'

const logout = vi.fn()

vi.mock('@/auth/AuthProvider', () => ({
	useAuth: () => ({
		me: { permissions: ['auditoria.read'], membership: { tenant_name: 'Torre A' } },
		logout,
	}),
}))

vi.mock('@/lib/featureFlags', () => ({
	isEnabled: () => true,
}))

describe('AppLayout', () => {
	beforeEach(() => {
		logout.mockReset()
	})

	it('expone un skip link y un main identificable', () => {
		render(
			<MemoryRouter initialEntries={['/app']}>
				<Routes>
					<Route path="/app" element={<AppLayout />}>
						<Route index element={<div>Inicio</div>} />
					</Route>
				</Routes>
			</MemoryRouter>,
		)

		expect(screen.getByRole('link', { name: 'Saltar al contenido' })).toHaveAttribute('href', '#main-content')
		expect(screen.getByRole('main')).toHaveAttribute('id', 'main-content')
	})
})

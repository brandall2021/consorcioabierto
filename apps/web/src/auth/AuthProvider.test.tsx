import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider, useAuth } from './AuthProvider'
import { Login } from '@/pages/login/Login'

const loginPost = vi.fn()
const selectTenantPost = vi.fn()
const logoutPost = vi.fn()
const meGet = vi.fn()

vi.mock('@/api/client', () => ({
  client: {
    POST: (path: string, ...args: unknown[]) => {
      if (path === '/auth/login') return loginPost(...args)
      if (path === '/auth/select-tenant') return selectTenantPost(...args)
      if (path === '/auth/logout') return logoutPost(...args)
      throw new Error(`POST sin mock: ${path}`)
    },
    GET: (path: string) => {
      if (path === '/me') return meGet()
      throw new Error(`GET sin mock: ${path}`)
    },
  },
}))

const user = { id: 'u1', email: 'a@x.com', name: 'Ana', status: 'active', mfa_enabled: false }
const membership = {
  id: 'm1',
  tenant_id: 't1',
  tenant_name: 'Torre A',
  roles: ['tenant_admin'],
}

function StatusProbe() {
  const { status } = useAuth()
  return <output data-testid="status">{status}</output>
}

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/login']}>
        <AuthProvider>
          <StatusProbe />
          <Login />
        </AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AuthProvider', () => {
  beforeEach(() => {
    loginPost.mockReset()
    selectTenantPost.mockReset()
    logoutPost.mockReset()
    meGet.mockReset()
    meGet.mockResolvedValue({ data: null, error: { status: 401 } })
  })

  it('restaura sesión existente vía /me', async () => {
    meGet.mockResolvedValue({
      data: { user, membership, permissions: ['auditoria.read'] },
      error: undefined,
    })
    renderApp()
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('authenticated'))
  })

  it('login con una membresía elige el tenant automáticamente', async () => {
    loginPost.mockResolvedValue({ data: { user, memberships: [membership] }, error: undefined })
    selectTenantPost.mockResolvedValue({
      data: { user, membership, permissions: [] },
      error: undefined,
    })
    renderApp()
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('anon'))

    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Contraseña'), 'secreto')
    await userEvent.click(screen.getByRole('button', { name: 'Ingresar' }))

    await waitFor(() => expect(selectTenantPost).toHaveBeenCalledWith({
      body: { membership_id: 'm1' },
    }))
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('authenticated'))
  })

  it('login con varias membresías espera selección de tenant', async () => {
    const other = { ...membership, id: 'm2', tenant_name: 'Torre B' }
    loginPost.mockResolvedValue({
      data: { user, memberships: [membership, other] },
      error: undefined,
    })
    renderApp()
    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('anon'))

    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Contraseña'), 'secreto')
    await userEvent.click(screen.getByRole('button', { name: 'Ingresar' }))

    await waitFor(() => expect(screen.getByTestId('status')).toHaveTextContent('select-tenant'))
  })
})

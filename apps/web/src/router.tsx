import { createBrowserRouter, Navigate, Outlet } from 'react-router'
import { AuthProvider, useAuth } from '@/auth/AuthProvider'
import { AppLayout } from '@/components/layout/AppLayout'
import { Auditoria } from '@/pages/auditoria/Auditoria'
import { Consorcios } from '@/pages/consorcios/Consorcios'
import { ConsorcioLayout } from '@/pages/consorcios/ConsorcioLayout'
import { Proveedores } from '@/pages/proveedores/Proveedores'
import { Unidades } from '@/pages/unidades/Unidades'
import { Dashboard } from '@/pages/dashboard/Dashboard'
import { Login } from '@/pages/login/Login'
import { Perfil } from '@/pages/perfil/Perfil'

function RequireAuth() {
  const { status } = useAuth()
  if (status === 'loading') {
    return (
      <main className="grid min-h-screen place-items-center p-4 text-gray-500">
        Cargando…
      </main>
    )
  }
  if (status !== 'authenticated') {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}

export const router = createBrowserRouter([
  {
    element: (
      <AuthProvider>
        <Outlet />
      </AuthProvider>
    ),
    children: [
      { path: '/login', element: <Login /> },
      {
        element: <RequireAuth />,
        children: [
          {
            path: '/app',
            element: <AppLayout />,
            children: [
              { index: true, element: <Dashboard /> },
              { path: 'consorcios', element: <Consorcios /> },
              {
                path: 'consorcios/:consorcioId',
                element: <ConsorcioLayout />,
                children: [
                  { index: true, element: <Unidades /> },
                  { path: 'unidades', element: <Unidades /> },
                  { path: 'proveedores', element: <Proveedores /> },
                ],
              },
              { path: 'auditoria', element: <Auditoria /> },
              { path: 'perfil', element: <Perfil /> },
            ],
          },
        ],
      },
    ],
  },
  { path: '*', element: <Navigate to="/app" replace /> },
])

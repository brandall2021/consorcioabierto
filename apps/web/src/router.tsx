import { createBrowserRouter, Navigate } from 'react-router'
import { AppLayout } from '@/components/layout/AppLayout'
import { Dashboard } from '@/pages/dashboard/Dashboard'
import { Login } from '@/pages/login/Login'

export const router = createBrowserRouter([
  { path: '/login', element: <Login /> },
  {
    path: '/app',
    element: <AppLayout />,
    children: [{ index: true, element: <Dashboard /> }],
  },
  { path: '*', element: <Navigate to="/app" replace /> },
])

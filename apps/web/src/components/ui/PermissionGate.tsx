import type { ReactNode } from 'react'
import { useAuth } from '@/auth/AuthProvider'

export function PermissionGate({
  permission,
  children,
  fallback = <p className="text-sm text-gray-500">No tenés permiso para ver esta sección.</p>,
}: {
  permission: string
  children: ReactNode
  fallback?: ReactNode
}) {
  const { me } = useAuth()
  if (!me?.permissions.includes(permission)) {
    return <>{fallback}</>
  }
  return <>{children}</>
}
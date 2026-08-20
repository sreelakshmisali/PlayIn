import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import type { Role } from '@/lib/auth'

interface ProtectedRouteProps {
  /** When set, the signed-in user must hold one of these roles. */
  roles?: Role[]
}

/**
 * A route guard. Mount it as a layout route and its children require a session.
 *
 * It renders nothing while the session is still resolving. Redirecting during
 * that window would bounce a signed-in user to the login screen on every hard
 * refresh, before the token has been checked.
 */
export function ProtectedRoute({ roles }: ProtectedRouteProps) {
  const { status, user } = useAuth()
  const location = useLocation()

  if (status === 'loading') {
    return <RouteFallback />
  }

  if (status === 'anonymous' || user === null) {
    // The attempted path rides along so the login screen can return the user to
    // where they were headed.
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  }

  if (roles && !roles.includes(user.role)) {
    return <Forbidden />
  }

  return <Outlet />
}

function RouteFallback() {
  return (
    <div className="flex min-h-48 items-center justify-center" role="status" aria-live="polite">
      <span className="text-sm text-neutral-500">Checking your session</span>
    </div>
  )
}

function Forbidden() {
  return (
    <section className="mx-auto max-w-md text-center">
      <p className="text-sm font-medium text-pitch-600">403</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-tight text-neutral-900">
        You do not have access
      </h1>
      <p className="mt-3 text-neutral-600">
        This area is limited to accounts with a different role.
      </p>
    </section>
  )
}

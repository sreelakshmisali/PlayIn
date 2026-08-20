import { Link, NavLink } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'

const navigation = [
  { to: '/', label: 'Home' },
  { to: '/turfs', label: 'Turfs' },
  { to: '/status', label: 'Status' },
]

export function Header() {
  const { status, user, logout } = useAuth()

  return (
    <header className="border-b border-neutral-200 bg-white">
      <div className="mx-auto flex h-16 max-w-5xl items-center justify-between gap-3 px-4 sm:px-6">
        <NavLink to="/" className="flex items-center gap-2">
          <span className="grid size-8 place-items-center rounded-lg bg-pitch-600 text-sm font-bold text-white">
            P
          </span>
          <span className="text-lg font-semibold tracking-tight">PlayHub</span>
        </NavLink>

        <div className="flex items-center gap-1">
          {/* The primary nav is hidden on the narrowest screens so the session
              controls, which are what someone actually reaches for, keep their
              tap targets. */}
          <nav className="hidden items-center gap-1 sm:flex" aria-label="Main">
            {navigation.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/'}
                className={({ isActive }) =>
                  [
                    'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-pitch-50 text-pitch-700'
                      : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900',
                  ].join(' ')
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>

          {status === 'authenticated' && user ? (
            <>
              {user.role === 'PLAYER' && (
                <NavLink
                  to="/profile"
                  className={({ isActive }) =>
                    [
                      'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      isActive
                        ? 'bg-pitch-50 text-pitch-700'
                        : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900',
                    ].join(' ')
                  }
                >
                  Profile
                </NavLink>
              )}
              {user.role === 'OWNER' && (
                <NavLink
                  to="/owner/turfs"
                  className={({ isActive }) =>
                    [
                      'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      isActive
                        ? 'bg-pitch-50 text-pitch-700'
                        : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900',
                    ].join(' ')
                  }
                >
                  My Turfs
                </NavLink>
              )}
              {user.role === 'ADMIN' && (
                <NavLink
                  to="/admin"
                  className={({ isActive }) =>
                    [
                      'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      isActive
                        ? 'bg-pitch-50 text-pitch-700'
                        : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900',
                    ].join(' ')
                  }
                >
                  Admin
                </NavLink>
              )}
              <NavLink
                to="/account"
                className={({ isActive }) =>
                  [
                    'max-w-32 truncate rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-pitch-50 text-pitch-700'
                      : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900',
                  ].join(' ')
                }
              >
                {user.full_name}
              </NavLink>
              <button
                type="button"
                onClick={() => void logout()}
                className="rounded-md px-3 py-2 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
              >
                Sign out
              </button>
            </>
          ) : (
            status === 'anonymous' && (
              <>
                <NavLink
                  to="/login"
                  className="rounded-md px-3 py-2 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
                >
                  Sign in
                </NavLink>
                <Link
                  to="/register"
                  className="rounded-md bg-pitch-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-pitch-700"
                >
                  Sign up
                </Link>
              </>
            )
          )}
        </div>
      </div>
    </header>
  )
}

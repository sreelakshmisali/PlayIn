import { Link } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'

export function HomePage() {
  const { status, user } = useAuth()

  return (
    <section className="max-w-2xl">
      <p className="text-sm font-medium text-pitch-600">Phase 1</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-tight text-neutral-900 sm:text-3xl">
        {status === 'authenticated' && user
          ? `Welcome back, ${user.full_name}`
          : 'PlayHub accounts are live'}
      </h1>
      <p className="mt-4 text-neutral-600">
        Registration, sign in and session handling work end to end. Turfs, bookings, teams
        and tournaments are not built yet.
      </p>

      <div className="mt-8 flex flex-col gap-3 sm:flex-row">
        {status === 'authenticated' ? (
          <PrimaryLink to="/account">View your account</PrimaryLink>
        ) : (
          <>
            <PrimaryLink to="/register">Create an account</PrimaryLink>
            <SecondaryLink to="/login">Sign in</SecondaryLink>
          </>
        )}
        <SecondaryLink to="/status">Check API status</SecondaryLink>
      </div>
    </section>
  )
}

function PrimaryLink({ to, children }: { to: string; children: string }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center justify-center rounded-lg bg-pitch-600 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-pitch-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-pitch-600"
    >
      {children}
    </Link>
  )
}

function SecondaryLink({ to, children }: { to: string; children: string }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center justify-center rounded-lg border border-neutral-300 bg-white px-4 py-3 text-sm font-medium text-neutral-800 transition-colors hover:bg-neutral-50"
    >
      {children}
    </Link>
  )
}

import type { ReactNode } from 'react'

interface AuthCardProps {
  title: string
  subtitle: string
  children: ReactNode
  footer: ReactNode
}

/**
 * The shell for the login and registration screens.
 *
 * Full bleed on mobile, a bordered card from the small breakpoint up. A card
 * with margins on a 360px screen wastes the width the form needs.
 */
export function AuthCard({ title, subtitle, children, footer }: AuthCardProps) {
  return (
    <div className="mx-auto w-full max-w-sm">
      <header className="text-center">
        <h1 className="text-2xl font-semibold tracking-tight text-neutral-900">{title}</h1>
        <p className="mt-2 text-sm text-neutral-600">{subtitle}</p>
      </header>

      <div className="mt-8 sm:rounded-xl sm:border sm:border-neutral-200 sm:bg-white sm:p-6 sm:shadow-sm">
        {children}
      </div>

      <p className="mt-6 text-center text-sm text-neutral-600">{footer}</p>
    </div>
  )
}

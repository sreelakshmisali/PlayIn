import { Outlet } from 'react-router-dom'

import { Footer } from '@/components/layout/Footer'
import { Header } from '@/components/layout/Header'

/**
 * The application shell. Every route renders inside it through <Outlet />, so
 * the header and footer mount once and survive navigation.
 */
export function AppLayout() {
  return (
    <div className="flex min-h-full flex-col">
      <Header />
      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8 sm:px-6 sm:py-10">
        <Outlet />
      </main>
      <Footer />
    </div>
  )
}

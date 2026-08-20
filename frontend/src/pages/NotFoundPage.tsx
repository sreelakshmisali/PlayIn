import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <section className="max-w-md">
      <p className="text-sm font-medium text-pitch-600">404</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-tight text-neutral-900">
        Page not found
      </h1>
      <p className="mt-3 text-neutral-600">That route does not exist.</p>
      <Link to="/" className="mt-6 inline-block text-sm font-medium text-pitch-700 underline">
        Back to home
      </Link>
    </section>
  )
}

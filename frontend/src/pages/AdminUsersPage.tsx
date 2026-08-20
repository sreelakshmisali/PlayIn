import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { fetchUsers, type UserPage } from '@/lib/admin'
import { ApiError } from '@/lib/api'

type State = { kind: 'loading' } | { kind: 'ready'; page: UserPage } | { kind: 'failed'; message: string }

const PAGE_SIZE = 20

/** Every account on the platform, paginated. Tap through to act on one. */
export function AdminUsersPage() {
  const [offset, setOffset] = useState(0)
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    const controller = new AbortController()
    setState({ kind: 'loading' })

    fetchUsers({ limit: PAGE_SIZE, offset }, controller.signal)
      .then((page) => setState({ kind: 'ready', page }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load users.',
        })
      })

    return () => controller.abort()
  }, [offset])

  return (
    <div className="mx-auto w-full max-w-md">
      <p className="text-sm">
        <Link to="/admin" className="text-pitch-700 underline">
          Admin
        </Link>
      </p>

      <h1 className="mt-1 text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">Users</h1>

      <div className="mt-6">
        {state.kind === 'loading' && <p className="text-neutral-500">Loading users.</p>}

        {state.kind === 'failed' && (
          <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
            <p className="text-sm text-red-800">{state.message}</p>
          </div>
        )}

        {state.kind === 'ready' && (
          <>
            <ul className="space-y-2">
              {state.page.users.map((user) => (
                <li key={user.id}>
                  <Link
                    to={`/admin/users/${user.id}`}
                    className="flex items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-white p-4 transition-colors hover:bg-neutral-50"
                  >
                    <div className="min-w-0">
                      <p className="truncate font-medium text-neutral-900">{user.full_name}</p>
                      <p className="truncate text-sm text-neutral-500">{user.email}</p>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {!user.is_active && (
                        <span className="rounded-full bg-neutral-100 px-2.5 py-1 text-xs font-medium text-neutral-500">
                          Deactivated
                        </span>
                      )}
                      <span className="rounded-full bg-pitch-50 px-2.5 py-1 text-xs font-medium text-pitch-700">
                        {user.role}
                      </span>
                    </div>
                  </Link>
                </li>
              ))}
            </ul>

            <Pagination
              offset={offset}
              limit={state.page.limit}
              total={state.page.total}
              onChange={setOffset}
            />
          </>
        )}
      </div>
    </div>
  )
}

function Pagination({
  offset,
  limit,
  total,
  onChange,
}: {
  offset: number
  limit: number
  total: number
  onChange: (offset: number) => void
}) {
  const from = total === 0 ? 0 : offset + 1
  const to = Math.min(offset + limit, total)

  return (
    <div className="mt-4 flex items-center justify-between gap-3">
      <p className="text-sm text-neutral-500">
        {from}–{to} of {total}
      </p>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => onChange(Math.max(0, offset - limit))}
          disabled={offset === 0}
          className="rounded-lg border border-neutral-300 bg-white px-3.5 py-2 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-100 disabled:cursor-not-allowed disabled:text-neutral-300"
        >
          Previous
        </button>
        <button
          type="button"
          onClick={() => onChange(offset + limit)}
          disabled={offset + limit >= total}
          className="rounded-lg border border-neutral-300 bg-white px-3.5 py-2 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-100 disabled:cursor-not-allowed disabled:text-neutral-300"
        >
          Next
        </button>
      </div>
    </div>
  )
}

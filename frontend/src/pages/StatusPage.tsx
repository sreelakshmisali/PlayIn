import { useCallback, useEffect, useState } from 'react'

import { ApiError } from '@/lib/api'
import { fetchHealth, type HealthReport } from '@/lib/health'

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; report: HealthReport }
  | { kind: 'failed'; message: string; requestId?: string }

/**
 * Calls GET /api/v1/health and renders the result. This page is the end-to-end
 * proof that the browser reaches the API and the API reaches PostgreSQL.
 */
export function StatusPage() {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const load = useCallback((signal?: AbortSignal) => {
    setState({ kind: 'loading' })

    fetchHealth(signal)
      .then((report) => setState({ kind: 'ready', report }))
      .catch((error: unknown) => {
        if (signal?.aborted) return

        if (error instanceof ApiError) {
          setState({
            kind: 'failed',
            message: error.message,
            ...(error.requestId ? { requestId: error.requestId } : {}),
          })
          return
        }
        setState({ kind: 'failed', message: 'Unexpected error.' })
      })
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    load(controller.signal)
    return () => controller.abort()
  }, [load])

  return (
    <section>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight text-neutral-900">API status</h1>
        <button
          type="button"
          onClick={() => load()}
          className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50"
        >
          Refresh
        </button>
      </div>

      <p className="mt-1 text-sm text-neutral-500">
        <code className="rounded bg-neutral-100 px-1.5 py-0.5">GET /api/v1/health</code>
      </p>

      <div className="mt-6">
        {state.kind === 'loading' && <p className="text-neutral-500">Checking the API.</p>}

        {state.kind === 'failed' && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4">
            <p className="font-medium text-red-800">The API did not respond correctly.</p>
            <p className="mt-1 text-sm text-red-700">{state.message}</p>
            {state.requestId && (
              <p className="mt-2 font-mono text-xs text-red-600">request id: {state.requestId}</p>
            )}
          </div>
        )}

        {state.kind === 'ready' && <HealthCard report={state.report} />}
      </div>
    </section>
  )
}

function HealthCard({ report }: { report: HealthReport }) {
  const healthy = report.status === 'ok'

  return (
    <div className="overflow-hidden rounded-lg border border-neutral-200 bg-white">
      <div className="flex items-center gap-3 border-b border-neutral-200 px-5 py-4">
        <span
          className={`size-2.5 rounded-full ${healthy ? 'bg-pitch-500' : 'bg-amber-500'}`}
          aria-hidden="true"
        />
        <span className="font-medium text-neutral-900">
          {healthy ? 'All systems operational' : 'Degraded'}
        </span>
      </div>

      <dl className="divide-y divide-neutral-100 text-sm">
        <Row label="Service" value={report.service} />
        <Row label="Version" value={report.version} />
        <Row label="Environment" value={report.env} />
        <Row label="Checked at" value={new Date(report.timestamp).toLocaleString()} />

        {Object.entries(report.checks).map(([name, check]) => (
          <Row
            key={name}
            label={name}
            value={
              check.status === 'ok'
                ? `ok in ${check.latency_ms} ms`
                : (check.error ?? check.status)
            }
          />
        ))}
      </dl>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-4 px-5 py-3">
      <dt className="capitalize text-neutral-500">{label}</dt>
      <dd className="text-right font-medium text-neutral-900">{value}</dd>
    </div>
  )
}

import { api } from '@/lib/api'

export type HealthStatus = 'ok' | 'degraded'

export interface HealthCheck {
  status: HealthStatus
  latency_ms: number
  error?: string
}

/** Response body of GET /api/v1/health. */
export interface HealthReport {
  status: HealthStatus
  service: string
  version: string
  env: string
  timestamp: string
  checks: Record<string, HealthCheck>
}

/**
 * Fetches the API health report.
 *
 * A degraded backend answers 503, which the client raises as an ApiError. The
 * caller handles that the same way as any other failure.
 */
export async function fetchHealth(signal?: AbortSignal): Promise<HealthReport> {
  return api.get<HealthReport>('/health', signal ? { signal } : {})
}

import { clearTokens, readTokens, writeTokens } from '../storage/tokens'
import type { ApiErrorBody, ApiFieldError } from '../types/api'
import { API_PREFIX, apiBaseUrl, apiTimeoutMs } from './config'

/**
 * Error thrown for any failed request. It carries the HTTP status and, when
 * the server sent one, the parsed error envelope, so callers can branch on
 * `error.status` or `error.code` instead of parsing strings.
 */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId: string | undefined
  readonly details: ApiFieldError[]

  constructor(status: number, body?: Partial<ApiErrorBody>) {
    super(body?.message ?? `Request failed with status ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.code = body?.code ?? 'unknown_error'
    this.requestId = body?.request_id
    this.details = body?.details ?? []
  }

  /** True when the request never reached the server or timed out. */
  get isNetworkError(): boolean {
    return this.status === 0
  }

  /** Maps the per-field reasons to `{ email: 'Email is required.' }`. */
  fieldErrors(): Record<string, string> {
    const out: Record<string, string> = {}
    for (const detail of this.details) {
      out[detail.field] ??= detail.message
    }
    return out
  }
}

export interface RequestOptions {
  /** Query string parameters. Undefined and null values are dropped. */
  query?: Record<string, string | number | boolean | undefined | null>
  /** Request body. Serialised as JSON. */
  body?: unknown
  headers?: Record<string, string>
  /** Omit the Authorization header. Used by the endpoints that mint tokens. */
  skipAuth?: boolean
  /** Do not attempt the 401 refresh-and-retry. Used by refresh and logout. */
  skipRefresh?: boolean
}

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

// Listeners notified when the session is beyond saving: the access token was
// rejected and the refresh token could not replace it. The auth context
// subscribes so the app signs out instead of retrying forever.
const sessionEndedListeners = new Set<() => void>()

/** Subscribes to the session ending. Returns an unsubscribe function. */
export function onSessionEnded(listener: () => void): () => void {
  sessionEndedListeners.add(listener)
  return () => sessionEndedListeners.delete(listener)
}

async function endSession(): Promise<void> {
  await clearTokens()
  for (const listener of sessionEndedListeners) listener()
}

function buildUrl(path: string, query: RequestOptions['query']): string {
  const url = `${apiBaseUrl}${API_PREFIX}${path}`

  if (!query) return url

  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null) {
      params.append(key, String(value))
    }
  }

  const search = params.toString()
  return search ? `${url}?${search}` : url
}

async function send(method: Method, path: string, options: RequestOptions): Promise<Response> {
  const { query, body, headers = {}, skipAuth } = options

  const url = buildUrl(path, query)

  console.log('🌐 API REQUEST:', method, url)
  console.log('📦 BODY:', body)

  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), apiTimeoutMs)

  const requestHeaders: Record<string, string> = { Accept: 'application/json', ...headers }

  if (!skipAuth) {
    const tokens = await readTokens()
    if (tokens) {
      requestHeaders.Authorization = `Bearer ${tokens.accessToken}`
    }
  }

  const init: RequestInit = {
    method,
    signal: controller.signal,
    headers: requestHeaders,
  }

  if (body !== undefined) {
    init.body = JSON.stringify(body)
    requestHeaders['Content-Type'] = 'application/json'
  }

  try {
    const response = await fetch(url, init)

    console.log('📥 API RESPONSE:', method, url, response.status)

    return response
  } catch (cause) {
    console.log('❌ API NETWORK ERROR:', method, url, cause)

    const message =
      cause instanceof Error && cause.name === 'AbortError'
        ? 'The request timed out.'
        : 'Could not reach the PlayHub API. Check the API URL and that your device can reach it.'

    throw new ApiError(0, { code: 'network_error', message })
  } finally {
    clearTimeout(timeout)
  }
}

// A single in-flight refresh, shared by every request that got a 401 at the
// same time. Without this, a screen issuing three parallel calls on an
// expired token would fire three refreshes, and rotation would invalidate
// two of them.
let inFlightRefresh: Promise<boolean> | null = null

/**
 * Exchanges the stored refresh token for a new pair. Resolves to whether the
 * session survived. On failure the tokens are cleared and subscribers are told.
 */
function refreshSession(): Promise<boolean> {
  inFlightRefresh ??= (async () => {
    const tokens = await readTokens()
    if (!tokens) return false

    try {
      const response = await send('POST', '/auth/refresh', {
        body: { refresh_token: tokens.refreshToken },
        skipAuth: true,
      })
      if (!response.ok) {
        await endSession()
        return false
      }

      const payload = (await response.json()) as { tokens?: { access_token: string; refresh_token: string } }
      if (!payload.tokens) {
        await endSession()
        return false
      }

      await writeTokens({
        accessToken: payload.tokens.access_token,
        refreshToken: payload.tokens.refresh_token,
      })
      return true
    } catch {
      // A network failure is not proof the session is over, so the tokens
      // are left alone and the original 401 surfaces to the caller.
      return false
    } finally {
      inFlightRefresh = null
    }
  })()

  return inFlightRefresh
}

async function toResult<T>(response: Response): Promise<T> {
  if (response.status === 204) {
    return undefined as T
  }

  const payload: unknown = await response.json().catch(() => undefined)

  if (!response.ok) {
    const envelope = (payload as { error?: Partial<ApiErrorBody> } | undefined)?.error
    throw new ApiError(response.status, envelope)
  }

  return payload as T
}

async function request<T>(method: Method, path: string, options: RequestOptions = {}): Promise<T> {
  const response = await send(method, path, options)

  // A 401 on a request that carried a token means the access token expired.
  // One refresh-and-retry recovers it; a second failure is a real sign-out.
  const canRetry = response.status === 401 && !options.skipAuth && !options.skipRefresh

  if (canRetry && (await refreshSession())) {
    return toResult<T>(await send(method, path, options))
  }
  if (canRetry) {
    await endSession()
  }

  return toResult<T>(response)
}

/**
 * The API client. Every network call in the app goes through this object, so
 * the bearer token, the 401 refresh and error handling live in one place
 * rather than at each screen.
 */
export const api = {
  get: <T>(path: string, options?: RequestOptions) => request<T>('GET', path, options),
  post: <T>(path: string, options?: RequestOptions) => request<T>('POST', path, options),
  put: <T>(path: string, options?: RequestOptions) => request<T>('PUT', path, options),
  patch: <T>(path: string, options?: RequestOptions) => request<T>('PATCH', path, options),
  delete: <T>(path: string, options?: RequestOptions) => request<T>('DELETE', path, options),
}

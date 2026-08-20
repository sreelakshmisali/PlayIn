/**
 * Runtime configuration, read from Vite environment variables.
 *
 * VITE_API_BASE_URL is empty by default. An empty base means requests go to the
 * same origin, which is what both the dev server proxy and the nginx container
 * are set up to forward. Point it at an absolute URL only when the API is on a
 * different host.
 */
export const config = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '',
  apiTimeoutMs: Number(import.meta.env.VITE_API_TIMEOUT_MS ?? 10_000),
} as const

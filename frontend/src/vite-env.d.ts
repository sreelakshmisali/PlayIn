/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the PlayHub API. Empty string means same-origin. */
  readonly VITE_API_BASE_URL?: string
  /** Request timeout in milliseconds. */
  readonly VITE_API_TIMEOUT_MS?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

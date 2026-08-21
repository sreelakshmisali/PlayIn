/** One rejected input field. Present only on validation failures. */
export interface ApiFieldError {
  field: string
  message: string
}

/** JSON error envelope returned by the API for every failed request. */
export interface ApiErrorBody {
  code: string
  message: string
  request_id?: string
  details?: ApiFieldError[]
}

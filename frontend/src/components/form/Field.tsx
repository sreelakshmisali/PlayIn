import { useId, type InputHTMLAttributes, type ReactNode } from 'react'

interface FieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'id' | 'className'> {
  label: string
  /** Server or client validation message for this field. */
  error?: string | undefined
  hint?: ReactNode
}

/**
 * A labelled text input.
 *
 * The input is 16px on mobile on purpose: iOS Safari zooms the viewport when a
 * focused input is smaller than that, and the zoom does not undo itself.
 */
export function Field({ label, error, hint, ...input }: FieldProps) {
  const id = useId()
  const errorId = `${id}-error`
  const hintId = `${id}-hint`

  const describedBy = [error ? errorId : null, hint ? hintId : null].filter(Boolean).join(' ')

  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-neutral-800">
        {label}
      </label>

      <input
        {...input}
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy || undefined}
        className={[
          'mt-1.5 block w-full rounded-lg border bg-white px-3.5 py-3 text-base text-neutral-900',
          'placeholder:text-neutral-400 focus:outline-2 focus:outline-offset-1',
          error
            ? 'border-red-400 focus:outline-red-500'
            : 'border-neutral-300 focus:outline-pitch-600',
        ].join(' ')}
      />

      {hint && !error && (
        <p id={hintId} className="mt-1.5 text-xs text-neutral-500">
          {hint}
        </p>
      )}

      {error && (
        <p id={errorId} className="mt-1.5 text-sm text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}

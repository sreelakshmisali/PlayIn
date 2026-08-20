import { useId, type TextareaHTMLAttributes } from 'react'

interface TextAreaProps extends Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'id' | 'className'> {
  label: string
  error?: string | undefined
  hint?: string | undefined
  /** Shows a live character count against this ceiling. */
  maxLength?: number
}

/**
 * A labelled multi-line input, matching Field so the two sit together in a
 * form. Text is 16px for the same reason: iOS Safari zooms a focused control
 * smaller than that and does not zoom back out.
 */
export function TextArea({ label, error, hint, maxLength, value, ...textarea }: TextAreaProps) {
  const id = useId()
  const errorId = `${id}-error`
  const hintId = `${id}-hint`

  const used = typeof value === 'string' ? value.length : 0
  const describedBy = [error ? errorId : null, hint ? hintId : null].filter(Boolean).join(' ')

  return (
    <div>
      <div className="flex items-baseline justify-between gap-2">
        <label htmlFor={id} className="block text-sm font-medium text-neutral-800">
          {label}
        </label>
        {maxLength !== undefined && (
          <span className={`text-xs ${used > maxLength ? 'text-red-700' : 'text-neutral-400'}`}>
            {used}/{maxLength}
          </span>
        )}
      </div>

      <textarea
        {...textarea}
        id={id}
        value={value}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy || undefined}
        className={[
          'mt-1.5 block w-full resize-y rounded-lg border bg-white px-3.5 py-3 text-base text-neutral-900',
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

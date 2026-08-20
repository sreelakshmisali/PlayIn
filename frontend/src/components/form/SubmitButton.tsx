interface SubmitButtonProps {
  pending: boolean
  children: string
  pendingLabel: string
}

/**
 * The primary action of a form.
 *
 * It stays mounted and swaps its label while pending rather than being replaced
 * by a spinner, so the button keeps its position and focus survives the submit.
 */
export function SubmitButton({ pending, children, pendingLabel }: SubmitButtonProps) {
  return (
    <button
      type="submit"
      disabled={pending}
      className={[
        'w-full rounded-lg bg-pitch-600 px-4 py-3 text-base font-medium text-white',
        'transition-colors hover:bg-pitch-700',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-pitch-600',
        'disabled:cursor-not-allowed disabled:bg-neutral-300',
      ].join(' ')}
    >
      {pending ? pendingLabel : children}
    </button>
  )
}

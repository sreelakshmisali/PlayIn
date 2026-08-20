import { useState, type FormEvent } from 'react'

import { TextArea } from '@/components/form/TextArea'

const MAX_REASON_LENGTH = 500

interface ModerationActionFormProps {
  /** What the trigger button reads, and the heading once the form opens. */
  label: string
  pendingLabel: string
  /** Styling hint: reject and suspend both read as a warning, not the default action. */
  tone: 'danger' | 'warning'
  pending: boolean
  onSubmit: (reason: string) => void | Promise<void>
}

const TONE_STYLES: Record<ModerationActionFormProps['tone'], string> = {
  danger: 'border-red-200 bg-white text-red-700 hover:bg-red-50',
  warning: 'border-amber-200 bg-white text-amber-800 hover:bg-amber-50',
}

const TONE_SUBMIT_STYLES: Record<ModerationActionFormProps['tone'], string> = {
  danger: 'bg-red-600 hover:bg-red-700',
  warning: 'bg-amber-600 hover:bg-amber-700',
}

/**
 * A collapsed button that expands into a reason form. Reject and suspend both
 * require a reason (3–500 characters, enforced by the API); this is the one
 * place that shape is captured, rather than duplicating a form per action.
 */
export function ModerationActionForm({
  label,
  pendingLabel,
  tone,
  pending,
  onSubmit,
}: ModerationActionFormProps) {
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState('')

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={`w-full rounded-lg border px-4 py-3 text-base font-medium transition-colors ${TONE_STYLES[tone]}`}
      >
        {label}
      </button>
    )
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    void onSubmit(reason.trim())
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-neutral-200 bg-neutral-50 p-4"
    >
      <TextArea
        label={`Reason for: ${label}`}
        value={reason}
        onChange={(event) => setReason(event.target.value)}
        maxLength={MAX_REASON_LENGTH}
        rows={3}
        placeholder="Explain why, so the owner knows what to fix."
        required
        minLength={3}
      />
      <div className="mt-3 flex gap-2">
        <button
          type="button"
          onClick={() => {
            setOpen(false)
            setReason('')
          }}
          disabled={pending}
          className="flex-1 rounded-lg border border-neutral-300 bg-white px-4 py-2.5 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-100 disabled:cursor-not-allowed disabled:text-neutral-400"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={pending || reason.trim().length < 3}
          className={`flex-1 rounded-lg px-4 py-2.5 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:bg-neutral-300 ${TONE_SUBMIT_STYLES[tone]}`}
        >
          {pending ? pendingLabel : `Confirm ${label.toLowerCase()}`}
        </button>
      </div>
    </form>
  )
}

/**
 * The form-level failure message: the one a field error cannot express, such as
 * wrong credentials or an unreachable API.
 *
 * role="alert" makes a screen reader announce it when it appears, which matters
 * because the message is the only feedback a failed submit produces.
 */
export function FormAlert({ message }: { message: string }) {
  return (
    <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
      <p className="text-sm text-red-800">{message}</p>
    </div>
  )
}

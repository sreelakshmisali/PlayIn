import type { TurfStatus } from '@/lib/owners'

const STYLES: Record<TurfStatus, string> = {
  DRAFT: 'bg-neutral-100 text-neutral-600',
  PENDING_APPROVAL: 'bg-amber-50 text-amber-700',
  APPROVED: 'bg-pitch-50 text-pitch-700',
  REJECTED: 'bg-red-50 text-red-700',
  SUSPENDED: 'bg-red-50 text-red-700',
}

const LABELS: Record<TurfStatus, string> = {
  DRAFT: 'Draft',
  PENDING_APPROVAL: 'Pending review',
  APPROVED: 'Approved',
  REJECTED: 'Rejected',
  SUSPENDED: 'Suspended',
}

/** A small colour-coded label for a turf's place in the approval workflow. */
export function TurfStatusBadge({ status }: { status: TurfStatus }) {
  return (
    <span className={`shrink-0 rounded-full px-3 py-1 text-xs font-medium ${STYLES[status]}`}>
      {LABELS[status]}
    </span>
  )
}

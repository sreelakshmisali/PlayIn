/**
 * Formats a Date as the plain calendar date the API expects (YYYY-MM-DD),
 * using the browser's local date components.
 *
 * `date.toISOString()` is the tempting one-liner here, but it converts to
 * UTC first: for anyone east of Greenwich (India included, the whole point
 * of this product) the early hours of a local day fall on the *previous*
 * UTC day, so the date this app treats as "today" would run behind the
 * calendar on the viewer's own wall clock until well past midnight UTC.
 */
export function toLocalDateString(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

/** Today's date, local. */
export function today(): string {
  return toLocalDateString(new Date())
}

/** A date offset by a number of local days, positive or negative. */
export function addDays(isoDate: string, days: number): string {
  const d = new Date(isoDate + 'T00:00:00')
  d.setDate(d.getDate() + days)
  return toLocalDateString(d)
}

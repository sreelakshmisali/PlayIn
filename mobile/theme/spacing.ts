/** A 4px base scale, used for padding, margin and gap everywhere. */
export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
} as const

export const radius = {
  sm: 8,
  md: 12,
  lg: 16,
  pill: 999,
} as const

/** Minimum touch target edge, matching the platform guidelines on both iOS
 * (44pt) and Android (48dp): 44 is the smaller of the two and satisfies both. */
export const minTouchTarget = 44

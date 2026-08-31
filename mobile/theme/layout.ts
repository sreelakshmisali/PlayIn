import { spacing } from './spacing'

/**
 * App-wide layout constants. Screen-level chrome (padding, header/tab bar
 * heights) belongs here rather than being re-guessed per screen.
 */
export const layout = {
  /** Matches `components/Screen.tsx`'s own content padding — kept as a
   * token so anything measuring against the screen edge (a sticky footer,
   * a full-bleed image) can stay in sync with it. */
  screenPadding: spacing.xl,
  /** Caps line length / control width on a large device (e.g. a tablet);
   * mobile phone widths never hit this. */
  contentMaxWidth: 480,
  headerHeight: 56,
  tabBarHeight: 56,
} as const

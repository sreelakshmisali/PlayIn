import type { TextStyle } from 'react-native'

/**
 * Text styles as plain objects, spread into a component's own StyleSheet
 * rather than exported as styled components: this is a small app, and one
 * extra layer of text-component abstraction is not worth it yet.
 */
export const typography: Record<string, TextStyle> = {
  title: { fontSize: 24, fontWeight: '700', letterSpacing: -0.3 },
  heading: { fontSize: 18, fontWeight: '600' },
  body: { fontSize: 16, fontWeight: '400' },
  bodyMedium: { fontSize: 16, fontWeight: '500' },
  caption: { fontSize: 13, fontWeight: '400' },
  label: { fontSize: 14, fontWeight: '600' },
  button: { fontSize: 16, fontWeight: '600' },
}

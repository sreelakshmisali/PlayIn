import { StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native'

import { spacing as spacingScale, theme } from '../theme'

interface DividerProps {
  /** Vertical margin around the line, from the spacing scale. Defaults to
   * 'lg', matching the gap most screens already use between sections. */
  spacing?: keyof typeof spacingScale
  style?: StyleProp<ViewStyle>
}

/** A hairline horizontal rule for separating sections within a screen. */
export function Divider({ spacing = 'lg', style }: DividerProps) {
  return <View style={[styles.line, { marginVertical: spacingScale[spacing] }, style]} />
}

const styles = StyleSheet.create({
  line: { height: StyleSheet.hairlineWidth, backgroundColor: theme.border, alignSelf: 'stretch' },
})

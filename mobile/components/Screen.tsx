import type { ReactNode } from 'react'
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type ViewStyle } from 'react-native'
import { SafeAreaView } from 'react-native-safe-area-context'

import { layout, theme } from '../theme'

interface ScreenProps {
  children: ReactNode
  /** Off for a screen that manages its own scrolling (e.g. a long list). */
  scroll?: boolean
  /** On for a screen with text inputs, so the keyboard never covers the
   * field being edited. */
  keyboardSafe?: boolean
  /** 'default' (white) is the app's background everywhere. 'muted' is the
   * soft tinted surface, for a screen that wants its content to read as
   * sitting apart from full white — use sparingly (e.g. a booking review
   * step). */
  background?: 'default' | 'muted'
  contentStyle?: ViewStyle
}

/**
 * The one place every screen gets its safe-area insets, background and
 * (optionally) scrolling and keyboard avoidance from. A screen body is
 * written as if it were a plain View; this is what makes it behave on an
 * actual phone — notches, home indicators, and a keyboard that would
 * otherwise sit on top of the field being typed into.
 */
export function Screen({ children, scroll = true, keyboardSafe = false, background = 'default', contentStyle }: ScreenProps) {
  const backgroundColor = background === 'muted' ? theme.surfaceMuted : theme.background

  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[styles.scrollContent, contentStyle]}
      keyboardShouldPersistTaps="handled"
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.content, contentStyle]}>{children}</View>
  )

  return (
    <SafeAreaView style={[styles.safeArea, { backgroundColor }]} edges={['top', 'bottom', 'left', 'right']}>
      {keyboardSafe ? (
        <KeyboardAvoidingView
          style={styles.flex}
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          keyboardVerticalOffset={Platform.OS === 'ios' ? 16 : 0}
        >
          {body}
        </KeyboardAvoidingView>
      ) : (
        body
      )}
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  safeArea: { flex: 1 },
  flex: { flex: 1 },
  content: { flex: 1, padding: layout.screenPadding },
  scrollContent: { flexGrow: 1, padding: layout.screenPadding, paddingBottom: layout.screenPadding * 2 },
})

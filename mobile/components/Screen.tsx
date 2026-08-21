import type { ReactNode } from 'react'
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type ViewStyle } from 'react-native'
import { SafeAreaView } from 'react-native-safe-area-context'

import { colors, spacing } from '../theme'

interface ScreenProps {
  children: ReactNode
  /** Off for a screen that manages its own scrolling (e.g. a long list). */
  scroll?: boolean
  /** On for a screen with text inputs, so the keyboard never covers the
   * field being edited. */
  keyboardSafe?: boolean
  contentStyle?: ViewStyle
}

/**
 * The one place every screen gets its safe-area insets, background and
 * (optionally) scrolling and keyboard avoidance from. A screen body is
 * written as if it were a plain View; this is what makes it behave on an
 * actual phone — notches, home indicators, and a keyboard that would
 * otherwise sit on top of the field being typed into.
 */
export function Screen({ children, scroll = true, keyboardSafe = false, contentStyle }: ScreenProps) {
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
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom', 'left', 'right']}>
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
  safeArea: { flex: 1, backgroundColor: colors.white },
  flex: { flex: 1 },
  content: { flex: 1, padding: spacing.lg },
  scrollContent: { flexGrow: 1, padding: spacing.lg, paddingBottom: spacing.xxl },
})

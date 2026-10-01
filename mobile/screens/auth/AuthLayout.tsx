import type { ReactNode } from 'react'
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  View,
} from 'react-native'
import { SafeAreaView } from 'react-native-safe-area-context'

import { authTheme } from '../../theme'

interface AuthLayoutProps {
  children: ReactNode
}

/**
 * Shared responsive layout container for Authentication screens (Sign In & Sign Up).
 *
 * Key behaviors:
 * 1. Vertically and horizontally centers content on all screen sizes using flexbox.
 * 2. Responsive max-width constraint (400px) so content stays balanced on tablets and large phones.
 * 3. Keyboard avoiding and natural scroll behavior so active inputs are never blocked.
 * 4. Safe-area aware for notch, home indicator, and dynamic island compatibility.
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <SafeAreaView style={styles.safeArea} edges={['top', 'bottom', 'left', 'right']}>
      <KeyboardAvoidingView
        style={styles.keyboardAvoid}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        keyboardVerticalOffset={Platform.OS === 'ios' ? 12 : 0}
      >
        <ScrollView
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
          bounces={false}
        >
          <View style={styles.container}>{children}</View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: authTheme.colors.background,
  },
  keyboardAvoid: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: authTheme.spacing.xl,
    paddingVertical: authTheme.spacing.xl,
  },
  container: {
    width: '100%',
    maxWidth: authTheme.layout.containerMaxWidth,
  },
})

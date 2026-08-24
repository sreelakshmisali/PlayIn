import { StyleSheet } from 'react-native'
import type { BottomTabNavigationOptions } from '@react-navigation/bottom-tabs'
import type { NativeStackNavigationOptions } from '@react-navigation/native-stack'

import { fontSizes, fontWeights, theme, typography } from '../theme'

/**
 * The app's global navigation chrome — shared by every stack/tab navigator
 * (`AuthNavigator`, `PlayerNavigator`, `OwnerNavigator`) so a pushed screen's
 * header and the bottom tab bar look identical regardless of which flow
 * they're in. This is the visual shell only: which screens exist, which
 * tab a role sees, and what each route is named are still each
 * navigator's own concern.
 */

/** Applied as every stack navigator's `screenOptions`. A flat white header
 * with no shadow/border, a centered title in the app's own type scale, and
 * a chevron-only back button tinted with the brand accent — the one place
 * green appears in the shell, and only as an icon tint, never a filled bar.
 *
 * `animation` is left at native-stack's platform default (a native
 * push/pop on iOS, a native fade-through on Android) rather than a custom
 * curve: it's already GPU-driven, respects the OS's own Reduce Motion
 * setting automatically, and is faster/smoother than anything hand-rolled
 * here would be — exactly what "fast, smooth, subtle" asks for. */
export const stackScreenOptions: NativeStackNavigationOptions = {
  headerStyle: { backgroundColor: theme.background },
  headerShadowVisible: false,
  headerTitleAlign: 'center',
  headerTitleStyle: { ...typography.sectionTitle, color: theme.textPrimary },
  headerTintColor: theme.primary,
  headerBackButtonDisplayMode: 'minimal',
  contentStyle: { backgroundColor: theme.background },
}

/** Applied as every bottom tab navigator's `screenOptions`. A white bar with
 * a hairline top border instead of the platform's default drop shadow —
 * lightweight rather than a heavy, floating, or colored bar. Each
 * navigator still supplies its own `tabBarIcon` (the icons differ per
 * role). */
export const tabScreenOptions: BottomTabNavigationOptions = {
  headerShown: false,
  tabBarActiveTintColor: theme.primary,
  tabBarInactiveTintColor: theme.textMuted,
  tabBarStyle: {
    backgroundColor: theme.background,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.border,
    elevation: 0,
    shadowOpacity: 0,
  },
  tabBarLabelStyle: { fontSize: fontSizes.sm, fontWeight: fontWeights.medium },
}

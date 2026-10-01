import { NavigationContainer } from '@react-navigation/native'
import { StatusBar } from 'expo-status-bar'
import { SafeAreaProvider } from 'react-native-safe-area-context'
import * as SplashScreenModule from 'expo-splash-screen'

import { AuthProvider } from '../hooks'
import { RootNavigator } from '../navigation'

// Prevent native splash screen from hiding before auth state resolves
SplashScreenModule.preventAutoHideAsync().catch(() => {
  /* ignore unhandled rejections on web or dev reloads */
})

/**
 * The app's composition root: every provider the rest of the tree needs,
 * wrapping the one thing that decides what's actually on screen
 * (RootNavigator). Kept separate from the project-root App.tsx so that file
 * can stay the thin, Expo-required entry point.
 */
export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <NavigationContainer>
          <RootNavigator />
          <StatusBar style="dark" />
        </NavigationContainer>
      </AuthProvider>
    </SafeAreaProvider>
  )
}

import { createNativeStackNavigator } from '@react-navigation/native-stack'

import { LoginScreen, RegisterScreen } from '../screens'
import { theme } from '../theme'
import type { AuthStackParamList } from './types'

const Stack = createNativeStackNavigator<AuthStackParamList>()

/** Signed-out flow: sign in or create an account. */
export function AuthNavigator() {
  return (
    <Stack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: theme.background } }}>
      <Stack.Screen name="Login" component={LoginScreen} />
      <Stack.Screen name="Register" component={RegisterScreen} />
    </Stack.Navigator>
  )
}

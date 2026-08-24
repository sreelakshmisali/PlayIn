import { createNativeStackNavigator } from '@react-navigation/native-stack'

import { LoginScreen, RegisterScreen } from '../screens'
import { stackScreenOptions } from './chrome'
import type { AuthStackParamList } from './types'

const Stack = createNativeStackNavigator<AuthStackParamList>()

/** Signed-out flow: sign in or create an account. */
export function AuthNavigator() {
  return (
    <Stack.Navigator screenOptions={{ ...stackScreenOptions, headerShown: false }}>
      <Stack.Screen name="Login" component={LoginScreen} />
      <Stack.Screen name="Register" component={RegisterScreen} />
    </Stack.Navigator>
  )
}

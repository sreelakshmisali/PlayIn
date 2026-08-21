import { Ionicons } from '@expo/vector-icons'
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs'
import { createNativeStackNavigator } from '@react-navigation/native-stack'

import {
  AccountScreen,
  OwnerProfileEditScreen,
  OwnerProfileScreen,
  OwnerTurfEditScreen,
  OwnerTurfListScreen,
  TurfDetailScreen,
  TurfListScreen,
} from '../screens'
import { colors, theme } from '../theme'
import type { OwnerStackParamList, OwnerTabParamList } from './types'

const Tab = createBottomTabNavigator<OwnerTabParamList>()
const Stack = createNativeStackNavigator<OwnerStackParamList>()

const TAB_ICONS: Record<keyof OwnerTabParamList, keyof typeof Ionicons.glyphMap> = {
  MyTurfs: 'list-outline',
  Turfs: 'location-outline',
  Profile: 'person-outline',
  Account: 'settings-outline',
}

function OwnerTabs() {
  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: colors.neutral400,
        tabBarIcon: ({ color, size }) => <Ionicons name={TAB_ICONS[route.name]} color={color} size={size} />,
      })}
    >
      <Tab.Screen name="MyTurfs" component={OwnerTurfListScreen} options={{ title: 'My Turfs' }} />
      <Tab.Screen name="Turfs" component={TurfListScreen} options={{ title: 'Browse' }} />
      <Tab.Screen name="Profile" component={OwnerProfileScreen} options={{ title: 'Profile' }} />
      <Tab.Screen name="Account" component={AccountScreen} />
    </Tab.Navigator>
  )
}

/** Signed-in OWNER flow: the tab bar plus the screens pushed above it. */
export function OwnerNavigator() {
  return (
    <Stack.Navigator screenOptions={{ contentStyle: { backgroundColor: theme.background } }}>
      <Stack.Screen name="OwnerTabs" component={OwnerTabs} options={{ headerShown: false }} />
      <Stack.Screen name="TurfDetail" component={TurfDetailScreen} options={{ title: 'Turf' }} />
      <Stack.Screen name="OwnerProfileEdit" component={OwnerProfileEditScreen} options={{ title: 'Edit profile' }} />
      <Stack.Screen
        name="OwnerTurfEdit"
        component={OwnerTurfEditScreen}
        options={({ route }) => ({ title: route.params.turfId ? 'Edit turf' : 'New turf' })}
      />
    </Stack.Navigator>
  )
}

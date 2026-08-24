import { Ionicons } from '@expo/vector-icons'
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs'
import { createNativeStackNavigator } from '@react-navigation/native-stack'

import {
  AccountScreen,
  HomeScreen,
  MyBookingsScreen,
  PlayerProfileEditScreen,
  PlayerProfileScreen,
  TurfDetailScreen,
  TurfListScreen,
} from '../screens'
import { stackScreenOptions, tabScreenOptions } from './chrome'
import type { PlayerStackParamList, PlayerTabParamList } from './types'

const Tab = createBottomTabNavigator<PlayerTabParamList>()
const Stack = createNativeStackNavigator<PlayerStackParamList>()

const TAB_ICONS: Record<keyof PlayerTabParamList, keyof typeof Ionicons.glyphMap> = {
  Home: 'home-outline',
  Turfs: 'location-outline',
  Bookings: 'calendar-outline',
  Profile: 'person-outline',
  Account: 'settings-outline',
}

function PlayerTabs() {
  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        ...tabScreenOptions,
        tabBarIcon: ({ color, size }) => <Ionicons name={TAB_ICONS[route.name]} color={color} size={size} />,
      })}
    >
      <Tab.Screen name="Home" component={HomeScreen} />
      <Tab.Screen name="Turfs" component={TurfListScreen} />
      <Tab.Screen name="Bookings" component={MyBookingsScreen} options={{ title: 'My Bookings' }} />
      <Tab.Screen name="Profile" component={PlayerProfileScreen} options={{ title: 'Profile' }} />
      <Tab.Screen name="Account" component={AccountScreen} />
    </Tab.Navigator>
  )
}

/** Signed-in PLAYER flow: the tab bar plus the screens pushed above it. */
export function PlayerNavigator() {
  return (
    <Stack.Navigator screenOptions={stackScreenOptions}>
      <Stack.Screen name="PlayerTabs" component={PlayerTabs} options={{ headerShown: false }} />
      <Stack.Screen name="TurfDetail" component={TurfDetailScreen} options={{ title: 'Turf' }} />
      <Stack.Screen name="PlayerProfileEdit" component={PlayerProfileEditScreen} options={{ title: 'Edit profile' }} />
    </Stack.Navigator>
  )
}

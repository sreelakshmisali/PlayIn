import { Ionicons } from '@expo/vector-icons'
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs'
import { createNativeStackNavigator } from '@react-navigation/native-stack'

import {
  AccountScreen,
  HomeScreen,
  PlayerProfileEditScreen,
  PlayerProfileScreen,
  TurfDetailScreen,
  TurfListScreen,
} from '../screens'
import { colors, theme } from '../theme'
import type { PlayerStackParamList, PlayerTabParamList } from './types'

const Tab = createBottomTabNavigator<PlayerTabParamList>()
const Stack = createNativeStackNavigator<PlayerStackParamList>()

const TAB_ICONS: Record<keyof PlayerTabParamList, keyof typeof Ionicons.glyphMap> = {
  Home: 'home-outline',
  Turfs: 'location-outline',
  Profile: 'person-outline',
  Account: 'settings-outline',
}

function PlayerTabs() {
  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: colors.neutral400,
        tabBarIcon: ({ color, size }) => <Ionicons name={TAB_ICONS[route.name]} color={color} size={size} />,
      })}
    >
      <Tab.Screen name="Home" component={HomeScreen} />
      <Tab.Screen name="Turfs" component={TurfListScreen} />
      <Tab.Screen name="Profile" component={PlayerProfileScreen} options={{ title: 'Profile' }} />
      <Tab.Screen name="Account" component={AccountScreen} />
    </Tab.Navigator>
  )
}

/** Signed-in PLAYER flow: the tab bar plus the screens pushed above it. */
export function PlayerNavigator() {
  return (
    <Stack.Navigator screenOptions={{ contentStyle: { backgroundColor: theme.background } }}>
      <Stack.Screen name="PlayerTabs" component={PlayerTabs} options={{ headerShown: false }} />
      <Stack.Screen name="TurfDetail" component={TurfDetailScreen} options={{ title: 'Turf' }} />
      <Stack.Screen name="PlayerProfileEdit" component={PlayerProfileEditScreen} options={{ title: 'Edit profile' }} />
    </Stack.Navigator>
  )
}

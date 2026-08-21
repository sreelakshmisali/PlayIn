import { useAuth } from '../hooks'
import { LoadingView } from '../components'
import { NotAvailableScreen } from '../screens'
import { AuthNavigator } from './AuthNavigator'
import { PlayerNavigator } from './PlayerNavigator'
import { OwnerNavigator } from './OwnerNavigator'

/**
 * Picks the whole navigator tree from the signed-in user's role, per the
 * mobile foundation's brief: authentication flow, player flow, owner flow.
 * ADMIN gets an honest "not on mobile yet" rather than being dropped into
 * either flow — the admin web app remains the tool for that role.
 */
export function RootNavigator() {
  const { status, user } = useAuth()

  if (status === 'loading') {
    return <LoadingView message="Loading PlayHub" />
  }

  if (status === 'anonymous' || !user) {
    return <AuthNavigator />
  }

  switch (user.role) {
    case 'PLAYER':
      return <PlayerNavigator />
    case 'OWNER':
      return <OwnerNavigator />
    default:
      return <NotAvailableScreen />
  }
}

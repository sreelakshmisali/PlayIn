import { Ionicons } from '@expo/vector-icons'
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { useCallback, useEffect, useState } from 'react'
import { Pressable, StyleSheet, View } from 'react-native'

import { Button, Divider, EmptyState, ErrorBanner, IconContainer, LoadingView, Screen, Text } from '../../components'
import { useAuth } from '../../hooks'
import { fetchMyPlayerProfile } from '../../services/players'
import { ApiError } from '../../services/api'
import { cardPresets, iconSizes, spacing, theme } from '../../theme'
import type { PlayerProfile } from '../../types/players'

type Props = BottomTabScreenProps<Record<string, object | undefined>>

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; profile: PlayerProfile }
  | { kind: 'no-profile' }
  | { kind: 'failed'; message: string }

// ---------------------------------------------------------------------------
// Reusable rows
// ---------------------------------------------------------------------------

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.infoRow}>
      <Text variant="caption" color="secondary">{label}</Text>
      <Text variant="bodyEmphasized" color="primary" numberOfLines={1} style={styles.infoValue}>{value}</Text>
    </View>
  )
}

function MenuRow({ icon, label, onPress }: { icon: keyof typeof Ionicons.glyphMap; label: string; onPress: () => void }) {
  return (
    <Pressable style={styles.menuRow} onPress={onPress} accessibilityRole="button">
      <Ionicons name={icon} size={iconSizes.md} color={theme.textSecondary} />
      <Text variant="body" color="primary" style={styles.menuLabel}>{label}</Text>
      <Ionicons name="chevron-forward" size={iconSizes.sm} color={theme.textMuted} />
    </Pressable>
  )
}

// ---------------------------------------------------------------------------
// Avatar
// ---------------------------------------------------------------------------

function Avatar({ name, imageUrl }: { name: string; imageUrl?: string }) {
  // Use initials when no image is available
  const initials = name
    .split(' ')
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')

  if (imageUrl) {
    // Image avatar would go here when image upload is supported
  }

  return (
    <View style={styles.avatar}>
      <Text variant="screenTitle" color="onPrimary">{initials}</Text>
    </View>
  )
}

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

/** The signed-in player's profile: identity, account details, actions, and
 * sign-out, all on one screen. Merges what was previously split between
 * Profile and Account tabs into a single, minimal view. */
export function PlayerProfileScreen({ navigation }: Props) {
  const { user, logout } = useAuth()
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [signingOut, setSigningOut] = useState(false)

  const load = useCallback(() => {
    setState({ kind: 'loading' })
    fetchMyPlayerProfile()
      .then((profile) => setState({ kind: 'ready', profile }))
      .catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 404) {
          setState({ kind: 'no-profile' })
          return
        }
        setState({ kind: 'failed', message: error instanceof ApiError ? error.message : 'Could not load your profile.' })
      })
  }, [])

  useEffect(() => {
    const unsubscribe = navigation.addListener('focus', load)
    return unsubscribe
  }, [navigation, load])

  if (state.kind === 'loading') {
    return <LoadingView message="Loading your profile" />
  }

  if (state.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner message={state.message} onRetry={load} />
      </Screen>
    )
  }

  if (state.kind === 'no-profile') {
    return (
      <Screen>
        {/* Even without a player profile, show account identity */}
        {user && (
          <View style={styles.identityBlock}>
            <Avatar name={user.full_name} />
            <Text variant="screenTitle" color="primary" style={styles.identityName}>
              {user.full_name}
            </Text>
            <Text variant="caption" color="secondary">{user.email}</Text>
          </View>
        )}

        <View style={styles.emptyBlock}>
          <EmptyState
            title="No profile yet"
            message="You haven't set up your player profile yet."
            icon={
              <IconContainer tone="muted" size="lg">
                <Ionicons name="person-outline" size={iconSizes.lg} color={theme.textMuted} />
              </IconContainer>
            }
            actionLabel="Create your profile"
            onAction={() => navigation.navigate('PlayerProfileEdit')}
          />
        </View>

        <View style={styles.signOutBlock}>
          <Button
            label={signingOut ? 'Signing out' : 'Sign out'}
            variant="secondary"
            pending={signingOut}
            onPress={() => {
              setSigningOut(true)
              void logout()
            }}
          />
        </View>
      </Screen>
    )
  }

  const { profile } = state

  return (
    <Screen>
      {/* ── Identity ──────────────────────────────────────────── */}
      <View style={styles.identityBlock}>
        <Avatar name={profile.display_name} imageUrl={profile.image_url} />
        <Text variant="screenTitle" color="primary" style={styles.identityName}>
          {profile.display_name}
        </Text>
        {profile.location ? (
          <View style={styles.locationRow}>
            <Ionicons name="location-outline" size={14} color={theme.textMuted} />
            <Text variant="caption" color="secondary">{profile.location}</Text>
          </View>
        ) : null}
        {profile.bio ? (
          <Text variant="body" color="secondary" style={styles.bio}>
            {profile.bio}
          </Text>
        ) : null}
      </View>

      {/* ── Sports ────────────────────────────────────────────── */}
      {profile.sports.length > 0 && (
        <View style={styles.section}>
          <Text variant="label" color="secondary" style={styles.sectionLabel}>Sports</Text>
          <View style={styles.pillRow}>
            {profile.sports.map((ps) => (
              <View key={ps.sport.id} style={styles.pill}>
                <Text variant="caption" color="primary">
                  {ps.sport.name}
                  {ps.position ? ` · ${ps.position}` : ''}
                </Text>
              </View>
            ))}
          </View>
        </View>
      )}

      <Divider spacing="lg" />

      {/* ── Account info ──────────────────────────────────────── */}
      <View style={styles.section}>
        <Text variant="label" color="secondary" style={styles.sectionLabel}>Account</Text>
        {user && (
          <>
            <InfoRow label="Email" value={user.email} />
            <InfoRow label="Role" value={user.role === 'PLAYER' ? 'Player' : user.role === 'OWNER' ? 'Owner' : user.role} />
            <InfoRow label="Status" value={user.is_active ? 'Active' : 'Deactivated'} />
            <InfoRow label="Member since" value={new Date(user.created_at).toLocaleDateString('en-US', { month: 'short', year: 'numeric' })} />
          </>
        )}
      </View>

      <Divider spacing="lg" />

      {/* ── Actions ───────────────────────────────────────────── */}
      <View style={styles.section}>
        <MenuRow icon="create-outline" label="Edit profile" onPress={() => navigation.navigate('PlayerProfileEdit')} />
      </View>

      {/* ── Sign out ──────────────────────────────────────────── */}
      <View style={styles.signOutBlock}>
        <Button
          label={signingOut ? 'Signing out' : 'Sign out'}
          variant="secondary"
          pending={signingOut}
          onPress={() => {
            setSigningOut(true)
            void logout()
          }}
        />
      </View>
    </Screen>
  )
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const AVATAR_SIZE = 72

const styles = StyleSheet.create({
  // Identity
  identityBlock: {
    alignItems: 'center',
    paddingTop: spacing.lg,
    paddingBottom: spacing.sm,
  },
  avatar: {
    width: AVATAR_SIZE,
    height: AVATAR_SIZE,
    borderRadius: AVATAR_SIZE / 2,
    backgroundColor: theme.primary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  identityName: {
    marginTop: spacing.md,
    textAlign: 'center',
  },
  locationRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    marginTop: spacing.xs,
  },
  bio: {
    marginTop: spacing.sm,
    textAlign: 'center',
    maxWidth: 280,
  },

  // Sections
  section: {
    marginTop: spacing.xs,
  },
  sectionLabel: {
    marginBottom: spacing.sm,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },

  // Sports pills
  pillRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.sm,
  },
  pill: {
    ...cardPresets.pill,
    backgroundColor: theme.surfaceMuted,
  },

  // Info rows
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: spacing.md,
    paddingVertical: spacing.sm,
  },
  // A long email address gets an ellipsis instead of overflowing past
  // the card edge or squeezing the "Email"/"Role"/etc. label.
  infoValue: {
    flexShrink: 1,
    textAlign: 'right',
  },

  // Menu rows
  menuRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: spacing.md,
    gap: spacing.md,
  },
  menuLabel: {
    flex: 1,
  },

  // Blocks
  emptyBlock: {
    marginTop: spacing.xl,
  },
  signOutBlock: {
    marginTop: spacing.xxl,
    paddingBottom: spacing.lg,
  },
})

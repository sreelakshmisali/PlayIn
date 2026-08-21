/**
 * Token storage.
 *
 * Tokens live in the platform keychain/keystore via expo-secure-store, never
 * in AsyncStorage or any other plain, unencrypted store: they are bearer
 * credentials, and this is the mobile equivalent of not putting them in
 * localStorage. Everything that touches storage goes through this module.
 */
import * as SecureStore from 'expo-secure-store'

const ACCESS_KEY = 'playhub.access_token'
const REFRESH_KEY = 'playhub.refresh_token'

export interface StoredTokens {
  accessToken: string
  refreshToken: string
}

/** Reads the stored pair, or null when the user is signed out. */
export async function readTokens(): Promise<StoredTokens | null> {
  const [accessToken, refreshToken] = await Promise.all([
    SecureStore.getItemAsync(ACCESS_KEY),
    SecureStore.getItemAsync(REFRESH_KEY),
  ])

  if (!accessToken || !refreshToken) return null
  return { accessToken, refreshToken }
}

/** Replaces the stored pair. */
export async function writeTokens(tokens: StoredTokens): Promise<void> {
  await Promise.all([
    SecureStore.setItemAsync(ACCESS_KEY, tokens.accessToken),
    SecureStore.setItemAsync(REFRESH_KEY, tokens.refreshToken),
  ])
}

/** Removes the stored pair. */
export async function clearTokens(): Promise<void> {
  await Promise.all([SecureStore.deleteItemAsync(ACCESS_KEY), SecureStore.deleteItemAsync(REFRESH_KEY)])
}

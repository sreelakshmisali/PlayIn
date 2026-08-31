/**
 * The API's origin, e.g. "http://192.168.1.20:8080". Configurable, never
 * hardcoded: `localhost` inside a physical phone or a real Android emulator
 * refers to the device itself, not the development machine, so a plain
 * "http://localhost:8080" only ever works in the iOS Simulator and Expo web.
 *
 * Set with EXPO_PUBLIC_API_URL, read by Expo's bundler at build/start time
 * (see .env.example and the README for the exact values to use per target).
 * The fallback below is the Android emulator's own alias for the host
 * machine's localhost, so `npm run android` works out of the box without a
 * .env file; every other target needs EXPO_PUBLIC_API_URL set explicitly.
 */
const DEFAULT_ANDROID_EMULATOR_API_URL = 'http://172.17.0.235:8080'

export const apiBaseUrl: string = process.env.EXPO_PUBLIC_API_URL ?? DEFAULT_ANDROID_EMULATOR_API_URL

/** Version prefix. A breaking API change gets a new client, not a new branch. */
export const API_PREFIX = '/api/v1'

/** How long a request waits before it is treated as a network failure. */
export const apiTimeoutMs = 15_000

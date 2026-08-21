# PlayHub Mobile

React Native + Expo + TypeScript. The primary PlayHub client for players and
owners, talking to the existing Go API under `/api/v1`. See the root
[README.md](../README.md#mobile-app-react-native--expo) for the full picture,
including the mobile/API/database relationship and known limitations of this
foundation phase. This file is the quick reference for working inside `mobile/`.

## Install and run

```bash
npm install
cp .env.example .env   # then edit EXPO_PUBLIC_API_URL — see below
npm start
```

Press `a` for Android, `i` for iOS (macOS only), or scan the QR code with
Expo Go on a physical device.

## Configuring the API URL

Set `EXPO_PUBLIC_API_URL` in `.env`. **`localhost` does not mean your
development machine** once code runs inside an emulator or on a phone:

| Target | Value |
|---|---|
| Android emulator | `http://10.0.2.2:8080` (already the default, no `.env` needed) |
| iOS Simulator | `http://localhost:8080` |
| Physical device | `http://<your-LAN-IP>:8080` — find it with `ipconfig` (Windows) or `ifconfig`/`ip addr` (macOS/Linux) |

## Verify

```bash
npm run typecheck
npx expo-doctor
npx expo export --platform android   # produces a real bundle, not just a type-check
```

## Structure

```
app/          root composition: providers + RootNavigator
components/   shared UI primitives (Button, TextField, Screen, ...)
screens/      auth/, public/, player/, owner/ — one screen per route
navigation/   AuthNavigator, PlayerNavigator, OwnerNavigator, RootNavigator
services/     api.ts (the one HTTP client) + one file per domain
hooks/        AuthProvider / useAuth
types/        shared request/response shapes
storage/      expo-secure-store-backed token storage
theme/        colours, spacing, typography
```

Screens never call `fetch` directly — always through `services/`.

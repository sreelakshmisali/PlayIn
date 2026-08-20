import { createBrowserRouter } from 'react-router-dom'

import { ProtectedRoute } from '@/auth/ProtectedRoute'
import { AppLayout } from '@/components/layout/AppLayout'
import { AccountPage } from '@/pages/AccountPage'
import { AdminDashboardPage } from '@/pages/AdminDashboardPage'
import { AdminPendingTurfsPage } from '@/pages/AdminPendingTurfsPage'
import { AdminTurfReviewPage } from '@/pages/AdminTurfReviewPage'
import { AdminUserDetailPage } from '@/pages/AdminUserDetailPage'
import { AdminUsersPage } from '@/pages/AdminUsersPage'
import { HomePage } from '@/pages/HomePage'
import { LoginPage } from '@/pages/LoginPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { OwnerProfileEditPage } from '@/pages/OwnerProfileEditPage'
import { OwnerProfilePage } from '@/pages/OwnerProfilePage'
import { OwnerTurfCreatePage } from '@/pages/OwnerTurfCreatePage'
import { OwnerTurfEditPage } from '@/pages/OwnerTurfEditPage'
import { OwnerTurfListPage } from '@/pages/OwnerTurfListPage'
import { PlayerProfileEditPage } from '@/pages/PlayerProfileEditPage'
import { PlayerProfilePage } from '@/pages/PlayerProfilePage'
import { PublicPlayerPage } from '@/pages/PublicPlayerPage'
import { PublicTurfDetailPage } from '@/pages/PublicTurfDetailPage'
import { PublicTurfListPage } from '@/pages/PublicTurfListPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { StatusPage } from '@/pages/StatusPage'

/**
 * The route table. Every route is a child of AppLayout, so the shell renders
 * once and pages swap inside it.
 *
 * Routes that need a session are grouped under ProtectedRoute rather than each
 * page checking for itself, so a page cannot be added to a protected group and
 * forget its guard.
 */
export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: 'status', element: <StatusPage /> },
      { path: 'login', element: <LoginPage /> },
      { path: 'register', element: <RegisterPage /> },

      // Public: a player profile and a turf listing are both meant to be
      // found without a session.
      { path: 'players/:userId', element: <PublicPlayerPage /> },
      { path: 'turfs', element: <PublicTurfListPage /> },
      { path: 'turfs/:turfId', element: <PublicTurfDetailPage /> },

      {
        element: <ProtectedRoute />,
        children: [{ path: 'account', element: <AccountPage /> }],
      },

      // Managing a player profile is PLAYER-only, matching the API. An OWNER
      // or ADMIN reaching these paths gets the same refusal the server gives.
      {
        element: <ProtectedRoute roles={['PLAYER']} />,
        children: [
          { path: 'profile', element: <PlayerProfilePage /> },
          { path: 'profile/edit', element: <PlayerProfileEditPage /> },
        ],
      },

      // Managing an owner profile and its turfs is OWNER-only, matching the
      // API. A PLAYER or ADMIN reaching these paths gets the same refusal the
      // server gives.
      {
        element: <ProtectedRoute roles={['OWNER']} />,
        children: [
          { path: 'owner/profile', element: <OwnerProfilePage /> },
          { path: 'owner/profile/edit', element: <OwnerProfileEditPage /> },
          { path: 'owner/turfs', element: <OwnerTurfListPage /> },
          { path: 'owner/turfs/new', element: <OwnerTurfCreatePage /> },
          { path: 'owner/turfs/:turfId/edit', element: <OwnerTurfEditPage /> },
        ],
      },

      // Turf moderation and user management are ADMIN-only, matching the API.
      // A PLAYER or OWNER reaching these paths gets the same refusal the
      // server gives.
      {
        element: <ProtectedRoute roles={['ADMIN']} />,
        children: [
          { path: 'admin', element: <AdminDashboardPage /> },
          { path: 'admin/turfs', element: <AdminPendingTurfsPage /> },
          { path: 'admin/turfs/:turfId', element: <AdminTurfReviewPage /> },
          { path: 'admin/users', element: <AdminUsersPage /> },
          { path: 'admin/users/:userId', element: <AdminUserDetailPage /> },
        ],
      },

      { path: '*', element: <NotFoundPage /> },
    ],
  },
])

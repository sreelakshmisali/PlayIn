import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'

import { AuthProvider } from '@/auth/AuthProvider'
import { router } from '@/routes/router'

import './index.css'

const container = document.getElementById('root')

if (!container) {
  throw new Error('Root element #root is missing from index.html')
}

createRoot(container).render(
  <StrictMode>
    {/* AuthProvider wraps the router so every route, including the guards in
        the route table, reads one session. */}
    <AuthProvider>
      <RouterProvider router={router} />
    </AuthProvider>
  </StrictMode>,
)

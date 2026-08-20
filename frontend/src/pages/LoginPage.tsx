import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import { Field } from '@/components/form/Field'
import { FormAlert } from '@/components/form/FormAlert'
import { SubmitButton } from '@/components/form/SubmitButton'
import { AuthCard } from '@/components/layout/AuthCard'
import { ApiError } from '@/lib/api'

/** Where ProtectedRoute stashes the path the user was trying to reach. */
interface LocationState {
  from?: string
}

export function LoginPage() {
  const { status, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [message, setMessage] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const destination = (location.state as LocationState | null)?.from ?? '/account'

  // Someone already signed in has no business on the login screen.
  if (status === 'authenticated') {
    return <Navigate to={destination} replace />
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setPending(true)
    setMessage('')
    setFieldErrors({})

    try {
      await login({ email, password })
      navigate(destination, { replace: true })
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fieldErrors())
        setMessage(error.message)
      } else {
        setMessage('Something went wrong. Try again.')
      }
      setPending(false)
    }
  }

  return (
    <AuthCard
      title="Sign in to PlayHub"
      subtitle="Book turfs, build a team, play more."
      footer={
        <>
          New to PlayHub?{' '}
          <Link to="/register" className="font-medium text-pitch-700 underline">
            Create an account
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} noValidate className="space-y-5">
        {message && <FormAlert message={message} />}

        <Field
          label="Email"
          type="email"
          name="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          autoComplete="email"
          inputMode="email"
          autoCapitalize="none"
          spellCheck={false}
          required
          placeholder="you@example.com"
          error={fieldErrors.email}
        />

        <Field
          label="Password"
          type="password"
          name="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete="current-password"
          required
          placeholder="Your password"
          error={fieldErrors.password}
        />

        <SubmitButton pending={pending} pendingLabel="Signing in">
          Sign in
        </SubmitButton>
      </form>
    </AuthCard>
  )
}

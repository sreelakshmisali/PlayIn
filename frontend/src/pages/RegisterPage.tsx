import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import { Field } from '@/components/form/Field'
import { FormAlert } from '@/components/form/FormAlert'
import { SubmitButton } from '@/components/form/SubmitButton'
import { AuthCard } from '@/components/layout/AuthCard'
import { ApiError } from '@/lib/api'
import type { Role } from '@/lib/auth'

// ADMIN is missing on purpose: the API refuses to self-assign it, so offering
// it here would only produce a rejected submit.
const ROLE_CHOICES: { value: Role; label: string; description: string }[] = [
  { value: 'PLAYER', label: 'Player', description: 'Book turfs and join teams' },
  { value: 'OWNER', label: 'Turf owner', description: 'List and manage turfs' },
]

export function RegisterPage() {
  const { status, register } = useAuth()
  const navigate = useNavigate()

  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('PLAYER')
  const [pending, setPending] = useState(false)
  const [message, setMessage] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  if (status === 'authenticated') {
    return <Navigate to="/account" replace />
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setPending(true)
    setMessage('')
    setFieldErrors({})

    try {
      await register({ email, password, full_name: fullName, role })
      navigate('/account', { replace: true })
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
      title="Create your account"
      subtitle="One account for booking, teams and tournaments."
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="font-medium text-pitch-700 underline">
            Sign in
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} noValidate className="space-y-5">
        {message && <FormAlert message={message} />}

        <Field
          label="Full name"
          type="text"
          name="full_name"
          value={fullName}
          onChange={(event) => setFullName(event.target.value)}
          autoComplete="name"
          required
          placeholder="Your name"
          error={fieldErrors.full_name}
        />

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
          autoComplete="new-password"
          required
          placeholder="At least 10 characters"
          hint="At least 10 characters, with a letter and a number."
          error={fieldErrors.password}
        />

        <fieldset>
          <legend className="text-sm font-medium text-neutral-800">I am joining as</legend>

          <div className="mt-2 space-y-2">
            {ROLE_CHOICES.map((choice) => (
              <label
                key={choice.value}
                className={[
                  'flex cursor-pointer items-start gap-3 rounded-lg border px-3.5 py-3 transition-colors',
                  role === choice.value
                    ? 'border-pitch-600 bg-pitch-50'
                    : 'border-neutral-300 bg-white hover:bg-neutral-50',
                ].join(' ')}
              >
                <input
                  type="radio"
                  name="role"
                  value={choice.value}
                  checked={role === choice.value}
                  onChange={() => setRole(choice.value)}
                  className="mt-1 size-4 accent-pitch-600"
                />
                <span>
                  <span className="block text-sm font-medium text-neutral-900">{choice.label}</span>
                  <span className="block text-xs text-neutral-500">{choice.description}</span>
                </span>
              </label>
            ))}
          </div>

          {fieldErrors.role && <p className="mt-1.5 text-sm text-red-700">{fieldErrors.role}</p>}
        </fieldset>

        <SubmitButton pending={pending} pendingLabel="Creating account">
          Create account
        </SubmitButton>
      </form>
    </AuthCard>
  )
}

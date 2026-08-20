import { useState, type ChangeEvent, type FormEvent } from 'react'

import { Field } from '@/components/form/Field'
import { SubmitButton } from '@/components/form/SubmitButton'
import { TextArea } from '@/components/form/TextArea'
import type { SaveTurfPayload, Turf } from '@/lib/owners'

const DESCRIPTION_LIMIT = 2000

/** The values TurfDetailsForm edits, as strings so every input stays controlled. */
export interface TurfFormValues {
  name: string
  description: string
  address: string
  city: string
  latitude: string
  longitude: string
  capacity: string
  openingTime: string
  closingTime: string
}

export const emptyTurfFormValues: TurfFormValues = {
  name: '',
  description: '',
  address: '',
  city: '',
  latitude: '',
  longitude: '',
  capacity: '',
  openingTime: '06:00',
  closingTime: '22:00',
}

/** Maps a stored turf onto editable form values. */
export function turfToFormValues(turf: Turf): TurfFormValues {
  return {
    name: turf.name,
    description: turf.description ?? '',
    address: turf.address,
    city: turf.city,
    latitude: turf.latitude !== undefined ? String(turf.latitude) : '',
    longitude: turf.longitude !== undefined ? String(turf.longitude) : '',
    capacity: turf.capacity !== undefined ? String(turf.capacity) : '',
    openingTime: turf.opening_time,
    closingTime: turf.closing_time,
  }
}

function toPayload(values: TurfFormValues): SaveTurfPayload {
  return {
    name: values.name,
    description: values.description,
    address: values.address,
    city: values.city,
    latitude: values.latitude.trim() === '' ? null : Number(values.latitude),
    longitude: values.longitude.trim() === '' ? null : Number(values.longitude),
    capacity: values.capacity.trim() === '' ? null : Number(values.capacity),
    opening_time: values.openingTime,
    closing_time: values.closingTime,
  }
}

interface TurfDetailsFormProps {
  initial: TurfFormValues
  pending: boolean
  submitLabel: string
  pendingLabel: string
  fieldErrors: Record<string, string>
  onSubmit: (payload: SaveTurfPayload) => void
}

/**
 * The turf details form, shared between create and edit: both screens edit
 * the exact same fields, so one component serves both rather than two copies
 * drifting apart.
 */
export function TurfDetailsForm({
  initial,
  pending,
  submitLabel,
  pendingLabel,
  fieldErrors,
  onSubmit,
}: TurfDetailsFormProps) {
  const [values, setValues] = useState(initial)

  function set<K extends keyof TurfFormValues>(key: K) {
    return (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setValues((prev) => ({ ...prev, [key]: event.target.value }))
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSubmit(toPayload(values))
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-5">
      <Field
        label="Turf name"
        type="text"
        value={values.name}
        onChange={set('name')}
        required
        placeholder="Riverside Turf"
        error={fieldErrors.name}
      />

      <TextArea
        label="Description"
        rows={3}
        value={values.description}
        onChange={set('description')}
        maxLength={DESCRIPTION_LIMIT}
        placeholder="What makes this turf worth booking."
        hint="Optional."
        error={fieldErrors.description}
      />

      <Field
        label="Address"
        type="text"
        value={values.address}
        onChange={set('address')}
        required
        placeholder="123 River Road, Panampilly Nagar"
        error={fieldErrors.address}
      />

      <Field
        label="City"
        type="text"
        value={values.city}
        onChange={set('city')}
        required
        placeholder="Kochi"
        error={fieldErrors.city}
      />

      <div className="grid grid-cols-2 gap-3">
        <Field
          label="Opening time"
          type="time"
          value={values.openingTime}
          onChange={set('openingTime')}
          required
          error={fieldErrors.opening_time}
        />
        <Field
          label="Closing time"
          type="time"
          value={values.closingTime}
          onChange={set('closingTime')}
          required
          error={fieldErrors.closing_time}
        />
      </div>

      <Field
        label="Capacity"
        type="number"
        inputMode="numeric"
        min={1}
        value={values.capacity}
        onChange={set('capacity')}
        placeholder="22"
        hint="Optional. Number of players the turf comfortably holds."
        error={fieldErrors.capacity}
      />

      <div className="grid grid-cols-2 gap-3">
        <Field
          label="Latitude"
          type="number"
          inputMode="decimal"
          step="any"
          value={values.latitude}
          onChange={set('latitude')}
          placeholder="9.9312"
          hint="Optional."
          error={fieldErrors.latitude}
        />
        <Field
          label="Longitude"
          type="number"
          inputMode="decimal"
          step="any"
          value={values.longitude}
          onChange={set('longitude')}
          placeholder="76.2673"
          hint="Optional."
          error={fieldErrors.longitude}
        />
      </div>

      <SubmitButton pending={pending} pendingLabel={pendingLabel}>
        {submitLabel}
      </SubmitButton>
    </form>
  )
}

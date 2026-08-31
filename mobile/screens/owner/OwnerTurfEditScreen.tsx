import { useEffect, useState } from 'react'
import { StyleSheet, Text, View } from 'react-native'

import { Button, ErrorBanner, LoadingView, Screen, StatusBadge, TextField } from '../../components'
import { createTurf, deleteTurf, fetchMyTurf, submitTurf, updateTurf } from '../../services/owners'
import { ApiError } from '../../services/api'
import { spacing, theme, typography } from '../../theme'
import type { SaveTurfPayload, Turf } from '../../types/owners'

interface Props {
  route: { params: { turfId?: string } }
  navigation: { goBack: () => void }
}

const SUBMITTABLE = new Set(['DRAFT', 'REJECTED'])

/**
 * Create-or-edit form for one of the owner's own turfs. Absent turfId means
 * create: the form starts blank and the primary action inserts a new DRAFT.
 */
export function OwnerTurfEditScreen({ route, navigation }: Props) {
  const turfId = route.params.turfId
  const isEditing = turfId !== undefined

  const [loading, setLoading] = useState(isEditing)
  const [turf, setTurf] = useState<Turf | null>(null)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [address, setAddress] = useState('')
  const [city, setCity] = useState('')
  const [capacity, setCapacity] = useState('')
  const [openingTime, setOpeningTime] = useState('06:00')
  const [closingTime, setClosingTime] = useState('22:00')

  const [saving, setSaving] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!turfId) return
    fetchMyTurf(turfId)
      .then((loaded) => {
        setTurf(loaded)
        setName(loaded.name)
        setDescription(loaded.description ?? '')
        setAddress(loaded.address)
        setCity(loaded.city)
        setCapacity(loaded.capacity !== undefined ? String(loaded.capacity) : '')
        setOpeningTime(loaded.opening_time)
        setClosingTime(loaded.closing_time)
      })
      .catch((cause: unknown) => {
        setError(cause instanceof ApiError ? cause.message : 'Could not load this turf.')
      })
      .finally(() => setLoading(false))
  }, [turfId])

  function payload(): SaveTurfPayload {
    return {
      name,
      description,
      address,
      city,
      latitude: null,
      longitude: null,
      capacity: capacity.trim() === '' ? null : Number(capacity),
      opening_time: openingTime,
      closing_time: closingTime,
    }
  }

  async function handleSave() {
    setSaving(true)
    setError('')
    setFieldErrors({})
    try {
      const saved = isEditing && turfId ? await updateTurf(turfId, payload()) : await createTurf(payload())
      if (isEditing) {
        setTurf(saved)
      } else {
        navigation.goBack()
      }
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fieldErrors())
        setError(cause.message)
      } else {
        setError('Something went wrong. Try again.')
      }
    } finally {
      setSaving(false)
    }
  }

  async function handleSubmitForReview() {
    if (!turfId) return
    setSubmitting(true)
    setError('')
    try {
      setTurf(await submitTurf(turfId))
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Could not submit this turf.')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete() {
    if (!turfId) return
    setDeleting(true)
    setError('')
    try {
      await deleteTurf(turfId)
      navigation.goBack()
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Could not delete this turf.')
      setDeleting(false)
    }
  }

  if (loading) {
    return <LoadingView message="Loading this turf" />
  }

  return (
    <Screen keyboardSafe>
      {turf ? (
        <View style={styles.statusRow}>
          <Text style={styles.title}>{turf.name}</Text>
          <StatusBadge status={turf.status} />
        </View>
      ) : (
        <Text style={styles.title}>New turf</Text>
      )}

      {error ? <ErrorBanner message={error} /> : null}

      {turf && SUBMITTABLE.has(turf.status) && (
        <View style={styles.submitAction}>
          <Button
            label={submitting ? 'Submitting' : 'Submit for review'}
            onPress={() => void handleSubmitForReview()}
            pending={submitting}
          />
        </View>
      )}
      {turf?.status === 'PENDING_APPROVAL' && (
        <View style={styles.notice}>
          <Text style={styles.noticeText}>Waiting for admin review. Not visible to players yet.</Text>
        </View>
      )}
      {turf?.status === 'SUSPENDED' && (
        <View style={[styles.notice, styles.noticeDanger]}>
          <Text style={[styles.noticeText, styles.noticeDangerText]}>
            This turf has been suspended and is not visible to players.
          </Text>
        </View>
      )}

      <View style={styles.form}>
        <TextField label="Name" value={name} onChangeText={setName} error={fieldErrors.name} placeholder="Riverside Turf" />
        <TextField
          label="Description"
          value={description}
          onChangeText={setDescription}
          error={fieldErrors.description}
          multiline
          numberOfLines={4}
          placeholder="What makes this turf worth booking"
        />
        <TextField label="Address" value={address} onChangeText={setAddress} error={fieldErrors.address} placeholder="123 River Road" />
        <TextField label="City" value={city} onChangeText={setCity} error={fieldErrors.city} placeholder="Kochi" />
        <TextField
          label="Capacity"
          value={capacity}
          onChangeText={setCapacity}
          error={fieldErrors.capacity}
          keyboardType="number-pad"
          placeholder="22"
        />
        <View style={styles.timeRow}>
          <View style={styles.timeField}>
            <TextField label="Opening time" value={openingTime} onChangeText={setOpeningTime} error={fieldErrors.opening_time} placeholder="06:00" />
          </View>
          <View style={styles.timeField}>
            <TextField label="Closing time" value={closingTime} onChangeText={setClosingTime} error={fieldErrors.closing_time} placeholder="22:00" />
          </View>
        </View>

        <Button label={isEditing ? 'Save details' : 'Create turf'} onPress={() => void handleSave()} pending={saving} />
      </View>

      {isEditing && (
        <View style={styles.deleteAction}>
          <Button label={deleting ? 'Deleting' : 'Delete turf'} variant="danger" onPress={() => void handleDelete()} pending={deleting} />
        </View>
      )}
    </Screen>
  )
}

const styles = StyleSheet.create({
  statusRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: spacing.sm },
  title: { ...typography.screenTitle, color: theme.textPrimary, flexShrink: 1 },
  submitAction: { marginTop: spacing.lg },
  notice: { marginTop: spacing.lg, backgroundColor: theme.warningSurface, borderRadius: 12, padding: spacing.md },
  noticeText: { ...typography.caption, color: theme.warningText },
  noticeDanger: { backgroundColor: theme.dangerSurface },
  noticeDangerText: { color: theme.dangerText },
  form: { marginTop: spacing.xl },
  timeRow: { flexDirection: 'row', gap: spacing.md },
  timeField: { flex: 1 },
  deleteAction: { marginTop: spacing.xxl },
})

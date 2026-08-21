import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { SlotStatusBadge } from '@/components/owners/SlotStatusBadge'
import { Field } from '@/components/form/Field'
import { FormAlert } from '@/components/form/FormAlert'
import { TextArea } from '@/components/form/TextArea'
import { ApiError } from '@/lib/api'
import {
  blockDate,
  blockTimeRange,
  deleteSlot,
  fetchBlockedDates,
  fetchBlockedTimeRanges,
  fetchMySlots,
  generateSlots,
  setSlotStatus,
  unblockDate,
  unblockTimeRange,
  updateSlotSettings,
  type BlockedDate,
  type BlockedTimeRange,
  type Slot,
} from '@/lib/availability'
import { addDays, today } from '@/lib/date'
import { fetchMyTurf, type Turf } from '@/lib/owners'

type LoadState = { kind: 'loading' } | { kind: 'ready' } | { kind: 'failed'; message: string }

/**
 * The owner's availability workspace for one turf: slot duration and price,
 * bulk slot generation, a per-date slot list with block/unblock/delete, and
 * the two kinds of block. Sections stack on mobile, matching
 * OwnerTurfEditPage's own layout for the same reason: one screen, no tabs to
 * lose state switching between.
 */
export function OwnerTurfAvailabilityPage() {
  const { turfId = '' } = useParams()

  const [load, setLoad] = useState<LoadState>({ kind: 'loading' })
  const [turf, setTurf] = useState<Turf | null>(null)

  const [durationInput, setDurationInput] = useState('60')
  const [priceInput, setPriceInput] = useState('')
  const [savingSettings, setSavingSettings] = useState(false)
  const [settingsMessage, setSettingsMessage] = useState('')

  const [generateFrom, setGenerateFrom] = useState(today())
  const [generateTo, setGenerateTo] = useState(addDays(today(), 6))
  const [generating, setGenerating] = useState(false)
  const [generateMessage, setGenerateMessage] = useState('')

  const [viewDate, setViewDate] = useState(today())
  const [slots, setSlots] = useState<Slot[]>([])
  const [slotsLoading, setSlotsLoading] = useState(false)
  const [slotsError, setSlotsError] = useState('')
  const [pendingSlotId, setPendingSlotId] = useState<string | null>(null)

  const [blockedDates, setBlockedDates] = useState<BlockedDate[]>([])
  const [newBlockedDate, setNewBlockedDate] = useState(today())
  const [newBlockedDateReason, setNewBlockedDateReason] = useState('')
  const [blockingDate, setBlockingDate] = useState(false)
  const [blockedDatesError, setBlockedDatesError] = useState('')

  const [blockedRanges, setBlockedRanges] = useState<BlockedTimeRange[]>([])
  const [rangeDate, setRangeDate] = useState(today())
  const [rangeStart, setRangeStart] = useState('')
  const [rangeEnd, setRangeEnd] = useState('')
  const [rangeReason, setRangeReason] = useState('')
  const [blockingRange, setBlockingRange] = useState(false)
  const [blockedRangesError, setBlockedRangesError] = useState('')

  useEffect(() => {
    const controller = new AbortController()

    fetchMyTurf(turfId, controller.signal)
      .then((loaded) => {
        setTurf(loaded)
        if (loaded.slot_duration_minutes) setDurationInput(String(loaded.slot_duration_minutes))
        if (loaded.slot_price !== undefined) setPriceInput(String(loaded.slot_price))
        setLoad({ kind: 'ready' })
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setLoad({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load this turf.',
        })
      })

    return () => controller.abort()
  }, [turfId])

  function loadSlots(date: string) {
    setSlotsLoading(true)
    setSlotsError('')
    fetchMySlots(turfId, date, date)
      .then(setSlots)
      .catch((error: unknown) => {
        setSlotsError(error instanceof ApiError ? error.message : 'Could not load slots for this date.')
      })
      .finally(() => setSlotsLoading(false))
  }

  function loadBlocks() {
    fetchBlockedDates(turfId)
      .then(setBlockedDates)
      .catch((error: unknown) => {
        setBlockedDatesError(error instanceof ApiError ? error.message : 'Could not load blocked dates.')
      })
    fetchBlockedTimeRanges(turfId)
      .then(setBlockedRanges)
      .catch((error: unknown) => {
        setBlockedRangesError(error instanceof ApiError ? error.message : 'Could not load blocked time ranges.')
      })
  }

  useEffect(() => {
    if (load.kind !== 'ready') return
    loadSlots(viewDate)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [load.kind, viewDate])

  useEffect(() => {
    if (load.kind !== 'ready') return
    loadBlocks()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [load.kind])

  async function handleSaveSettings() {
    setSavingSettings(true)
    setSettingsMessage('')
    try {
      const updated = await updateSlotSettings(turfId, {
        slot_duration_minutes: Number(durationInput),
        slot_price: Number(priceInput),
      })
      setTurf(updated)
    } catch (error) {
      setSettingsMessage(error instanceof ApiError ? error.message : 'Could not save slot settings.')
    } finally {
      setSavingSettings(false)
    }
  }

  async function handleGenerate() {
    setGenerating(true)
    setGenerateMessage('')
    try {
      const generated = await generateSlots(turfId, generateFrom, generateTo)
      setGenerateMessage(`${generated.length} slot${generated.length === 1 ? '' : 's'} in this range.`)
      if (viewDate >= generateFrom && viewDate <= generateTo) {
        loadSlots(viewDate)
      }
    } catch (error) {
      setGenerateMessage(error instanceof ApiError ? error.message : 'Could not generate slots.')
    } finally {
      setGenerating(false)
    }
  }

  async function toggleSlot(slot: Slot) {
    setPendingSlotId(slot.id)
    try {
      const updated = await setSlotStatus(turfId, slot.id, slot.status === 'OPEN' ? 'BLOCKED' : 'OPEN')
      setSlots((current) => current.map((s) => (s.id === updated.id ? updated : s)))
    } catch (error) {
      setSlotsError(error instanceof ApiError ? error.message : 'Could not update that slot.')
    } finally {
      setPendingSlotId(null)
    }
  }

  async function removeSlot(slot: Slot) {
    setPendingSlotId(slot.id)
    try {
      await deleteSlot(turfId, slot.id)
      setSlots((current) => current.filter((s) => s.id !== slot.id))
    } catch (error) {
      setSlotsError(error instanceof ApiError ? error.message : 'Could not remove that slot.')
    } finally {
      setPendingSlotId(null)
    }
  }

  async function handleBlockDate() {
    setBlockingDate(true)
    setBlockedDatesError('')
    try {
      const created = await blockDate(turfId, newBlockedDate, newBlockedDateReason)
      setBlockedDates((current) => [...current, created].sort((a, b) => a.date.localeCompare(b.date)))
      setNewBlockedDateReason('')
      if (newBlockedDate === viewDate) loadSlots(viewDate)
    } catch (error) {
      setBlockedDatesError(error instanceof ApiError ? error.message : 'Could not block that date.')
    } finally {
      setBlockingDate(false)
    }
  }

  async function handleUnblockDate(blocked: BlockedDate) {
    try {
      await unblockDate(turfId, blocked.id)
      setBlockedDates((current) => current.filter((b) => b.id !== blocked.id))
      if (blocked.date === viewDate) loadSlots(viewDate)
    } catch (error) {
      setBlockedDatesError(error instanceof ApiError ? error.message : 'Could not unblock that date.')
    }
  }

  async function handleBlockRange() {
    setBlockingRange(true)
    setBlockedRangesError('')
    try {
      const created = await blockTimeRange(turfId, {
        date: rangeDate, start_time: rangeStart, end_time: rangeEnd, reason: rangeReason,
      })
      setBlockedRanges((current) =>
        [...current, created].sort((a, b) => a.date.localeCompare(b.date) || a.start_time.localeCompare(b.start_time)),
      )
      setRangeStart('')
      setRangeEnd('')
      setRangeReason('')
      if (rangeDate === viewDate) loadSlots(viewDate)
    } catch (error) {
      setBlockedRangesError(error instanceof ApiError ? error.message : 'Could not block that time range.')
    } finally {
      setBlockingRange(false)
    }
  }

  async function handleUnblockRange(blocked: BlockedTimeRange) {
    try {
      await unblockTimeRange(turfId, blocked.id)
      setBlockedRanges((current) => current.filter((b) => b.id !== blocked.id))
      if (blocked.date === viewDate) loadSlots(viewDate)
    } catch (error) {
      setBlockedRangesError(error instanceof ApiError ? error.message : 'Could not unblock that time range.')
    }
  }

  if (load.kind === 'loading') {
    return <p className="text-neutral-500">Loading availability.</p>
  }

  if (load.kind === 'failed' || !turf) {
    return (
      <div className="mx-auto w-full max-w-md">
        <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-sm text-red-800">{load.kind === 'failed' ? load.message : 'This turf could not be loaded.'}</p>
        </div>
        <Link to="/owner/turfs" className="mt-4 inline-block text-sm font-medium text-pitch-700 underline">
          Back to your turfs
        </Link>
      </div>
    )
  }

  const settingsConfigured = turf.slot_duration_minutes !== undefined && turf.slot_price !== undefined

  return (
    <div className="mx-auto w-full max-w-md">
      <p className="text-sm">
        <Link to={`/owner/turfs/${turfId}/edit`} className="text-pitch-700 underline">
          {turf.name}
        </Link>
      </p>
      <h1 className="mt-1 text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">Availability</h1>
      <p className="mt-1 text-sm text-neutral-500">
        Operating hours are {turf.opening_time}–{turf.closing_time}, set on the turf's own details.
      </p>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Slot settings</h2>
        <div className="mt-3 space-y-3 rounded-xl border border-neutral-200 bg-white p-4">
          {settingsMessage && <FormAlert message={settingsMessage} />}
          <Field
            label="Slot duration (minutes)"
            type="number"
            inputMode="numeric"
            min={15}
            max={240}
            step={15}
            value={durationInput}
            onChange={(e) => setDurationInput(e.target.value)}
          />
          <Field
            label="Price per slot"
            type="number"
            inputMode="decimal"
            min={0}
            step="0.01"
            value={priceInput}
            onChange={(e) => setPriceInput(e.target.value)}
          />
          <button
            type="button"
            onClick={() => void handleSaveSettings()}
            disabled={savingSettings}
            className="w-full rounded-lg bg-pitch-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-pitch-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
          >
            {savingSettings ? 'Saving' : 'Save settings'}
          </button>
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Generate slots</h2>
        <div className="mt-3 space-y-3 rounded-xl border border-neutral-200 bg-white p-4">
          {!settingsConfigured && (
            <p className="text-sm text-amber-700">Save a slot duration and price above first.</p>
          )}
          {generateMessage && <p className="text-sm text-neutral-600">{generateMessage}</p>}
          <div className="grid grid-cols-2 gap-3">
            <Field label="From" type="date" value={generateFrom} onChange={(e) => setGenerateFrom(e.target.value)} />
            <Field label="To" type="date" value={generateTo} onChange={(e) => setGenerateTo(e.target.value)} />
          </div>
          <button
            type="button"
            onClick={() => void handleGenerate()}
            disabled={generating || !settingsConfigured}
            className="w-full rounded-lg bg-pitch-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-pitch-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
          >
            {generating ? 'Generating' : 'Generate slots'}
          </button>
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Slots for a date</h2>
        <div className="mt-3">
          <Field label="Date" type="date" value={viewDate} onChange={(e) => setViewDate(e.target.value)} />

          <div className="mt-3">
            {slotsLoading && <p className="text-sm text-neutral-500">Loading slots.</p>}
            {slotsError && <FormAlert message={slotsError} />}
            {!slotsLoading && !slotsError && slots.length === 0 && (
              <div className="rounded-xl border border-dashed border-neutral-300 p-4 text-center">
                <p className="text-sm text-neutral-600">No slots generated for this date yet.</p>
              </div>
            )}
            {!slotsLoading && slots.length > 0 && (
              <ul className="space-y-2">
                {slots.map((slot) => (
                  <li
                    key={slot.id}
                    className="flex items-center justify-between gap-2 rounded-xl border border-neutral-200 bg-white p-3"
                  >
                    <div className="min-w-0">
                      <p className="font-medium text-neutral-900">
                        {slot.start_time}–{slot.end_time}
                      </p>
                      <p className="text-sm text-neutral-500">₹{slot.price}</p>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      <SlotStatusBadge slot={slot} />
                      <button
                        type="button"
                        onClick={() => void toggleSlot(slot)}
                        disabled={pendingSlotId === slot.id}
                        className="rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-medium text-neutral-700 transition-colors hover:bg-neutral-100 disabled:cursor-not-allowed disabled:text-neutral-300"
                      >
                        {slot.status === 'OPEN' ? 'Block' : 'Unblock'}
                      </button>
                      <button
                        type="button"
                        onClick={() => void removeSlot(slot)}
                        disabled={pendingSlotId === slot.id}
                        className="rounded-lg border border-red-200 bg-white px-2.5 py-1.5 text-xs font-medium text-red-700 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:text-neutral-300"
                      >
                        Delete
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Blocked dates</h2>
        <div className="mt-3 space-y-3">
          {blockedDatesError && <FormAlert message={blockedDatesError} />}

          {blockedDates.length > 0 && (
            <ul className="space-y-2">
              {blockedDates.map((b) => (
                <li key={b.id} className="flex items-center justify-between gap-2 rounded-xl border border-neutral-200 bg-white p-3">
                  <div className="min-w-0">
                    <p className="font-medium text-neutral-900">{b.date}</p>
                    {b.reason && <p className="truncate text-sm text-neutral-500">{b.reason}</p>}
                  </div>
                  <button
                    type="button"
                    onClick={() => void handleUnblockDate(b)}
                    className="shrink-0 rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-medium text-neutral-700 transition-colors hover:bg-neutral-100"
                  >
                    Unblock
                  </button>
                </li>
              ))}
            </ul>
          )}

          <div className="space-y-3 rounded-xl border border-neutral-200 bg-white p-4">
            <Field label="Date to block" type="date" value={newBlockedDate} onChange={(e) => setNewBlockedDate(e.target.value)} />
            <Field
              label="Reason (optional)"
              value={newBlockedDateReason}
              onChange={(e) => setNewBlockedDateReason(e.target.value)}
              placeholder="e.g. Public holiday"
            />
            <button
              type="button"
              onClick={() => void handleBlockDate()}
              disabled={blockingDate}
              className="w-full rounded-lg border border-red-200 bg-white px-4 py-2.5 text-sm font-medium text-red-700 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:text-neutral-400"
            >
              {blockingDate ? 'Blocking' : 'Block this date'}
            </button>
          </div>
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Blocked time ranges</h2>
        <div className="mt-3 space-y-3">
          {blockedRangesError && <FormAlert message={blockedRangesError} />}

          {blockedRanges.length > 0 && (
            <ul className="space-y-2">
              {blockedRanges.map((b) => (
                <li key={b.id} className="flex items-center justify-between gap-2 rounded-xl border border-neutral-200 bg-white p-3">
                  <div className="min-w-0">
                    <p className="font-medium text-neutral-900">
                      {b.date}, {b.start_time}–{b.end_time}
                    </p>
                    {b.reason && <p className="truncate text-sm text-neutral-500">{b.reason}</p>}
                  </div>
                  <button
                    type="button"
                    onClick={() => void handleUnblockRange(b)}
                    className="shrink-0 rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-medium text-neutral-700 transition-colors hover:bg-neutral-100"
                  >
                    Unblock
                  </button>
                </li>
              ))}
            </ul>
          )}

          <div className="space-y-3 rounded-xl border border-neutral-200 bg-white p-4">
            <Field label="Date" type="date" value={rangeDate} onChange={(e) => setRangeDate(e.target.value)} />
            <div className="grid grid-cols-2 gap-3">
              <Field label="Start" type="time" value={rangeStart} onChange={(e) => setRangeStart(e.target.value)} />
              <Field label="End" type="time" value={rangeEnd} onChange={(e) => setRangeEnd(e.target.value)} />
            </div>
            <TextArea
              label="Reason (optional)"
              value={rangeReason}
              onChange={(e) => setRangeReason(e.target.value)}
              rows={2}
              placeholder="e.g. Pitch resurfacing"
            />
            <button
              type="button"
              onClick={() => void handleBlockRange()}
              disabled={blockingRange || !rangeStart || !rangeEnd}
              className="w-full rounded-lg border border-red-200 bg-white px-4 py-2.5 text-sm font-medium text-red-700 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:text-neutral-400"
            >
              {blockingRange ? 'Blocking' : 'Block this time range'}
            </button>
          </div>
        </div>
      </section>
    </div>
  )
}

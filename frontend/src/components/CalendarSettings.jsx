import { useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  DateField,
  Input,
  Select,
  SkeletonLines,
  StatusBadge,
  useConfirm,
  useToast,
} from './ui'
import api from '../lib/api'
import { useResource } from '../lib/hooks'
import { WEEKDAYS, formatDate, today } from '../lib/time'

const EMPTY = { date: '', name: '', workingDay: false, follows: '0' }

// CalendarSettings edits the weekly off days and the holiday list. Off days and holidays
// only stop the automatic generation of the fixed routine: a class, lab, makeup class or
// exam somebody creates for such a date is kept, published and notified as on any other
// day.
export default function CalendarSettings({ base = '/api/admin', title = 'Off days and holidays' }) {
  const toast = useToast()
  const confirm = useConfirm()
  const calendar = useResource(`${base}/calendar`)
  const [form, setForm] = useState(EMPTY)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState('')
  const [offBusy, setOffBusy] = useState(false)
  const offDays = calendar.data?.off_days || []
  const holidays = calendar.data?.holidays || []
  const set = (key) => (value) => setForm((current) => ({ ...current, [key]: value }))
  const toggleOffDay = async (day) => {
    const next = offDays.includes(day)
      ? offDays.filter((entry) => entry !== day)
      : [...offDays, day].sort((left, right) => left - right)
    setOffBusy(true)
    try {
      await api.put(`${base}/calendar/off-days`, { off_days: next })
      toast.success('Weekly off days saved')
      calendar.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setOffBusy(false)
    }
  }
  const save = async (event) => {
    event.preventDefault()
    setError('')
    if (!form.date) {
      setError('Pick a date.')
      return
    }
    if (!form.workingDay && !form.name.trim()) {
      setError('A holiday needs a name.')
      return
    }
    setSaving(true)
    try {
      const saved = await api.put(`${base}/calendar/holidays`, {
        date: form.date,
        name: form.name.trim(),
        working_day: form.workingDay,
        follows_weekday: form.workingDay ? Number(form.follows) : null,
      })
      const removed = saved?.removed_items || 0
      toast.success(
        removed > 0
          ? `Saved · ${removed} generated ${removed === 1 ? 'class' : 'classes'} removed`
          : 'Saved',
      )
      setForm(EMPTY)
      calendar.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const remove = async (entry) => {
    const ok = await confirm({
      title: 'Remove this entry?',
      message: `${formatDate(entry.date)} goes back to being an ordinary day.`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    setBusy(entry.date)
    try {
      await api.delete(`${base}/calendar/holidays/${entry.date}`)
      toast.success('Removed')
      calendar.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  return (
    <Card hover={false} className="mb-5">
      <CardHeader title={title} />
      {calendar.error && (
        <Alert tone="error" className="mt-3">
          {calendar.error}
        </Alert>
      )}
      <p className="mt-3 text-sm text-muted">
        Off days and holidays only stop the automatic fixed routine. A class or exam
        created for such a date is still kept, published and announced.
      </p>
      <div className="mt-4">
        <p className="text-sm font-semibold text-ink-soft">Weekly off days</p>
        <div className="mt-2 flex flex-wrap gap-2">
          {WEEKDAYS.map((name, day) => (
            <button
              key={name}
              type="button"
              disabled={offBusy || calendar.loading}
              onClick={() => toggleOffDay(day)}
              aria-pressed={offDays.includes(day)}
              className={
                offDays.includes(day)
                  ? 'press rounded-full bg-brand-600 px-3 py-1.5 text-sm font-semibold text-page'
                  : 'press rounded-full border border-line px-3 py-1.5 text-sm font-semibold text-ink-soft'
              }
            >
              {name}
            </button>
          ))}
        </div>
      </div>
      <form onSubmit={save} className="mt-5 flex flex-col gap-3">
        <p className="text-sm font-semibold text-ink-soft">Add a holiday or a working day</p>
        {error && <Alert tone="error">{error}</Alert>}
        <div className="grid gap-3 sm:grid-cols-2">
          <DateField label="Date" value={form.date} min={today()} onChange={set('date')} required />
          <Input
            label={form.workingDay ? 'Note' : 'Holiday name'}
            value={form.name}
            onChange={(event) => set('name')(event.target.value)}
            placeholder={form.workingDay ? 'Make-up Saturday' : 'Victory Day'}
          />
        </div>
        <label className="flex items-center gap-2 text-sm font-semibold text-ink-soft">
          <input
            type="checkbox"
            checked={form.workingDay}
            onChange={(event) => set('workingDay')(event.target.checked)}
            className="size-4 accent-brand-600"
          />
          This day is a working day
        </label>
        {form.workingDay && (
          <Select
            label="It follows the routine of"
            value={form.follows}
            onChange={(event) => set('follows')(event.target.value)}
            className="max-w-xs"
          >
            {WEEKDAYS.map((name, day) => (
              <option key={name} value={String(day)}>
                {name}
              </option>
            ))}
          </Select>
        )}
        <Button type="submit" loading={saving} className="self-start">
          <Plus size={16} />
          Save
        </Button>
      </form>
      <div className="mt-5">
        <p className="text-sm font-semibold text-ink-soft">Coming up</p>
        {calendar.loading ? (
          <SkeletonLines className="mt-2" />
        ) : holidays.length === 0 ? (
          <p className="mt-2 text-sm text-muted">
            No holiday or working day is set beyond the weekly off days.
          </p>
        ) : (
          <ul className="mt-2 flex flex-col gap-2">
            {holidays.map((entry) => (
              <li
                key={entry.date}
                className="flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-xl border border-line p-3"
              >
                <span className="font-medium text-ink">{entry.label}</span>
                <StatusBadge
                  tone={entry.working_day ? 'success' : 'warning'}
                  label={
                    entry.working_day
                      ? `Working day · follows ${entry.follows_weekday_name}`
                      : 'Holiday'
                  }
                />
                {entry.name && <span className="text-sm text-muted">{entry.name}</span>}
                <Button
                  size="sm"
                  variant="danger"
                  loading={busy === entry.date}
                  onClick={() => remove(entry)}
                  className="ml-auto"
                >
                  <Trash2 size={14} />
                  Remove
                </Button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Card>
  )
}

import { useEffect, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  Input,
  PageHeader,
  Select,
  SkeletonLines,
  Textarea,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { readSite } from '../../lib/site'

const FEATURE_LABELS = {
  feature_attendance: 'Attendance',
  feature_polls: 'Polls',
  feature_messages: 'Messages',
  feature_group_chats: 'Group chats',
  feature_web_attendance: 'Attendance from the website',
}

const ROLE_CHOICES = [
  { value: 'student', label: 'Students' },
  { value: 'cr', label: 'CRs' },
  { value: 'teacher', label: 'Teachers' },
  { value: 'admin', label: 'Office admins' },
]

const ON = 'on'

const OFF = 'off'

function Switch({ label, on, onChange }) {
  return (
    <label className="tap-target flex items-center justify-between gap-3 rounded-tile bg-wash px-3">
      <span className="text-sm font-semibold text-ink">{label}</span>
      <input
        type="checkbox"
        checked={on}
        onChange={(event) => onChange(event.target.checked ? ON : OFF)}
        className="size-5 accent-brand-600"
      />
    </label>
  )
}

export default function ControlSettingsPage() {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/super-admin/settings')
  const [values, setValues] = useState({})
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    if (data?.values) setValues(data.values)
  }, [data])
  const set = (key) => (value) => setValues((current) => ({ ...current, [key]: value }))
  const field = (key) => values[key] || ''
  const on = (key) => String(values[key] || '').toLowerCase() === ON
  const roles = (values.announcement_roles || '')
    .split(',')
    .map((entry) => entry.trim())
    .filter(Boolean)
  const toggleRole = (role) => {
    const next = roles.includes(role) ? roles.filter((entry) => entry !== role) : [...roles, role]
    set('announcement_roles')(next.join(','))
  }
  const save = async () => {
    setSaving(true)
    try {
      const result = await api.put('/api/super-admin/settings', { values })
      toast.success(result.message)
      await reload()
      await readSite()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setSaving(false)
    }
  }
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Site settings"
        subtitle="Every change takes effect within half a minute. No redeploy is needed."
        actions={
          <Button onClick={save} loading={saving}>
            Save settings
          </Button>
        }
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonLines lines={6} />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card hover={false}>
            <CardHeader
              title="Maintenance mode"
              subtitle="Everyone except the office and the control room sees a calm page."
            />
            <div className="mt-4 flex flex-col gap-3">
              <Switch label="Maintenance mode" on={on('maintenance_mode')} onChange={set('maintenance_mode')} />
              <Textarea
                label="Message"
                value={field('maintenance_message')}
                onChange={(event) => set('maintenance_message')(event.target.value)}
                rows={3}
              />
            </div>
          </Card>
          <Card hover={false}>
            <CardHeader title="Registration" subtitle="Whether new students may sign up." />
            <div className="mt-4 flex flex-col gap-3">
              <Switch
                label="Registration open"
                on={on('registration_open')}
                onChange={set('registration_open')}
              />
              <Textarea
                label="Message when closed"
                value={field('registration_message')}
                onChange={(event) => set('registration_message')(event.target.value)}
                rows={3}
              />
            </div>
          </Card>
          <Card hover={false}>
            <CardHeader title="Announcement banner" subtitle="Shown at the top for the roles you pick." />
            <div className="mt-4 flex flex-col gap-3">
              <Textarea
                label="Announcement"
                value={field('announcement_text')}
                onChange={(event) => set('announcement_text')(event.target.value)}
                rows={3}
              />
              <div className="flex flex-wrap gap-2">
                {ROLE_CHOICES.map((choice) => (
                  <label
                    key={choice.value}
                    className="tap-target inline-flex items-center gap-2 rounded-tile bg-wash px-3 text-sm"
                  >
                    <input
                      type="checkbox"
                      checked={roles.includes(choice.value)}
                      onChange={() => toggleRole(choice.value)}
                      className="size-4 accent-brand-600"
                    />
                    {choice.label}
                  </label>
                ))}
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <Input
                  label="Until"
                  type="date"
                  value={field('announcement_until')}
                  onChange={(event) => set('announcement_until')(event.target.value)}
                />
                <Select
                  label="Severity"
                  value={field('announcement_severity') || 'info'}
                  onChange={(event) => set('announcement_severity')(event.target.value)}
                >
                  <option value="info">Information</option>
                  <option value="warning">Warning</option>
                  <option value="error">Serious</option>
                </Select>
              </div>
            </div>
          </Card>
          <Card hover={false}>
            <CardHeader title="Features" subtitle="Switch a part of the portal off for everybody." />
            <div className="mt-4 flex flex-col gap-2">
              {(data?.features || []).map((key) => (
                <Switch key={key} label={FEATURE_LABELS[key] || key} on={on(key)} onChange={set(key)} />
              ))}
            </div>
          </Card>
          <Card hover={false}>
            <CardHeader
              title="Reminders and the routine window"
              subtitle="These used to live in the environment. The environment is only the default now."
            />
            <div className="mt-4 grid gap-3 sm:grid-cols-3">
              <Input
                label="Evening reminders"
                value={field('reminder_times')}
                onChange={(event) => set('reminder_times')(event.target.value)}
                hint="Like 16:00,21:00"
              />
              <Input
                label="Morning reminder"
                value={field('morning_reminder')}
                onChange={(event) => set('morning_reminder')(event.target.value)}
                hint="Like 07:30"
              />
              <Input
                label="Days ahead"
                value={field('schedule_days_ahead')}
                onChange={(event) => set('schedule_days_ahead')(event.target.value)}
                inputMode="numeric"
              />
            </div>
          </Card>
          <Card hover={false}>
            <CardHeader
              title="Minimum app version"
              subtitle="Older apps show a blocking Update required screen."
            />
            <Input
              label="Minimum version"
              value={field('min_app_version')}
              onChange={(event) => set('min_app_version')(event.target.value)}
              hint="Like 2.6.0, or empty for no floor"
              className="mt-4"
            />
          </Card>
        </div>
      )}
    </>
  )
}

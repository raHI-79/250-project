import { Link2, MapPin } from 'lucide-react'
import { Field, Input } from '../ui'
import { MODE_ONLINE, MODE_ROOM } from '../../lib/schedule'
import cx from '../../lib/cx'

const MODES = [
  { value: MODE_ROOM, label: 'Room', icon: MapPin },
  { value: MODE_ONLINE, label: 'Online', icon: Link2 },
]

export default function SlotFields({ value, onChange, error, linkRequired = true, className }) {
  const mode = value.mode || MODE_ROOM
  const set = (patch) => onChange({ ...value, ...patch })
  return (
    <div className={cx('flex flex-col gap-4', className)}>
      <Field label="Where">
        <div className="inline-flex rounded-lg border border-brand-200 bg-white p-1">
          {MODES.map((entry) => {
            const Icon = entry.icon
            const active = mode === entry.value
            return (
              <button
                key={entry.value}
                type="button"
                onClick={() => set({ mode: entry.value })}
                className={cx(
                  'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  active ? 'bg-brand-600 text-white' : 'text-brand-600 hover:bg-brand-50',
                )}
              >
                <Icon size={14} />
                {entry.label}
              </button>
            )
          })}
        </div>
      </Field>
      {mode === MODE_ROOM ? (
        <Input
          label="Room"
          value={value.room || ''}
          onChange={(event) => set({ room: event.target.value })}
          placeholder="1003"
          error={error}
          required
        />
      ) : (
        <Input
          label="Meeting link"
          type="url"
          value={value.meeting_link || ''}
          onChange={(event) => set({ meeting_link: event.target.value })}
          placeholder="https://meet.example.com/your-class"
          error={error}
          hint={linkRequired ? undefined : 'Optional, you can add it later'}
          required={linkRequired}
        />
      )}
      {mode === MODE_ROOM && (
        <Input
          label="Meeting link"
          type="url"
          value={value.meeting_link || ''}
          onChange={(event) => set({ meeting_link: event.target.value })}
          placeholder="https://meet.example.com/your-class"
          hint="Optional, for students who cannot come"
        />
      )}
    </div>
  )
}

import { Link2, MapPin } from 'lucide-react'
import { Field, Input } from '../ui'
import { MODE_ONLINE, MODE_ROOM } from '../../lib/schedule'
import cx from '../../lib/cx'

const MODES = [
  { value: MODE_ROOM, label: 'Room', icon: MapPin },
  { value: MODE_ONLINE, label: 'Online', icon: Link2 },
]

export default function SlotFields({ value, onChange, error, className }) {
  const mode = value.mode || MODE_ROOM
  const set = (patch) => onChange({ ...value, ...patch })
  return (
    <div className={cx('flex flex-col gap-4', className)}>
      <Field label="Where">
        <div className="inline-flex flex-wrap rounded-xl border border-line-strong bg-surface p-1">
          {MODES.map((entry) => {
            const Icon = entry.icon
            const active = mode === entry.value
            return (
              <button
                key={entry.value}
                type="button"
                onClick={() => set({ mode: entry.value })}
                className={cx(
                  'inline-flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-sm font-medium transition-colors',
                  active ? 'bg-brand-600 text-page' : 'text-brand-600 hover:bg-wash',
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
          hint="Optional, you can add it later"
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

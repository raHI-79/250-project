import cx from '../../lib/cx'
import {
  HOUR_CHOICES,
  LATE_AFTER,
  LATE_AT_END,
  MAX_MINUTES,
  MIN_MINUTES,
  MODE_CUSTOM,
  MODE_QUICK,
  MODE_UNTIL,
  QUICK_MINUTES,
} from '../../lib/attendanceTiming'

const chip = (active) =>
  cx(
    'press min-h-11 shrink-0 rounded-full border px-4 text-sm whitespace-nowrap',
    active
      ? 'border-brand-600 bg-brand-600 font-semibold text-page'
      : 'glass-chip font-medium text-muted hover:border-brand-300 hover:text-ink',
  )

const field =
  'h-11 w-28 rounded-xl border border-line-strong bg-wash px-3 text-sm text-ink tabular-nums outline-none focus:border-brand-600'

export default function SessionTiming({ timing, onChange, rollCall = false, problem = '' }) {
  const set = (patch) => onChange({ ...timing, ...patch })
  return (
    <div className="flex flex-col gap-5">
      <div>
        <p className="label-caps">How long it stays open</p>
        <div role="radiogroup" aria-label="Duration" className="mt-2 flex flex-wrap gap-2">
          {QUICK_MINUTES.map((value) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={timing.mode === MODE_QUICK && timing.minutes === value}
              onClick={() => set({ mode: MODE_QUICK, minutes: value })}
              className={chip(timing.mode === MODE_QUICK && timing.minutes === value)}
            >
              {value} min
            </button>
          ))}
          <button
            type="button"
            role="radio"
            aria-checked={timing.mode === MODE_CUSTOM}
            onClick={() => set({ mode: MODE_CUSTOM })}
            className={chip(timing.mode === MODE_CUSTOM)}
          >
            Custom
          </button>
          <button
            type="button"
            role="radio"
            aria-checked={timing.mode === MODE_UNTIL}
            onClick={() => set({ mode: MODE_UNTIL })}
            className={chip(timing.mode === MODE_UNTIL)}
          >
            Open until
          </button>
        </div>
        {timing.mode === MODE_CUSTOM && (
          <label className="mt-3 flex flex-wrap items-center gap-2 text-sm text-ink">
            <input
              type="number"
              inputMode="numeric"
              min={MIN_MINUTES}
              max={MAX_MINUTES}
              value={timing.custom}
              onChange={(event) => set({ custom: event.target.value })}
              aria-label="Custom minutes"
              placeholder="45"
              className={field}
            />
            minutes (1 to 240)
          </label>
        )}
        {timing.mode === MODE_UNTIL && (
          <label className="mt-3 flex flex-wrap items-center gap-2 text-sm text-ink">
            <input
              type="time"
              value={timing.until}
              onChange={(event) => set({ until: event.target.value })}
              aria-label="Open until"
              className={field}
            />
            today
          </label>
        )}
      </div>
      {!rollCall && (
        <div>
          <p className="label-caps">When a mark counts as late</p>
          <div role="radiogroup" aria-label="Late after" className="mt-2 flex flex-wrap gap-2">
            <button
              type="button"
              role="radio"
              aria-checked={timing.lateMode === LATE_AT_END}
              onClick={() => set({ lateMode: LATE_AT_END })}
              className={chip(timing.lateMode === LATE_AT_END)}
            >
              At the original end
            </button>
            <button
              type="button"
              role="radio"
              aria-checked={timing.lateMode === LATE_AFTER}
              onClick={() => set({ lateMode: LATE_AFTER })}
              className={chip(timing.lateMode === LATE_AFTER)}
            >
              Late after
            </button>
          </div>
          {timing.lateMode === LATE_AFTER ? (
            <label className="mt-3 flex flex-wrap items-center gap-2 text-sm text-ink">
              <input
                type="number"
                inputMode="numeric"
                min={1}
                max={MAX_MINUTES}
                value={timing.lateAfter}
                onChange={(event) => set({ lateAfter: event.target.value })}
                aria-label="Late after minutes"
                placeholder="5"
                className={field}
              />
              minutes from the start
            </label>
          ) : (
            <p className="mt-2 text-xs text-muted">
              Everybody who marks in time is present. If you extend the attendance, marks after the
              original end are saved as Late.
            </p>
          )}
        </div>
      )}
      <div>
        <p className="label-caps">Class hours</p>
        <div role="radiogroup" aria-label="Class hours" className="mt-2 flex flex-wrap gap-2">
          {HOUR_CHOICES.map((value) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={Number(timing.hours) === value}
              onClick={() => set({ hours: value })}
              className={chip(Number(timing.hours) === value)}
            >
              {value} {value === 1 ? 'hour' : 'hours'}
            </button>
          ))}
        </div>
        <p className="mt-2 text-xs text-muted">
          This class counts as {timing.hours} {Number(timing.hours) === 1 ? 'class' : 'classes'} in
          every percentage. It is filled in from the routine and you can change it.
        </p>
      </div>
      {problem && (
        <p role="alert" className="text-sm font-semibold text-danger-600">
          {problem}
        </p>
      )}
    </div>
  )
}

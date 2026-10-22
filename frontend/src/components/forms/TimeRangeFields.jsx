import { Select } from '../ui'
import { periodEndOptions, teachingPeriods } from '../../lib/schedule'
import { formatTimeRange } from '../../lib/time'
import cx from '../../lib/cx'

export default function TimeRangeFields({
  periods,
  start,
  end,
  onStartChange,
  onEndChange,
  error,
  className,
}) {
  const starts = teachingPeriods(periods)
  const ends = periodEndOptions(periods, start)
  const pickStart = (value) => {
    onStartChange(value)
    const first = periodEndOptions(periods, value)[0]
    onEndChange(first ? first.end_time : '')
  }
  return (
    <div className={cx('grid gap-4 sm:grid-cols-2', className)}>
      <Select
        label="From period"
        value={start}
        onChange={(event) => pickStart(event.target.value)}
        hint="Pick the period the class starts in"
        required
      >
        <option value="">Choose a period</option>
        {starts.map((period) => (
          <option key={period.id} value={period.start_time}>
            {period.label} · {formatTimeRange(period.start_time, period.end_time)}
          </option>
        ))}
      </Select>
      <Select
        label="To period"
        value={end}
        onChange={(event) => onEndChange(event.target.value)}
        error={error}
        hint="A lab may run over several periods"
        disabled={!start}
        required
      >
        <option value="">Choose a period</option>
        {ends.map((period) => (
          <option key={period.id} value={period.end_time}>
            {period.label} · ends {formatTimeRange(period.start_time, period.end_time).split(' – ')[1]}
          </option>
        ))}
      </Select>
    </div>
  )
}

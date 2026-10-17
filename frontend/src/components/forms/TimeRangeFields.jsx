import { Input } from '../ui'
import { CLASS_END, CLASS_START } from '../../lib/time'
import cx from '../../lib/cx'

export default function TimeRangeFields({ start, end, onStartChange, onEndChange, error, className }) {
  return (
    <div className={cx('grid gap-4 sm:grid-cols-2', className)}>
      <Input
        label="Start time"
        type="time"
        value={start}
        min={CLASS_START}
        max={CLASS_END}
        onChange={(event) => onStartChange(event.target.value)}
        hint="Class hours are 8:00 AM to 5:00 PM"
        required
      />
      <Input
        label="End time"
        type="time"
        value={end}
        min={CLASS_START}
        max={CLASS_END}
        onChange={(event) => onEndChange(event.target.value)}
        error={error}
        required
      />
    </div>
  )
}

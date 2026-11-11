import { Alert } from '../ui'
import { holidayOn } from '../../lib/dayKind'
import { useResource } from '../../lib/hooks'

// HolidayNote sits under the date of a create form. An off day or a holiday only stops
// the automatic fixed routine, so a class or an exam created for one is still kept,
// published and announced; the note says so instead of refusing the date.
export default function HolidayNote({ date, className }) {
  const calendar = useResource(date ? '/api/calendar' : null)
  const holiday = holidayOn(date, calendar.data)
  if (!holiday) return null
  return (
    <Alert tone="info" className={className}>
      This is a holiday · {holiday.name}. Whatever you create for it is still kept, published
      and announced.
    </Alert>
  )
}

import { Alert, Button, DataTable, Modal } from '../ui'
import { formatDate, formatTimeRange } from '../../lib/time'
import { courseLine, placeOf, sectionShort, statusLabel } from '../../lib/schedule'

const HEADINGS = ['Date', 'Day', 'Time', 'Course', 'Section', 'Where', 'Info']

const infoOf = (item) => {
  const parts = []
  if (item.note) parts.push(item.note)
  if (item.syllabus) parts.push(`Syllabus: ${item.syllabus}`)
  if (item.instructions) parts.push(item.instructions)
  if (item.status !== 'scheduled' && item.status !== 'confirmed') parts.push(statusLabel(item.status))
  return parts.join(' · ')
}

export default function PreviewModal({ open, days, onClose, onPublish, busy, count }) {
  const rows = (days || []).flatMap((day) =>
    day.items.map((item) => ({ ...item, weekday: day.weekday })),
  )
  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Preview the routine"
      className="max-w-4xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          <Button loading={busy} disabled={rows.length === 0} onClick={onPublish}>
            Publish {count} item{count === 1 ? '' : 's'}
          </Button>
        </>
      }
    >
      {rows.length === 0 ? (
        <Alert tone="info">Pick at least one item to preview.</Alert>
      ) : (
        <>
          <Alert tone="info" className="mb-4">
            This is exactly what your batch will read once you publish.
          </Alert>
          <DataTable>
            <thead>
              <tr className="border-b border-brand-100 bg-brand-50/60">
                {HEADINGS.map((heading) => (
                  <th key={heading} className="px-3 py-2 font-semibold text-brand-700">
                    {heading}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((item) => (
                <tr key={item.id} className="border-b border-brand-50 last:border-b-0">
                  <td className="px-3 py-2 whitespace-nowrap text-brand-700">
                    {formatDate(item.item_date)}
                  </td>
                  <td className="px-3 py-2 whitespace-nowrap text-brand-500">{item.weekday}</td>
                  <td className="px-3 py-2 whitespace-nowrap tabular-nums text-brand-600">
                    {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
                  </td>
                  <td className="px-3 py-2 text-brand-900">{courseLine(item)}</td>
                  <td className="px-3 py-2 text-brand-700">{sectionShort(item.section)}</td>
                  <td className="px-3 py-2 text-brand-700">{placeOf(item) || '—'}</td>
                  <td className="px-3 py-2 text-brand-500">{infoOf(item) || '—'}</td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        </>
      )}
    </Modal>
  )
}

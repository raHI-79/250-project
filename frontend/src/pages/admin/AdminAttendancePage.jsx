import { useState } from 'react'
import { Download, PieChart } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  DataTable,
  EmptyState,
  Field,
  PageHeader,
  Select,
  SkeletonTable,
  useToast,
} from '../../components/ui'
import cx from '../../lib/cx'
import { useResource } from '../../lib/hooks'
import { belowPass, classKey, downloadWorkbook, sectionLabel } from '../../lib/attendance'

const HEADINGS = ['Course', 'Batch', 'Section', 'Sessions', 'Average', 'Below 50 %']

export default function AdminAttendancePage() {
  const toast = useToast()
  const [batch, setBatch] = useState('')
  const [busy, setBusy] = useState('')
  const batches = useResource('/api/lookups/batches')
  const { data, loading, error } = useResource('/api/admin/attendance', batch ? { batch } : undefined)
  const items = data?.items || []
  const download = async () => {
    setBusy('export')
    try {
      const saved = await downloadWorkbook(
        '/api/admin/attendance/export',
        batch ? { batch } : {},
        batch ? `attendance_${batch}.xlsx` : 'attendance_all-batches.xlsx',
      )
      toast.success(saved.path ? `Saved as ${saved.name}` : 'The workbook is downloading.')
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  return (
    <>
      <PageHeader
        eyebrow="Department office"
        title="Attendance overview"
        subtitle="Every course of every batch with its average attendance and the students below 50 %"
        actions={
          <Button variant="secondary" loading={busy === 'export'} onClick={download}>
            <Download size={16} /> Excel
          </Button>
        }
      />
      <Card hover={false} className="mb-5">
        <Field label="Batch" htmlFor="attendance-batch" className="max-w-xs">
          <Select
            id="attendance-batch"
            value={batch}
            onChange={(event) => setBatch(event.target.value)}
          >
            <option value="">Every batch</option>
            {(batches.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
        </Field>
      </Card>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={6} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={PieChart}
          title="No attendance yet"
          text="Once the teachers start taking attendance, every course shows its average here."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              {HEADINGS.map((heading) => (
                <th key={heading} className="px-4 py-3 font-semibold text-ink-soft">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((entry) => (
              <tr key={classKey(entry)} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3 font-medium text-ink">
                  {entry.course_code}
                  <span className="ml-2 text-xs text-muted">{entry.course_title}</span>
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">{entry.batch}</td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {sectionLabel(entry.section)}
                </td>
                <td className="px-4 py-3 text-ink-soft">{entry.sessions}</td>
                <td
                  className={cx(
                    'px-4 py-3 font-bold',
                    belowPass(entry.average_rate) ? 'text-danger-600' : 'text-ink',
                  )}
                >
                  {entry.average_rate} %
                </td>
                <td className="px-4 py-3 text-ink-soft">{entry.below_fifty}</td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

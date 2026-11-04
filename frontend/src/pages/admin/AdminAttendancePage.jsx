import { useState } from 'react'
import { CalendarCheck, Download, PieChart, Trash2 } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  DataTable,
  EmptyState,
  Field,
  PageHeader,
  Select,
  SkeletonCard,
  SkeletonTable,
  StatusTabs,
  useConfirm,
  useToast,
} from '../../components/ui'
import AdminPhoneChanges from '../../components/attendance/AdminPhoneChanges'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { useQueryTab, useResource } from '../../lib/hooks'
import { formatClock, formatDate } from '../../lib/time'
import {
  OFFICE_TAB_HISTORY,
  OFFICE_TAB_PHONES,
  OFFICE_TAB_SHEETS,
  belowPass,
  classKey,
  deleteSheetQuestion,
  downloadWorkbook,
  sectionLabel,
} from '../../lib/attendance'

const HEADINGS = ['Course', 'Batch', 'Section', 'Sessions', 'Average', 'Below 50 %', '']

const TABS = [
  { value: OFFICE_TAB_HISTORY, label: 'History' },
  { value: OFFICE_TAB_SHEETS, label: 'Sheets' },
  { value: OFFICE_TAB_PHONES, label: 'Phone changes' },
]

const TAB_VALUES = TABS.map((entry) => entry.value)

export default function AdminAttendancePage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [tab, setTab] = useQueryTab(TAB_VALUES, OFFICE_TAB_HISTORY)
  const [batch, setBatch] = useState('')
  const [busy, setBusy] = useState('')
  const batches = useResource('/api/lookups/batches')
  const sheets = useResource(
    tab === OFFICE_TAB_SHEETS ? '/api/admin/attendance' : null,
    batch ? { batch } : undefined,
  )
  const sessions = useResource(
    tab === OFFICE_TAB_HISTORY ? '/api/admin/attendance/sessions' : null,
    batch ? { batch } : undefined,
  )
  const items = sheets.data?.items || []
  const rows = sessions.data?.items || []
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
  const removeSheet = async (entry) => {
    const ok = await confirm({
      title: 'Delete the attendance sheet?',
      message: deleteSheetQuestion(entry),
      confirmLabel: 'Delete',
    })
    if (!ok) return
    setBusy(`delete-${classKey(entry)}`)
    try {
      await api.delete('/api/admin/attendance/sheet', {
        course_id: entry.course_id,
        batch: entry.batch,
        section: entry.section,
      })
      toast.success('The attendance sheet is deleted.')
      sheets.reload()
      sessions.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const removeSession = async (session) => {
    const ok = await confirm({
      title: 'Delete this attendance session?',
      message: `${session.course_code} · ${session.batch} ${session.section} on ${formatDate(
        session.started_at,
      )} is removed for everybody. This cannot be undone.`,
      confirmLabel: 'Delete',
    })
    if (!ok) return
    setBusy(`session-${session.id}`)
    try {
      await api.delete(`/api/admin/attendance/sessions/${session.id}`)
      toast.success('The attendance session is deleted.')
      sessions.reload()
      sheets.reload()
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
        title="Attendance"
        subtitle="Every session the teachers took, the sheet of every course and the phone changes the teachers decided"
        actions={
          <Button variant="secondary" loading={busy === 'export'} onClick={download}>
            <Download size={16} /> Excel
          </Button>
        }
      />
      <StatusTabs tabs={TABS} value={tab} onChange={setTab} className="mb-5" />
      {tab !== OFFICE_TAB_PHONES && (
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
      )}
      {tab === OFFICE_TAB_PHONES && <AdminPhoneChanges />}
      {tab === OFFICE_TAB_HISTORY && (
        <>
          {sessions.error && (
            <Alert tone="error" className="mb-5">
              {sessions.error}
            </Alert>
          )}
          {sessions.loading ? (
            <SkeletonCard />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={CalendarCheck}
              title="No attendance yet"
              text="Every session a teacher takes shows up here with its method and its count."
            />
          ) : (
            <Card hover={false} flush>
              <ul className="divide-y divide-line">
                {rows.map((session) => (
                  <li key={session.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                    <span className="min-w-0 flex-1">
                      <span className="block wrap-long text-sm font-semibold text-ink">
                        {session.course_code} · {session.course_title}
                      </span>
                      <span className="mt-0.5 block wrap-long text-xs text-muted">
                        {session.batch} · {sectionLabel(session.section)} ·{' '}
                        {formatDate(session.started_at)} · {formatClock(session.started_at)} ·{' '}
                        {session.method === 'roll_call' ? 'Roll call' : 'Bluetooth'}
                      </span>
                    </span>
                    <span className="shrink-0 text-sm font-semibold text-ink">
                      {session.present}/{session.roster}
                    </span>
                    <Button
                      variant="danger"
                      size="sm"
                      loading={busy === `session-${session.id}`}
                      onClick={() => removeSession(session)}
                    >
                      <Trash2 size={15} /> Delete
                    </Button>
                  </li>
                ))}
              </ul>
            </Card>
          )}
        </>
      )}
      {tab === OFFICE_TAB_SHEETS && (
        <>
          {sheets.error && (
            <Alert tone="error" className="mb-5">
              {sheets.error}
            </Alert>
          )}
          {sheets.loading ? (
            <SkeletonTable columns={7} />
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
                    <td className="px-4 py-3 text-ink-soft">{entry.batch}</td>
                    <td className="px-4 py-3 text-ink-soft">{sectionLabel(entry.section)}</td>
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
                    <td className="px-4 py-3">
                      <Button
                        variant="danger"
                        size="sm"
                        loading={busy === `delete-${classKey(entry)}`}
                        onClick={() => removeSheet(entry)}
                      >
                        <Trash2 size={15} /> Delete
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </DataTable>
          )}
        </>
      )}
    </>
  )
}

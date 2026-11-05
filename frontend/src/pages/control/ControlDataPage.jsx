import { Database, Download, HardDriveDownload } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  DataTable,
  EmptyState,
  PageHeader,
  SkeletonLines,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { formatDateTime } from '../../lib/time'

const EXPORTS = [
  { label: 'Users', path: '/api/super-admin/users/export', file: 'users.csv' },
  { label: 'Activity log', path: '/api/admin/activity/export', file: 'activity-log.csv' },
  { label: 'Attendance summary', path: '/api/admin/attendance/export', file: 'attendance.csv' },
]

export default function ControlDataPage() {
  const toast = useToast()
  const { data, loading, error } = useResource('/api/super-admin/data')
  const download = async (entry) => {
    try {
      const text = await api.get(entry.path)
      const blob = new Blob([typeof text === 'string' ? text : JSON.stringify(text)], {
        type: 'text/csv;charset=utf-8',
      })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = entry.file
      link.click()
      URL.revokeObjectURL(url)
    } catch (failure) {
      toast.error(failure.message)
    }
  }
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Data"
        subtitle="Backups, exports and how long the portal keeps its own records"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonLines lines={5} />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card hover={false}>
            <CardHeader title="Last backup" subtitle="Reported by the nightly workflow." />
            <p className="font-display mt-4 text-section font-bold text-ink">
              {data?.backup_at ? formatDateTime(data.backup_at) : 'Never'}
            </p>
            {data?.backup_note && <p className="mt-1 text-sm text-muted">{data.backup_note}</p>}
            {(data?.backups || []).length > 0 && (
              <ul className="mt-4 flex flex-col gap-1 text-sm">
                {data.backups.map((entry) => (
                  <li key={entry.id} className="flex justify-between gap-3">
                    <span className="wrap-long text-ink-soft">{entry.target_id || 'Dump'}</span>
                    <span className="shrink-0 text-xs text-muted">
                      {formatDateTime(entry.created_at)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </Card>
          <Card hover={false}>
            <CardHeader title="Exports" subtitle="A CSV of the current data." />
            <div className="mt-4 flex flex-col gap-2">
              {EXPORTS.map((entry) => (
                <Button key={entry.path} variant="secondary" onClick={() => download(entry)}>
                  <Download size={16} aria-hidden="true" />
                  {entry.label}
                </Button>
              ))}
            </div>
          </Card>
          <Card hover={false} flush>
            <CardHeader title="What the portal keeps" subtitle="Rows stored right now." />
            <DataTable>
              <tbody>
                <tr className="border-b border-line">
                  <td className="px-4 py-3 text-ink-soft">Page views</td>
                  <td className="px-4 py-3 text-right font-semibold text-ink">{data?.views ?? 0}</td>
                  <td className="px-4 py-3 text-right text-xs text-muted">
                    kept {data?.retention?.analytics} days
                  </td>
                </tr>
                <tr className="border-b border-line">
                  <td className="px-4 py-3 text-ink-soft">Activity events</td>
                  <td className="px-4 py-3 text-right font-semibold text-ink">{data?.events ?? 0}</td>
                  <td className="px-4 py-3 text-right text-xs text-muted">
                    kept {data?.retention?.audit} days
                  </td>
                </tr>
                <tr className="border-b border-line">
                  <td className="px-4 py-3 text-ink-soft">Notifications</td>
                  <td className="px-4 py-3 text-right font-semibold text-ink">
                    {data?.notifications ?? 0}
                  </td>
                  <td className="px-4 py-3 text-right text-xs text-muted">
                    kept {data?.retention?.notifications} days
                  </td>
                </tr>
                <tr>
                  <td className="px-4 py-3 text-ink-soft">Open sessions</td>
                  <td className="px-4 py-3 text-right font-semibold text-ink">
                    {data?.sessions ?? 0}
                  </td>
                  <td className="px-4 py-3 text-right text-xs text-muted">until they expire</td>
                </tr>
              </tbody>
            </DataTable>
          </Card>
          <Card hover={false}>
            <CardHeader
              title="Retention"
              subtitle="A nightly job deletes anything older than these windows."
            />
            <EmptyState
              icon={Database}
              title="Set the windows in Site settings"
              text="Analytics, activity events and notifications each have their own number of days."
            />
            <Button as="a" href={paths.control.settings()} variant="secondary" className="mt-2">
              <HardDriveDownload size={16} aria-hidden="true" />
              Open site settings
            </Button>
          </Card>
        </div>
      )}
    </>
  )
}

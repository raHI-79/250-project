import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Activity, AlertTriangle, Database, HardDrive, Server } from 'lucide-react'
import {
  Alert,
  Card,
  CardHeader,
  DataTable,
  EmptyState,
  PageHeader,
  SkeletonCard,
  SkeletonTable,
} from '../../components/ui'
import Sparkline from '../../components/charts/Sparkline'
import RangePicker from '../../components/control/RangePicker'
import { AXIS_INK, GRID, SERIES, legendStyle, tooltipStyle } from '../../components/charts/theme'
import api from '../../lib/api'
import { formatDateTime } from '../../lib/time'

const REFRESH_AFTER = 60000

const ROLE_LABELS = {
  student: 'Students',
  cr: 'CRs',
  teacher: 'Teachers',
  admin: 'Office admins',
  super_admin: 'Control room',
  unknown: 'Unknown',
}

const CLIENT_LABELS = { web: 'Website', android: 'Android app' }

const megabytes = (bytes) => {
  const value = Number(bytes || 0)
  if (value >= 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`
  if (value >= 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`
  if (value >= 1024) return `${(value / 1024).toFixed(0)} KB`
  return `${value} B`
}

const uptime = (seconds) => {
  const total = Number(seconds || 0)
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

const change = (value, previous) => {
  if (!previous) return null
  const shift = ((value - previous) / previous) * 100
  if (!Number.isFinite(shift)) return null
  const rounded = Math.round(shift)
  return { text: `${rounded > 0 ? '+' : ''}${rounded}%`, up: rounded >= 0 }
}

const share = (ok, failed) => {
  const total = Number(ok || 0) + Number(failed || 0)
  if (total === 0) return '0%'
  return `${Math.round((Number(failed || 0) / total) * 100)}%`
}

function Metric({ label, value, hint }) {
  return (
    <div className="rounded-tile bg-wash p-3" title={hint}>
      <p className="text-2xs font-semibold tracking-wide text-muted uppercase">{label}</p>
      <p className="font-display mt-1 text-section font-bold text-ink">{value}</p>
    </div>
  )
}

function KpiCard({ card, color }) {
  const shift = change(card.value, card.previous)
  return (
    <Card className="flex flex-col gap-2" hover={false}>
      <div className="flex items-start justify-between gap-2">
        <p className="text-2xs font-semibold tracking-wide text-muted uppercase" title={card.hint}>
          {card.label}
        </p>
        <Sparkline points={card.spark || []} color={color} label={`${card.label} trend`} />
      </div>
      <p className="font-display text-title font-bold text-ink">{card.value}</p>
      {shift ? (
        <p className={shift.up ? 'text-xs font-semibold text-success-600' : 'text-xs font-semibold text-danger-600'}>
          {shift.text} vs the period before
        </p>
      ) : (
        <p className="text-xs text-muted">{card.hint}</p>
      )}
    </Card>
  )
}

export default function ControlDashboardPage() {
  const [range, setRange] = useState('7d')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const query = useMemo(() => {
    if (range === 'custom') return { range, from, to }
    return { range }
  }, [range, from, to])
  const load = useCallback(
    async (quiet) => {
      if (range === 'custom' && (!from || !to)) return
      if (!quiet) setLoading(true)
      try {
        const result = await api.get('/api/super-admin/dashboard', { query })
        setData(result)
        setError('')
      } catch (failure) {
        setError(failure.message)
      } finally {
        setLoading(false)
      }
    },
    [query, range, from, to],
  )
  useEffect(() => {
    load(false)
  }, [load])
  useEffect(() => {
    const timer = window.setInterval(() => load(true), REFRESH_AFTER)
    return () => window.clearInterval(timer)
  }, [load])
  const overview = data?.overview
  const academic = data?.academic
  const system = data?.system
  const roles = (overview?.roles || []).map((row) => ({
    ...row,
    name: ROLE_LABELS[row.label] || row.label,
  }))
  const clients = (overview?.clients || []).map((row) => ({
    ...row,
    name: CLIENT_LABELS[row.label] || row.label,
  }))
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Dashboard"
        subtitle={
          data?.updated_at
            ? `Everything the portal did in this period · updated ${formatDateTime(data.updated_at)}`
            : 'Everything the portal did in this period'
        }
        tabs={
          <RangePicker
            value={range}
            onChange={setRange}
            from={from}
            to={to}
            onFrom={setFrom}
            onTo={setTo}
          />
        }
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {(data?.alerts || []).length > 0 && (
        <div className="mb-5 flex flex-col gap-2">
          {data.alerts.map((alert) => (
            <Alert key={alert.key} tone={alert.tone === 'error' ? 'error' : 'warning'}>
              <span className="inline-flex items-center gap-2">
                <AlertTriangle size={16} aria-hidden="true" />
                {alert.message}
              </span>
            </Alert>
          ))}
        </div>
      )}
      {loading && !data ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          <section>
            <h2 className="font-display mb-3 text-section font-bold text-ink">Overview</h2>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              {(overview?.cards || []).map((card, index) => (
                <KpiCard key={card.key} card={card} color={SERIES[index % SERIES.length]} />
              ))}
            </div>
          </section>
          <Card hover={false} flush>
            <CardHeader
              title="Visitors and sign-ins"
              subtitle="Unique visitors and successful sign-ins per day"
            />
            <div className="h-72 px-2 pb-4">
              {(overview?.series || []).length === 0 ? (
                <EmptyState icon={Activity} title="No traffic yet" text="Numbers appear once people open the portal." />
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={overview.series} margin={{ top: 8, right: 16, bottom: 0, left: 0 }}>
                    <CartesianGrid stroke={GRID} vertical={false} />
                    <XAxis
                      dataKey="day"
                      tick={{ fill: AXIS_INK, fontSize: 12 }}
                      tickLine={false}
                      axisLine={{ stroke: GRID }}
                      minTickGap={24}
                    />
                    <YAxis
                      tick={{ fill: AXIS_INK, fontSize: 12 }}
                      tickLine={false}
                      axisLine={false}
                      width={36}
                      allowDecimals={false}
                    />
                    <Tooltip {...tooltipStyle} />
                    <Legend wrapperStyle={legendStyle} />
                    <Line
                      type="monotone"
                      dataKey="visitors"
                      name="Visitors"
                      stroke={SERIES[0]}
                      strokeWidth={2}
                      dot={false}
                      activeDot={{ r: 4, strokeWidth: 2, stroke: 'var(--color-surface)' }}
                    />
                    <Line
                      type="monotone"
                      dataKey="sign_ins"
                      name="Sign-ins"
                      stroke={SERIES[1]}
                      strokeWidth={2}
                      dot={false}
                      activeDot={{ r: 4, strokeWidth: 2, stroke: 'var(--color-surface)' }}
                    />
                  </LineChart>
                </ResponsiveContainer>
              )}
            </div>
          </Card>
          <div className="grid gap-4 lg:grid-cols-2">
            <Card hover={false} flush>
              <CardHeader title="Sign-ins by role" subtitle="Who signed in during this period" />
              <div className="h-64 px-2 pb-4">
                {roles.length === 0 ? (
                  <EmptyState icon={Activity} title="No sign-ins yet" text="Nobody signed in during this period." />
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={roles} margin={{ top: 8, right: 16, bottom: 0, left: 0 }}>
                      <CartesianGrid stroke={GRID} vertical={false} />
                      <XAxis
                        dataKey="name"
                        tick={{ fill: AXIS_INK, fontSize: 12 }}
                        tickLine={false}
                        axisLine={{ stroke: GRID }}
                      />
                      <YAxis
                        tick={{ fill: AXIS_INK, fontSize: 12 }}
                        tickLine={false}
                        axisLine={false}
                        width={36}
                        allowDecimals={false}
                      />
                      <Tooltip {...tooltipStyle} />
                      <Bar
                        dataKey="total"
                        name="Sign-ins"
                        fill={SERIES[0]}
                        radius={[4, 4, 0, 0]}
                        maxBarSize={38}
                      />
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </div>
            </Card>
            <Card hover={false} flush>
              <CardHeader title="Website and app" subtitle="Where the page views came from" />
              <div className="h-64 px-2 pb-4">
                {clients.length === 0 ? (
                  <EmptyState icon={Activity} title="No page views yet" text="The beacon has nothing to show." />
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={clients}
                        dataKey="total"
                        nameKey="name"
                        innerRadius="58%"
                        outerRadius="82%"
                        paddingAngle={2}
                        stroke="var(--color-surface)"
                        strokeWidth={2}
                      >
                        {clients.map((entry, index) => (
                          <Cell key={entry.label} fill={SERIES[index % SERIES.length]} />
                        ))}
                      </Pie>
                      <Tooltip {...tooltipStyle} />
                      <Legend wrapperStyle={legendStyle} />
                    </PieChart>
                  </ResponsiveContainer>
                )}
              </div>
            </Card>
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <Card hover={false} flush>
              <CardHeader title="App versions in use" subtitle="Unique app visitors per version" />
              {(overview?.versions || []).length === 0 ? (
                <div className="px-4 pb-4">
                  <EmptyState icon={Activity} title="No app traffic" text="Nobody opened the app in this period." />
                </div>
              ) : (
                <DataTable>
                  <thead>
                    <tr className="border-b border-line bg-wash">
                      <th className="px-4 py-3 font-semibold text-ink-soft">Version</th>
                      <th className="px-4 py-3 font-semibold text-ink-soft">Visitors</th>
                    </tr>
                  </thead>
                  <tbody>
                    {overview.versions.map((row) => (
                      <tr key={row.label} className="border-b border-line last:border-b-0">
                        <td className="px-4 py-3 text-ink">{row.label}</td>
                        <td className="px-4 py-3 text-ink-soft">{row.total}</td>
                      </tr>
                    ))}
                  </tbody>
                </DataTable>
              )}
            </Card>
            <Card hover={false} flush>
              <CardHeader title="Devices" subtitle="Unique visitors per device kind" />
              {(overview?.devices || []).length === 0 ? (
                <div className="px-4 pb-4">
                  <EmptyState icon={Activity} title="No devices yet" text="Numbers appear with the first visit." />
                </div>
              ) : (
                <DataTable>
                  <thead>
                    <tr className="border-b border-line bg-wash">
                      <th className="px-4 py-3 font-semibold text-ink-soft">Device</th>
                      <th className="px-4 py-3 font-semibold text-ink-soft">Visitors</th>
                    </tr>
                  </thead>
                  <tbody>
                    {overview.devices.map((row) => (
                      <tr key={row.label} className="border-b border-line last:border-b-0">
                        <td className="px-4 py-3 text-ink">{row.label}</td>
                        <td className="px-4 py-3 text-ink-soft">{row.total}</td>
                      </tr>
                    ))}
                  </tbody>
                </DataTable>
              )}
            </Card>
          </div>
          <Card hover={false} flush>
            <CardHeader title="Top pages" subtitle="The pages people opened most" />
            {(overview?.pages || []).length === 0 ? (
              <div className="px-4 pb-4">
                <EmptyState icon={Activity} title="No pages yet" text="Numbers appear with the first visit." />
              </div>
            ) : (
              <DataTable>
                <thead>
                  <tr className="border-b border-line bg-wash">
                    <th className="px-4 py-3 font-semibold text-ink-soft">Path</th>
                    <th className="px-4 py-3 font-semibold text-ink-soft">Views</th>
                    <th className="px-4 py-3 font-semibold text-ink-soft">Visitors</th>
                  </tr>
                </thead>
                <tbody>
                  {overview.pages.map((row) => (
                    <tr key={row.path} className="border-b border-line last:border-b-0">
                      <td className="px-4 py-3 wrap-long text-ink">{row.path}</td>
                      <td className="px-4 py-3 text-ink-soft">{row.views}</td>
                      <td className="px-4 py-3 text-ink-soft">{row.visitors}</td>
                    </tr>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>
          <section>
            <h2 className="font-display mb-3 text-section font-bold text-ink">Academic</h2>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <Metric
                label="Below 50%"
                value={academic?.below_half ?? 0}
                hint="Students whose attendance in this period is under half."
              />
              <Metric
                label="Phone requests"
                value={academic?.phone_pending ?? 0}
                hint="Phone change requests still waiting for a teacher."
              />
              <Metric
                label="Notices"
                value={academic?.notices ?? 0}
                hint="Notices posted in this period."
              />
              <Metric label="Polls" value={academic?.polls ?? 0} hint="Polls opened in this period." />
              <Metric
                label="Messages"
                value={academic?.messages ?? 0}
                hint="Messages sent in this period."
              />
              <Metric
                label="Classes held"
                value={academic?.classes_held ?? 0}
                hint="Confirmed or scheduled classes in this period."
              />
              <Metric
                label="Cancelled"
                value={academic?.cancelled ?? 0}
                hint="Classes cancelled in this period."
              />
              <Metric
                label="Rescheduled"
                value={academic?.rescheduled ?? 0}
                hint="Classes moved to another slot in this period."
              />
            </div>
          </section>
          <div className="grid gap-4 lg:grid-cols-2">
            <Card hover={false} flush>
              <CardHeader
                title="Attendance per batch"
                subtitle="Average presence across the sessions in this period"
              />
              {(academic?.attendance || []).length === 0 ? (
                <div className="px-4 pb-4">
                  <EmptyState icon={Activity} title="No sessions yet" text="No attendance was taken in this period." />
                </div>
              ) : (
                <DataTable>
                  <thead>
                    <tr className="border-b border-line bg-wash">
                      <th className="px-4 py-3 font-semibold text-ink-soft">Batch</th>
                      <th className="px-4 py-3 font-semibold text-ink-soft">Section</th>
                      <th className="px-4 py-3 font-semibold text-ink-soft">Average</th>
                      <th className="px-4 py-3 font-semibold text-ink-soft">Classes</th>
                    </tr>
                  </thead>
                  <tbody>
                    {academic.attendance.map((row) => (
                      <tr key={`${row.batch}-${row.section}`} className="border-b border-line last:border-b-0">
                        <td className="px-4 py-3 text-ink">{row.batch}</td>
                        <td className="px-4 py-3 text-ink-soft">{row.section}</td>
                        <td className="px-4 py-3 font-semibold text-ink">{Math.round(row.average)}%</td>
                        <td className="px-4 py-3 text-ink-soft">{row.classes}</td>
                      </tr>
                    ))}
                  </tbody>
                </DataTable>
              )}
            </Card>
            <Card hover={false} flush>
              <CardHeader
                title="Tomorrow's undecided classes"
                subtitle="Teachers who have not answered yet"
              />
              {(academic?.undecided || []).length === 0 ? (
                <div className="px-4 pb-4">
                  <EmptyState
                    icon={Activity}
                    title="Every class is decided"
                    text="No class is waiting for a teacher tomorrow."
                  />
                </div>
              ) : (
                <DataTable>
                  <thead>
                    <tr className="border-b border-line bg-wash">
                      <th className="px-4 py-3 font-semibold text-ink-soft">Teacher</th>
                      <th className="px-4 py-3 font-semibold text-ink-soft">Classes</th>
                    </tr>
                  </thead>
                  <tbody>
                    {academic.undecided.map((row) => (
                      <tr key={row.label} className="border-b border-line last:border-b-0">
                        <td className="px-4 py-3 text-ink">{row.label}</td>
                        <td className="px-4 py-3 text-ink-soft">{row.total}</td>
                      </tr>
                    ))}
                  </tbody>
                </DataTable>
              )}
            </Card>
          </div>
          <Card hover={false} flush>
            <CardHeader
              title="Classes per course"
              subtitle="Held, cancelled and rescheduled in this period"
            />
            {(academic?.courses || []).length === 0 ? (
              <div className="px-4 pb-4">
                <EmptyState icon={Activity} title="No classes yet" text="No class was planned in this period." />
              </div>
            ) : (
              <div className="h-80 px-2 pb-4">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={academic.courses} margin={{ top: 8, right: 16, bottom: 0, left: 0 }}>
                    <CartesianGrid stroke={GRID} vertical={false} />
                    <XAxis
                      dataKey="code"
                      tick={{ fill: AXIS_INK, fontSize: 12 }}
                      tickLine={false}
                      axisLine={{ stroke: GRID }}
                      interval={0}
                      angle={-30}
                      textAnchor="end"
                      height={60}
                    />
                    <YAxis
                      tick={{ fill: AXIS_INK, fontSize: 12 }}
                      tickLine={false}
                      axisLine={false}
                      width={36}
                      allowDecimals={false}
                    />
                    <Tooltip {...tooltipStyle} />
                    <Legend wrapperStyle={legendStyle} />
                    <Bar dataKey="held" name="Held" fill={SERIES[0]} radius={[4, 4, 0, 0]} maxBarSize={22} />
                    <Bar
                      dataKey="cancelled"
                      name="Cancelled"
                      fill={SERIES[1]}
                      radius={[4, 4, 0, 0]}
                      maxBarSize={22}
                    />
                    <Bar
                      dataKey="rescheduled"
                      name="Rescheduled"
                      fill={SERIES[2]}
                      radius={[4, 4, 0, 0]}
                      maxBarSize={22}
                    />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            )}
          </Card>
          <section>
            <h2 className="font-display mb-3 text-section font-bold text-ink">System health</h2>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <Metric
                label="Version"
                value={system?.version || '—'}
                hint={system?.commit ? `Commit ${system.commit}` : 'The running release'}
              />
              <Metric
                label="Uptime"
                value={uptime(system?.uptime_seconds)}
                hint="How long this server process has been running."
              />
              <Metric
                label="Database"
                value={megabytes(system?.database_bytes)}
                hint="Size of the Postgres database."
              />
              <Metric
                label="Files"
                value={megabytes(system?.file_bytes)}
                hint="Size of the uploaded photos and attachments."
              />
              <Metric
                label="4xx answers"
                value={system?.client_4xx ?? 0}
                hint="Requests refused in this period."
              />
              <Metric
                label="5xx answers"
                value={system?.server_5xx ?? 0}
                hint="Server errors in this period."
              />
              <Metric
                label="Email failures"
                value={share(system?.emails_sent, system?.emails_failed)}
                hint={`${system?.emails_sent ?? 0} sent, ${system?.emails_failed ?? 0} failed.`}
              />
              <Metric
                label="Push failures"
                value={share(
                  (system?.push_sent ?? 0) + (system?.fcm_sent ?? 0),
                  (system?.push_failed ?? 0) + (system?.fcm_failed ?? 0),
                )}
                hint={`${system?.push_tokens ?? 0} web subscriptions, ${system?.fcm_tokens ?? 0} app tokens.`}
              />
              <Metric
                label="Last backup"
                value={system?.backup_at ? formatDateTime(system.backup_at) : 'Never'}
                hint="Recorded by the backup workflow."
              />
            </div>
          </section>
          <Card hover={false} flush>
            <CardHeader title="Latest server errors" subtitle="The five newest 5xx answers" />
            {(system?.server_errors || []).length === 0 ? (
              <div className="px-4 pb-4">
                <EmptyState icon={Server} title="No server errors" text="Nothing broke recently." />
              </div>
            ) : (
              <DataTable>
                <thead>
                  <tr className="border-b border-line bg-wash">
                    <th className="px-4 py-3 font-semibold text-ink-soft">When</th>
                    <th className="px-4 py-3 font-semibold text-ink-soft">Request</th>
                  </tr>
                </thead>
                <tbody>
                  {system.server_errors.map((row) => (
                    <tr key={`${row.at}-${row.path}`} className="border-b border-line last:border-b-0">
                      <td className="px-4 py-3 whitespace-nowrap text-ink-soft">{formatDateTime(row.at)}</td>
                      <td className="px-4 py-3 wrap-long text-ink">{row.path}</td>
                    </tr>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>
          {loading && (
            <div className="flex items-center gap-2 text-sm text-muted">
              <Database size={14} aria-hidden="true" />
              Refreshing the numbers…
            </div>
          )}
          <p className="flex items-center gap-2 text-xs text-muted">
            <HardDrive size={14} aria-hidden="true" />
            The dashboard refreshes itself every minute.
          </p>
        </div>
      )}
    </>
  )
}

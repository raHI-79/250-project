import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Download, ScrollText } from 'lucide-react'
import {
  Alert,
  Button,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SkeletonTable,
  StatusBadge,
} from '../../components/ui'
import api from '../../lib/api'
import { useAuth } from '../../context/auth'
import { ACTIVITY_ACTIONS, ACTIVITY_ROLES, activityLabel } from '../../lib/options'
import { ROLE_SUPER_ADMIN, paths } from '../../lib/paths'
import { formatDateTime } from '../../lib/time'

const HEADINGS = ['When', 'Person', 'Action', 'What', 'Where', '']

const PAGE_SIZE = 50

const tone = (action) => {
  if (action === 'sign_in_failed') return 'rejected'
  if (action === 'delete' || action.startsWith('remove')) return 'removed'
  if (action === 'sign_in' || action === 'create') return 'active'
  return 'pending'
}

export default function ActivityLogPage() {
  const { role } = useAuth()
  const watching = role === ROLE_SUPER_ADMIN
  const [person, setPerson] = useState('')
  const [personRole, setPersonRole] = useState('')
  const [action, setAction] = useState('')
  const [client, setClient] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [search, setSearch] = useState('')
  const [items, setItems] = useState([])
  const [cursor, setCursor] = useState(0)
  const [more, setMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState('')
  const [people, setPeople] = useState([])
  const sentinel = useRef(null)
  const filters = useMemo(
    () => ({ person, role: personRole, action, client, from, to, search }),
    [person, personRole, action, client, from, to, search],
  )
  useEffect(() => {
    api
      .get('/api/admin/activity/people')
      .then((data) => setPeople(data.items || []))
      .catch(() => setPeople([]))
  }, [])
  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await api.get('/api/admin/activity', {
        query: { ...filters, limit: PAGE_SIZE },
      })
      const rows = data.items || []
      setItems(rows)
      setCursor(data.cursor || 0)
      setMore(rows.length === PAGE_SIZE)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }, [filters])
  useEffect(() => {
    load()
  }, [load])
  const loadMore = useCallback(async () => {
    if (!more || adding || !cursor) return
    setAdding(true)
    try {
      const data = await api.get('/api/admin/activity', {
        query: { ...filters, limit: PAGE_SIZE, before: cursor },
      })
      const rows = data.items || []
      setItems((current) => [...current, ...rows])
      setCursor(data.cursor || 0)
      setMore(rows.length === PAGE_SIZE)
    } catch (failure) {
      setError(failure.message)
      setMore(false)
    } finally {
      setAdding(false)
    }
  }, [filters, cursor, more, adding])
  useEffect(() => {
    const node = sentinel.current
    if (!node || typeof IntersectionObserver === 'undefined') return undefined
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) loadMore()
      },
      { rootMargin: '200px' },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [loadMore])
  const exportCSV = async () => {
    try {
      const text = await api.get('/api/admin/activity/export', { query: filters })
      const blob = new Blob([typeof text === 'string' ? text : JSON.stringify(text)], {
        type: 'text/csv;charset=utf-8',
      })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'activity-log.csv'
      link.click()
      URL.revokeObjectURL(url)
    } catch (failure) {
      setError(failure.message)
    }
  }
  return (
    <>
      <PageHeader
        eyebrow={watching ? 'Control room' : 'Department office'}
        title="Activity log"
        subtitle="Every sign-in, change and removal, newest first"
        actions={
          <Button variant="secondary" onClick={exportCSV}>
            <Download size={16} aria-hidden="true" />
            Export CSV
          </Button>
        }
      />
      <div className="mb-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Select label="Person" value={person} onChange={(event) => setPerson(event.target.value)}>
          <option value="">Everyone</option>
          {people.map((entry) => (
            <option key={`${entry.id}-${entry.label}`} value={entry.id}>
              {entry.label} ({entry.total})
            </option>
          ))}
        </Select>
        <Select label="Role" value={personRole} onChange={(event) => setPersonRole(event.target.value)}>
          <option value="">Every role</option>
          {ACTIVITY_ROLES.map((entry) => (
            <option key={entry.value} value={entry.value}>
              {entry.label}
            </option>
          ))}
        </Select>
        <Select label="Action" value={action} onChange={(event) => setAction(event.target.value)}>
          <option value="">Every action</option>
          {ACTIVITY_ACTIONS.map((entry) => (
            <option key={entry.value} value={entry.value}>
              {entry.label}
            </option>
          ))}
        </Select>
        <Select label="Client" value={client} onChange={(event) => setClient(event.target.value)}>
          <option value="">Website and app</option>
          <option value="web">Website</option>
          <option value="android">Android app</option>
        </Select>
        <Input label="From" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        <Input label="To" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
        <Input
          label="Search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          className="xl:col-span-2"
        />
      </div>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={6} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={ScrollText}
          title="Nothing logged yet"
          text="Signing in or changing something shows up here."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              {HEADINGS.map((heading, index) => (
                <th key={heading || index} className="px-4 py-3 font-semibold text-ink-soft">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {formatDateTime(item.created_at)}
                </td>
                <td className="px-4 py-3">
                  <p className="font-semibold text-ink">{item.actor}</p>
                  <p className="text-xs text-muted">{activityLabel(item.role)}</p>
                </td>
                <td className="px-4 py-3">
                  <StatusBadge status={tone(item.action)} label={activityLabel(item.action)} />
                </td>
                <td className="px-4 py-3 wrap-long text-ink">{item.summary}</td>
                <td className="px-4 py-3 text-ink-soft">
                  {item.client === 'android' ? 'App' : 'Website'}
                </td>
                <td className="px-4 py-3 text-right">
                  {watching && item.actor_id > 0 && (
                    <Link
                      to={paths.control.monitor(item.actor_id)}
                      className="text-sm font-semibold text-brand-600 hover:underline"
                    >
                      Monitor this admin
                    </Link>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      <div ref={sentinel} aria-hidden="true" className="h-8" />
      {adding && <p className="pb-4 text-center text-sm text-muted">Loading more…</p>}
    </>
  )
}

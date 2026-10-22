import { useState } from 'react'
import { Link } from 'react-router-dom'
import { GraduationCap } from 'lucide-react'
import {
  Alert,
  Avatar,
  Card,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  SkeletonTable,
} from '../../components/ui'
import { useDebounced, useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'

const HEADINGS = ['Name', 'Designation', 'Office', 'Phone', 'Email']

export default function TeachersPage() {
  const [search, setSearch] = useState('')
  const settled = useDebounced(search)
  const { data, loading, error } = useResource(
    '/api/teachers-info',
    settled.trim() ? { search: settled.trim() } : null,
  )
  const items = data?.items || []
  return (
    <>
      <PageHeader eyebrow="Department directory" title="Teachers" subtitle="Contact details of the department teachers" />
      <Card hover={false} className="mb-5">
        <Input
          label="Search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Teacher name"
          className="max-w-sm"
        />
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={5} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={GraduationCap}
          title="No teachers found"
          text="No active teacher matches this search."
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
            {items.map((item) => (
              <tr key={item.email} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3 text-ink">
                  <span className="flex items-center gap-2.5">
                    <Avatar src={item.photo_url} name={item.name} size="sm" />
                    <Link
                      to={paths.person(item.user_id)}
                      className="font-medium text-ink hover:text-brand-600 hover:underline"
                    >
                      {item.name}
                    </Link>
                  </span>
                </td>
                <td className="px-4 py-3 text-ink-soft">{item.designation || '—'}</td>
                <td className="px-4 py-3 text-ink-soft">{item.office_room || '—'}</td>
                <td className="px-4 py-3">
                  <a href={`tel:${item.phone}`} className="text-brand-600 hover:underline">
                    {item.phone}
                  </a>
                </td>
                <td className="px-4 py-3">
                  <a href={`mailto:${item.email}`} className="text-brand-600 hover:underline">
                    {item.email}
                  </a>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

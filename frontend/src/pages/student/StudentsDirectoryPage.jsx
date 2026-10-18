import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Users } from 'lucide-react'
import {
  Alert,
  Avatar,
  Card,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SkeletonTable,
  StatusBadge,
} from '../../components/ui'
import { useDebounced, useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { BLOOD_GROUPS } from '../../lib/options'

const HEADINGS = ['Name', 'Reg No', 'Batch', 'Section', 'Phone', 'Email', 'Blood Group']

export default function StudentsDirectoryPage() {
  const [batch, setBatch] = useState('')
  const [bloodGroup, setBloodGroup] = useState('')
  const [search, setSearch] = useState('')
  const settled = useDebounced(search)
  const batches = useResource('/api/lookups/batches')
  const query = {}
  if (batch) query.batch = batch
  if (bloodGroup) query.blood_group = bloodGroup
  if (settled.trim()) query.search = settled.trim()
  const { data, loading, error } = useResource('/api/students/directory', query)
  const items = data?.items || []
  return (
    <>
      <PageHeader title="Students" subtitle="Every active student of the department" />
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-3">
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            <option value="">Every batch</option>
            {(batches.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <Select
            label="Blood group"
            value={bloodGroup}
            onChange={(event) => setBloodGroup(event.target.value)}
          >
            <option value="">Every blood group</option>
            {BLOOD_GROUPS.map((group) => (
              <option key={group} value={group}>
                {group}
              </option>
            ))}
          </Select>
          <Input
            label="Search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Name or registration number"
          />
        </div>
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={7} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Users}
          title="No students found"
          text="No active student matches these filters."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-brand-100 bg-brand-50/60">
              {HEADINGS.map((heading) => (
                <th key={heading} className="px-4 py-3 font-semibold text-brand-700">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.reg_no} className="border-b border-brand-50 last:border-b-0">
                <td className="px-4 py-3 text-brand-900">
                  <span className="flex items-center gap-2.5">
                    <Avatar src={item.photo_url} name={item.full_name} size="sm" />
                    <Link
                      to={paths.person(item.user_id)}
                      className="font-medium text-brand-900 hover:text-brand-600 hover:underline"
                    >
                      {item.full_name}
                    </Link>
                    {item.is_cr && <StatusBadge status="approved" label="CR" />}
                  </span>
                </td>
                <td className="px-4 py-3 tabular-nums text-brand-700">{item.reg_no}</td>
                <td className="px-4 py-3 text-brand-700">{item.batch}</td>
                <td className="px-4 py-3 text-brand-700">{item.section}</td>
                <td className="px-4 py-3 text-brand-700">{item.phone}</td>
                <td className="px-4 py-3 text-brand-700">{item.email}</td>
                <td className="px-4 py-3 text-brand-700">{item.blood_group}</td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

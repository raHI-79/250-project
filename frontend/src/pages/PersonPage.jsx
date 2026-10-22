import { Link, useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Mail, Phone } from 'lucide-react'
import IdentityCard from '../components/IdentityCard'
import {
  Alert,
  Button,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../components/ui'
import { useResource } from '../lib/hooks'
import { useAuth } from '../context/auth'
import { accountRoleLabel } from '../lib/account'
import { homeFor } from '../lib/paths'

export default function PersonPage() {
  const { userId } = useParams()
  const navigate = useNavigate()
  const { role } = useAuth()
  const { data, loading, error } = useResource(`/api/people/${userId}`)
  const isTeacher = data?.role === 'teacher'
  const rows = isTeacher
    ? [
        { label: 'Designation', value: data?.designation },
        { label: 'Office room', value: data?.office_room },
      ]
    : [
        { label: 'Batch', value: data?.batch },
        { label: 'Section', value: data?.section },
      ]
  return (
    <>
      <PageHeader
        eyebrow="Department directory"
        title="Profile card"
        subtitle="Contact details shared inside the department"
        actions={
          <Button variant="secondary" size="sm" onClick={() => navigate(-1)}>
            <ArrowLeft size={15} />
            Back
          </Button>
        }
      />
      {loading ? (
        <SkeletonCard className="max-w-sm" />
      ) : error ? (
        <EmptyState
          title="Profile not available"
          text={error}
          action={
            <Link to={homeFor(role)} className="text-sm text-brand-600 hover:underline">
              Back to your panel
            </Link>
          }
        />
      ) : (
        <div className="max-w-sm">
          <IdentityCard
            name={data?.name}
            photoUrl={data?.photo_url}
            role={accountRoleLabel(data?.role)}
            chips={data?.role === 'cr' ? [<StatusBadge key="cr" status="approved" label="CR" />] : []}
            rows={[
              ...rows,
              { label: 'Blood group', value: data?.blood_group },
              {
                label: 'Email',
                value: (
                  <a
                    href={`mailto:${data?.email}`}
                    className="inline-flex items-center gap-1.5 text-brand-600 hover:underline"
                  >
                    <Mail size={13} />
                    {data?.email}
                  </a>
                ),
              },
              {
                label: 'Phone',
                value: (
                  <a
                    href={`tel:${data?.phone}`}
                    className="inline-flex items-center gap-1.5 text-brand-600 hover:underline"
                  >
                    <Phone size={13} />
                    {data?.phone}
                  </a>
                ),
              },
            ]}
          />
          <Alert tone="info" className="mt-4">
            Use these details for class and exam coordination only.
          </Alert>
        </div>
      )}
    </>
  )
}

import { Link, useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Mail, Phone } from 'lucide-react'
import {
  Alert,
  Avatar,
  Button,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../components/ui'
import { useResource } from '../lib/hooks'
import { useAuth } from '../context/auth'
import { accountRoleLabel } from '../lib/account'
import { homeFor } from '../lib/paths'

function Row({ label, children }) {
  if (!children) return null
  return (
    <div className="flex flex-col gap-0.5 border-b border-brand-50 py-2.5 last:border-b-0">
      <span className="text-xs font-medium tracking-wide text-brand-400 uppercase">{label}</span>
      <span className="text-sm text-brand-900">{children}</span>
    </div>
  )
}

export default function PersonPage() {
  const { userId } = useParams()
  const navigate = useNavigate()
  const { role } = useAuth()
  const { data, loading, error } = useResource(`/api/people/${userId}`)
  const isTeacher = data?.role === 'teacher'
  return (
    <>
      <PageHeader
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
        <SkeletonCard className="max-w-xl" />
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
        <Card hover={false} className="max-w-xl">
          <div className="flex items-center gap-4">
            <Avatar src={data?.photo_url} name={data?.name} size="lg" />
            <div className="min-w-0">
              <p className="font-display truncate text-lg font-semibold text-brand-900">
                {data?.name}
              </p>
              <div className="mt-1 flex flex-wrap items-center gap-2">
                <span className="text-xs font-medium tracking-wide text-brand-500 uppercase">
                  {accountRoleLabel(data?.role)}
                </span>
                {data?.role === 'cr' && <StatusBadge status="approved" label="CR" />}
              </div>
            </div>
          </div>
          <div className="mt-4 border-t border-brand-100 pt-2">
            {isTeacher ? (
              <>
                <Row label="Designation">{data?.designation}</Row>
                <Row label="Office room">{data?.office_room}</Row>
              </>
            ) : (
              <>
                <Row label="Batch">{data?.batch}</Row>
                <Row label="Section">{data?.section}</Row>
              </>
            )}
            <Row label="Blood group">{data?.blood_group}</Row>
            <Row label="Email">
              <a
                href={`mailto:${data?.email}`}
                className="inline-flex items-center gap-1.5 text-brand-600 hover:underline"
              >
                <Mail size={14} />
                {data?.email}
              </a>
            </Row>
            <Row label="Phone">
              <a
                href={`tel:${data?.phone}`}
                className="inline-flex items-center gap-1.5 text-brand-600 hover:underline"
              >
                <Phone size={14} />
                {data?.phone}
              </a>
            </Row>
          </div>
          <Alert tone="info" className="mt-4">
            Use these details for class and exam coordination only.
          </Alert>
        </Card>
      )}
    </>
  )
}

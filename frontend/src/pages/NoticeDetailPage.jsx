import { useEffect, useRef } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Megaphone } from 'lucide-react'
import NoticeCard from '../components/NoticeCard'
import { Alert, Button, EmptyState, PageHeader, SkeletonCard } from '../components/ui'
import { BADGE_NOTICES, useSectionSeen } from '../lib/badges'
import { useResource } from '../lib/hooks'
import paths from '../lib/paths'

const GONE = 404

export default function NoticeDetailPage() {
  useSectionSeen(BADGE_NOTICES)
  const { id } = useParams()
  const valid = /^\d+$/.test(id || '')
  const { data, loading, error, status } = useResource(valid ? `/api/students/notice/${id}` : '')
  const card = useRef(null)
  const found = Boolean(data?.id)
  useEffect(() => {
    if (!found || !card.current) return
    card.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [found, id])
  const gone = !valid || status === GONE
  const all = (
    <Button as={Link} to={paths.student.notices()} variant="secondary" size="sm">
      <ArrowLeft size={15} aria-hidden="true" />
      All notices
    </Button>
  )
  return (
    <>
      <PageHeader
        eyebrow="Notice board"
        title="Notice"
        subtitle="One notice from the department, your teachers or the class representatives"
      />
      {loading ? (
        <SkeletonCard />
      ) : gone ? (
        <EmptyState
          icon={Megaphone}
          title="This notice is no longer available"
          text="It was removed, or it was never meant for your batch."
          action={all}
        />
      ) : found ? (
        <div className="flex flex-col gap-4">
          <NoticeCard
            item={data}
            cardRef={card}
            highlighted
            title={
              <h2 className="font-display wrap-long text-lg font-bold text-ink">{data.title}</h2>
            }
          />
          <div>{all}</div>
        </div>
      ) : (
        <>
          <Alert tone="error" className="mb-5">
            {error || 'This notice could not be loaded.'}
          </Alert>
          {all}
        </>
      )}
    </>
  )
}

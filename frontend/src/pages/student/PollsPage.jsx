import { useState } from 'react'
import { Check, Vote } from 'lucide-react'
import {
  Alert,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  useToast,
} from '../../components/ui'
import PollResults from '../../components/PollResults'
import api from '../../lib/api'
import { useHighlight, useResource } from '../../lib/hooks'
import cx from '../../lib/cx'
import { formatDateTime } from '../../lib/time'

function PollCard({ poll, busy, onVote, flash }) {
  const voted = poll.my_option_id !== null && poll.my_option_id !== undefined
  const showResults = voted || poll.is_closed
  return (
    <Card
      hover={false}
      className={cx('animate-fade-in-up', flash && 'border-accent-300 bg-accent-50/60')}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <p className="font-display text-lg font-bold text-ink">{poll.question}</p>
        <StatusBadge
          status={poll.is_closed ? 'cancelled' : 'active'}
          label={poll.is_closed ? 'Closed' : 'Open'}
        />
      </div>
      <p className="mt-1 text-xs text-muted">
        {poll.created_by_name} · {formatDateTime(poll.created_at)} · {poll.total_votes} vote
        {poll.total_votes === 1 ? '' : 's'}
      </p>
      {showResults ? (
        <PollResults
          options={poll.options}
          total={poll.total_votes}
          myOptionId={poll.my_option_id}
        />
      ) : (
        <ul className="mt-4 flex flex-col gap-2">
          {poll.options.map((option) => (
            <li key={option.id}>
              <button
                type="button"
                disabled={busy}
                onClick={() => onVote(poll.id, option.id)}
                className="flex w-full items-center justify-between gap-3 rounded-md border border-line-strong bg-surface px-4 py-2.5 text-left text-sm text-ink transition-colors hover:border-line-strong hover:bg-wash disabled:cursor-not-allowed disabled:text-faint"
              >
                {option.text}
                <Check size={16} className="text-faint" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}

export default function PollsPage() {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/students/polls')
  const highlight = useHighlight(Boolean(data))
  const [busy, setBusy] = useState(0)
  const items = data?.items || []
  const onVote = async (pollId, optionId) => {
    setBusy(pollId)
    try {
      await api.post(`/api/students/polls/${pollId}/vote`, { option_id: optionId })
      toast.success('Your vote is recorded.')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  return (
    <>
      <PageHeader eyebrow="Your batch" title="Polls" subtitle="Polls your class representatives opened" />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Vote}
          title="No polls"
          text="Nothing is open for your batch and section right now. A class representative opens a poll when the batch has to pick a date or a time."
        />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((poll) => (
            <div key={poll.id} ref={highlight.register(poll.id)}>
              <PollCard
                poll={poll}
                busy={busy === poll.id}
                onVote={onVote}
                flash={highlight.highlighted(poll.id)}
              />
            </div>
          ))}
        </div>
      )}
    </>
  )
}

import { useState } from 'react'
import { Check, Vote } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  useToast,
  PullToRefresh,
} from '../../components/ui'
import PollResults from '../../components/PollResults'
import api from '../../lib/api'
import { useHighlight, useResource } from '../../lib/hooks'
import cx from '../../lib/cx'
import { formatDateTime } from '../../lib/time'
import { BADGE_POLLS, useSectionSeen } from '../../lib/badges'

const limitLine = (max) => (max > 1 ? `Pick up to ${max} options` : 'Pick one option')

function PollCard({ poll, busy, onVote, flash }) {
  const [picked, setPicked] = useState([])
  const mine = poll.my_option_ids || []
  const voted = mine.length > 0
  const showResults = voted || poll.is_closed
  const max = poll.max_choices || 1
  const toggle = (optionId) => {
    setPicked((current) => {
      if (current.includes(optionId)) return current.filter((entry) => entry !== optionId)
      if (current.length >= max) return max === 1 ? [optionId] : current
      return [...current, optionId]
    })
  }
  return (
    <Card hover={false} className={cx('animate-fade-in-up', flash && 'animate-flash border-gold-300')}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <p className="font-display wrap-long text-lg font-bold text-ink">{poll.question}</p>
        <StatusBadge
          status={poll.is_closed ? 'cancelled' : 'active'}
          label={poll.is_closed ? 'Closed' : 'Open'}
        />
      </div>
      <p className="mt-1 text-xs text-muted">
        {poll.created_by_name} · {formatDateTime(poll.created_at)} · {poll.total_voters} voter
        {poll.total_voters === 1 ? '' : 's'}
      </p>
      {showResults ? (
        <>
          <PollResults options={poll.options} total={poll.total_voters} myOptionIds={mine} />
          <p className="mt-3 text-xs text-muted">
            Only the counts are kept. Nobody, not even your class representative, sees who picked
            what.
          </p>
        </>
      ) : (
        <>
          <p className="mt-3 label-caps">{limitLine(max)}</p>
          <ul className="mt-2 flex flex-col gap-2">
            {poll.options.map((option) => {
              const chosen = picked.includes(option.id)
              return (
                <li key={option.id}>
                  <button
                    type="button"
                    disabled={busy}
                    aria-pressed={chosen}
                    onClick={() => toggle(option.id)}
                    className={cx(
                      'tap-target flex w-full items-center justify-between gap-3 rounded-xl border px-4 py-2.5 text-left text-sm transition-colors disabled:cursor-not-allowed',
                      chosen
                        ? 'border-brand-600 bg-brand-50 font-semibold text-ink'
                        : 'border-line-strong bg-surface text-ink hover:bg-wash',
                    )}
                  >
                    <span className="wrap-long">{option.text}</span>
                    <Check size={16} className={chosen ? 'text-brand-600' : 'text-faint'} />
                  </button>
                </li>
              )
            })}
          </ul>
          <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
            <p className="text-xs text-muted">
              {picked.length} of {max} picked
            </p>
            <Button
              loading={busy}
              disabled={picked.length === 0}
              onClick={() => onVote(poll.id, picked)}
            >
              Send my vote
            </Button>
          </div>
        </>
      )}
    </Card>
  )
}

export default function PollsPage() {
  useSectionSeen(BADGE_POLLS)
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/students/polls')
  const highlight = useHighlight(Boolean(data))
  const [busy, setBusy] = useState(0)
  const items = data?.items || []
  const onVote = async (pollId, optionIds) => {
    setBusy(pollId)
    try {
      await api.post(`/api/students/polls/${pollId}/vote`, { option_ids: optionIds })
      toast.success('Your vote is recorded.')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  return (
    <PullToRefresh onRefresh={reload}>
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
    </PullToRefresh>
  )
}

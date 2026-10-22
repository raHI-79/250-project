import { useState } from 'react'
import { Plus, Vote, X } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Field,
  Input,
  PageHeader,
  Select,
  SkeletonCard,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import PollResults from '../../components/PollResults'
import api from '../../lib/api'
import { useHighlight, useResource } from '../../lib/hooks'
import cx from '../../lib/cx'
import { formatDateTime } from '../../lib/time'

const MIN_OPTIONS = 2

const MAX_OPTIONS = 10

const pollAudience = (poll) => {
  if (poll.scope === 'all_batches') return 'Every batch'
  if (poll.scope === 'batch' || poll.section === 'ALL') return `${poll.batch} · whole batch`
  return `${poll.batch} · section ${poll.section}`
}

export default function CRPollsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [scope, setScope] = useState('section')
  const [maxChoices, setMaxChoices] = useState(1)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const [closing, setClosing] = useState(0)
  const polls = useResource('/api/cr/polls')
  const highlight = useHighlight(Boolean(polls.data))
  const setOption = (index) => (event) => {
    const { value } = event.target
    setOptions((current) => current.map((entry, position) => (position === index ? value : entry)))
  }
  const addOption = () => {
    setOptions((current) => (current.length < MAX_OPTIONS ? [...current, ''] : current))
  }
  const removeOption = (index) => {
    setOptions((current) =>
      current.length > MIN_OPTIONS ? current.filter((entry, position) => position !== index) : current,
    )
  }
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    const cleaned = options.map((entry) => entry.trim()).filter(Boolean)
    if (!question.trim()) {
      setError('Write the question first.')
      return
    }
    if (cleaned.length < MIN_OPTIONS) {
      setError(`A poll needs at least ${MIN_OPTIONS} options.`)
      return
    }
    if (new Set(cleaned.map((entry) => entry.toLowerCase())).size !== cleaned.length) {
      setError('Every option must be different.')
      return
    }
    setSending(true)
    try {
      await api.post('/api/cr/polls', {
        question: question.trim(),
        options: cleaned,
        scope,
        max_choices: Math.min(maxChoices, cleaned.length),
      })
      toast.success('Poll created')
      setQuestion('')
      setOptions(['', ''])
      setMaxChoices(1)
      polls.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSending(false)
    }
  }
  const close = async (poll) => {
    const ok = await confirm({
      title: 'Close this poll?',
      message: `"${poll.question}" stops accepting votes.`,
      confirmLabel: 'Close the poll',
    })
    if (!ok) return
    setClosing(poll.id)
    try {
      await api.post(`/api/cr/polls/${poll.id}/close`)
      toast.success('Poll closed')
      polls.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setClosing(0)
    }
  }
  const items = polls.data?.items || []
  const filledOptions = Math.max(
    MIN_OPTIONS,
    options.map((entry) => entry.trim()).filter(Boolean).length,
  )
  return (
    <>
      <PageHeader eyebrow="Ask your batch" title="Polls" subtitle="Ask your section, your batch or every batch a question" />
      <Card hover={false} className="mb-6">
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <Input
            label="Question"
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
            required
          />
          <Field label="Options" hint={`Between ${MIN_OPTIONS} and ${MAX_OPTIONS}`}>
            <div className="flex flex-col gap-2">
              {options.map((entry, index) => (
                <div key={index} className="flex items-center gap-2">
                  <input
                    value={entry}
                    onChange={setOption(index)}
                    placeholder={`Option ${index + 1}`}
                    className="w-full rounded-md border border-line-strong bg-surface px-3 py-2 text-sm text-ink transition-colors placeholder:text-faint focus:border-brand-500 focus:ring-2 focus:ring-brand-100 focus:outline-none"
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeOption(index)}
                    disabled={options.length <= MIN_OPTIONS}
                  >
                    <X size={15} />
                  </Button>
                </div>
              ))}
              <Button
                variant="secondary"
                size="sm"
                onClick={addOption}
                disabled={options.length >= MAX_OPTIONS}
                className="self-start"
              >
                <Plus size={15} />
                Add an option
              </Button>
            </div>
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Select label="Who votes" value={scope} onChange={(event) => setScope(event.target.value)}>
              <option value="section">My section</option>
              <option value="batch">Whole batch</option>
              <option value="all_batches">Every batch</option>
            </Select>
            <Select
              label="Each student may choose up to"
              value={String(Math.min(maxChoices, filledOptions))}
              onChange={(event) => setMaxChoices(Number(event.target.value))}
              hint="One vote per student, counted per option"
            >
              {Array.from({ length: filledOptions }, (_, index) => index + 1).map((count) => (
                <option key={count} value={count}>
                  {count === 1 ? '1 option' : `${count} options`}
                </option>
              ))}
            </Select>
          </div>
          <Button type="submit" loading={sending} className="w-full sm:w-auto sm:self-end">
            Create the poll
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-ink">My polls</p>
      {polls.error && <Alert tone="error" className="mb-5">{polls.error}</Alert>}
      {polls.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState icon={Vote} title="No polls yet" text="The polls you create appear here." />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((poll) => (
            <Card
              key={poll.id}
              ref={highlight.register(poll.id)}
              hover={false}
              className={cx(
                'animate-fade-in-up',
                highlight.highlighted(poll.id) && 'border-accent-300 bg-accent-50/60',
              )}
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <p className="font-display text-lg font-bold text-ink">
                  {poll.question}
                </p>
                <StatusBadge
                  status={poll.is_closed ? 'cancelled' : 'active'}
                  label={poll.is_closed ? 'Closed' : 'Open'}
                />
              </div>
              <p className="mt-1 text-xs text-muted">
                {pollAudience(poll)}{' '}
                · {formatDateTime(poll.created_at)} · {poll.total_voters} voter
                {poll.total_voters === 1 ? '' : 's'} ·{' '}
                {poll.max_choices > 1 ? `up to ${poll.max_choices} choices` : 'one choice'}
              </p>
              <PollResults
                options={poll.options}
                total={poll.total_voters}
                myOptionIds={poll.my_option_ids || []}
              />
              <p className="mt-3 text-xs text-muted">
                Votes are secret: you see the counts, never who picked what.
              </p>
              {!poll.is_closed && (
                <div className="mt-4 flex justify-end">
                  <Button
                    variant="danger"
                    size="sm"
                    loading={closing === poll.id}
                    onClick={() => close(poll)}
                  >
                    Close the poll
                  </Button>
                </div>
              )}
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

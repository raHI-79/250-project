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
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

const MIN_OPTIONS = 2

const MAX_OPTIONS = 10

export default function CRPollsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [scope, setScope] = useState('section')
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const [closing, setClosing] = useState(0)
  const polls = useResource('/api/cr/polls')
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
      await api.post('/api/cr/polls', { question: question.trim(), options: cleaned, scope })
      toast.success('Poll created')
      setQuestion('')
      setOptions(['', ''])
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
  return (
    <>
      <PageHeader title="Polls" subtitle="Ask your section or your whole batch a question" />
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
                    className="w-full rounded-lg border border-brand-200 bg-white px-3 py-2 text-sm text-brand-950 transition-colors placeholder:text-brand-300 focus:border-brand-500 focus:ring-2 focus:ring-brand-100 focus:outline-none"
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
          <Select label="Who votes" value={scope} onChange={(event) => setScope(event.target.value)}>
            <option value="section">My section</option>
            <option value="batch">Whole batch</option>
          </Select>
          <Button type="submit" loading={sending} className="w-full sm:w-auto sm:self-end">
            Create the poll
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-brand-800">My polls</p>
      {polls.error && <Alert tone="error" className="mb-5">{polls.error}</Alert>}
      {polls.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState icon={Vote} title="No polls yet" text="The polls you create appear here." />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((poll) => (
            <Card key={poll.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <p className="font-display text-base font-semibold text-brand-900">
                  {poll.question}
                </p>
                <StatusBadge
                  status={poll.is_closed ? 'cancelled' : 'active'}
                  label={poll.is_closed ? 'Closed' : 'Open'}
                />
              </div>
              <p className="mt-1 text-xs text-brand-500">
                {poll.section === 'ALL' ? `${poll.batch} · whole batch` : `${poll.batch} · section ${poll.section}`}{' '}
                · {formatDateTime(poll.created_at)} · {poll.total_votes} vote
                {poll.total_votes === 1 ? '' : 's'}
              </p>
              <PollResults options={poll.options} total={poll.total_votes} />
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

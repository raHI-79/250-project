import { useState } from 'react'
import { Plus, X } from 'lucide-react'
import { Button, Field, Input } from '../ui'
import SectionPicker from '../forms/SectionPicker'

const BATCH_SHAPE = /^[A-Z]{2,10}-\d{2}$/

export default function BatchPicker({ batches, onAdd, onDrop, lockedBatch }) {
  const [batch, setBatch] = useState(lockedBatch || '')
  const [section, setSection] = useState('')
  const [error, setError] = useState('')
  const add = () => {
    const wanted = batch.trim().toUpperCase()
    if (!BATCH_SHAPE.test(wanted)) {
      setError('Batch must look like CSE-23.')
      return
    }
    if (lockedBatch && wanted !== lockedBatch) {
      setError('You can only add your own batch.')
      return
    }
    if (batches.some((entry) => entry.batch === wanted && entry.section === section)) {
      setError('That batch is already on the list.')
      return
    }
    setError('')
    onAdd({ batch: wanted, section })
    setSection('')
  }
  return (
    <Field label="Add a whole batch" error={error} hint="Pick both sections or just one">
      <div className="grid gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
        <Input
          aria-label="Batch"
          value={batch}
          onChange={(event) => setBatch(event.target.value.toUpperCase())}
          placeholder="CSE-23"
          disabled={Boolean(lockedBatch)}
        />
        <SectionPicker
          value={section}
          onChange={setSection}
          emptyLabel="Both sections"
          hint="Both sections, or one of them"
          required={false}
        />
        <Button variant="secondary" onClick={add}>
          <Plus size={15} />
          Add
        </Button>
      </div>
      {batches.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2">
          {batches.map((entry) => (
            <span
              key={`${entry.batch}-${entry.section}`}
              className="inline-flex items-center gap-1.5 rounded-md border border-brand-200 bg-brand-50 py-1 pr-1 pl-2 text-xs text-ink"
            >
              {entry.batch}
              {entry.section ? ` section ${entry.section}` : ' (both sections)'}
              <button
                type="button"
                onClick={() => onDrop(entry)}
                aria-label={`Remove ${entry.batch}`}
                className="grid size-6 place-items-center text-muted hover:text-accent-600"
              >
                <X size={13} />
              </button>
            </span>
          ))}
        </div>
      )}
    </Field>
  )
}

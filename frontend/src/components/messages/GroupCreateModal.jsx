import { useState } from 'react'
import { Alert, Button, Input, Modal } from '../ui'
import BatchPicker from './BatchPicker'
import PeoplePicker from './PeoplePicker'
import api from '../../lib/api'

export default function GroupCreateModal({ open, onClose, onCreated, role, batch }) {
  const [title, setTitle] = useState('')
  const [search, setSearch] = useState('')
  const [picked, setPicked] = useState([])
  const [batches, setBatches] = useState([])
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const reset = () => {
    setTitle('')
    setSearch('')
    setPicked([])
    setBatches([])
    setError('')
  }
  const close = () => {
    reset()
    onClose()
  }
  const create = async () => {
    const name = title.trim()
    if (!name) {
      setError('Give the group a name first.')
      return
    }
    if (picked.length === 0 && batches.length === 0) {
      setError('Add at least one other person or a whole batch.')
      return
    }
    setSaving(true)
    setError('')
    try {
      const created = await api.post('/api/messages/groups', {
        title: name,
        user_ids: picked.map((contact) => contact.user_id),
        batches,
      })
      reset()
      onCreated(created)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  return (
    <Modal
      open={open}
      onClose={close}
      title="New group"
      footer={
        <>
          <Button variant="secondary" onClick={close}>
            Cancel
          </Button>
          <Button loading={saving} onClick={create}>
            Create the group
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {error && <Alert tone="error">{error}</Alert>}
        <Input
          label="Group name"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          placeholder="CSE-23 section A"
          maxLength={80}
          required
        />
        <PeoplePicker
          label="People"
          picked={picked}
          search={search}
          onSearchChange={setSearch}
          onPick={(contact) => setPicked((current) => [...current, contact])}
          onDrop={(contact) =>
            setPicked((current) => current.filter((entry) => entry.user_id !== contact.user_id))
          }
        />
        <BatchPicker
          batches={batches}
          lockedBatch={role === 'cr' ? batch : ''}
          onAdd={(entry) => setBatches((current) => [...current, entry])}
          onDrop={(entry) =>
            setBatches((current) =>
              current.filter(
                (item) => !(item.batch === entry.batch && item.section === entry.section),
              ),
            )
          }
        />
        <p className="label-caps">
          {picked.length + 1} picked
          {batches.length > 0
            ? ` · plus everyone in ${batches
                .map((entry) => (entry.section ? `${entry.batch} ${entry.section}` : entry.batch))
                .join(', ')}`
            : ''}
        </p>
      </div>
    </Modal>
  )
}

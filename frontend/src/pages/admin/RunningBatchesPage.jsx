import { useEffect, useState } from 'react'
import { Layers, Pencil, Plus, Power, Trash2 } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  DataTable,
  EmptyState,
  Input,
  Modal,
  PageHeader,
  Select,
  SkeletonTable,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import { useAuth } from '../../context/auth'
import api from '../../lib/api'
import { batchNameProblem, cleanBatchName } from '../../lib/batches'
import { useResource } from '../../lib/hooks'
import { SEMESTERS, semesterLabel } from '../../lib/options'

const EMPTY = { name: '', running_semester: SEMESTERS[0] }

function DeleteBatchModal({ batch, busy, onClose, onConfirmed }) {
  const [typed, setTyped] = useState('')
  const [problem, setProblem] = useState('')
  useEffect(() => {
    setTyped('')
    setProblem('')
  }, [batch])
  const expected = batch?.name || ''
  const matches = Boolean(expected) && cleanBatchName(typed) === expected
  const confirm = async () => {
    setProblem('')
    try {
      await onConfirmed(cleanBatchName(typed))
    } catch (failure) {
      if (failure?.cancelled) return
      setProblem(failure.message)
    }
  }
  return (
    <Modal
      open={Boolean(batch)}
      onClose={onClose}
      title={`Delete ${expected} permanently?`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" disabled={!matches} loading={busy} onClick={confirm}>
            Delete permanently
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        <Alert tone="error" title="This cannot be undone">
          The batch, its weekly routine, its classes and exams and its attendance sheets are deleted.
          Its {batch?.students || 0} student accounts are removed and follow the usual removed-account
          rules.
        </Alert>
        <Alert tone="info" title="Only need a pause?">
          Deactivate keeps every record and can be reversed at any time.
        </Alert>
        <Input
          label={`Type ${expected} to confirm`}
          value={typed}
          onChange={(event) => setTyped(event.target.value)}
          placeholder={expected}
          autoComplete="off"
        />
      </div>
    </Modal>
  )
}

export default function RunningBatchesPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const { role } = useAuth()
  const readOnly = role === 'super_admin'
  const [form, setForm] = useState(EMPTY)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState('')
  const [editing, setEditing] = useState('')
  const [draft, setDraft] = useState(SEMESTERS[0])
  const [deleting, setDeleting] = useState(null)
  const { data, loading, error: listError, reload } = useResource('/api/admin/batches')
  const items = data?.items || []
  const add = async (event) => {
    event.preventDefault()
    const name = cleanBatchName(form.name)
    const problem = batchNameProblem(name)
    if (problem) {
      setError(problem)
      return
    }
    setError('')
    setSaving(true)
    try {
      await api.post('/api/admin/batches', { name, running_semester: form.running_semester })
      toast.success(`${name} added`)
      setForm(EMPTY)
      reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const run = async (batch, work, done) => {
    setBusy(batch.name)
    try {
      await work()
      toast.success(done)
      reload()
    } catch (failure) {
      if (!failure?.cancelled) toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const saveSemester = (batch) =>
    run(
      batch,
      async () => {
        await api.patch(`/api/admin/batches/${batch.name}`, { running_semester: draft })
        setEditing('')
      },
      `${batch.name} now runs ${semesterLabel(draft)}`,
    )
  const deactivate = async (batch) => {
    const ok = await confirm({
      title: `Deactivate ${batch.name}?`,
      message: `Every record stays. Its ${batch.students} students are signed out and cannot sign in, and the batch leaves the routine, class, exam, notice and poll lists until you activate it again.`,
      confirmLabel: 'Deactivate',
    })
    if (!ok) return
    run(batch, () => api.post(`/api/admin/batches/${batch.name}/deactivate`), `${batch.name} deactivated`)
  }
  const activate = (batch) =>
    run(batch, () => api.post(`/api/admin/batches/${batch.name}/activate`), `${batch.name} activated`)
  const remove = async (confirmText) => {
    const batch = deleting
    setBusy(batch.name)
    try {
      const result = await api.delete(`/api/admin/batches/${batch.name}`, {
        body: { confirm: confirmText },
      })
      toast.success(result?.message || `${batch.name} deleted permanently`)
      setDeleting(null)
      reload()
    } finally {
      setBusy('')
    }
  }
  return (
    <>
      <PageHeader
        eyebrow="The department"
        title="Running batches"
        subtitle="The batches every routine, class, exam, notice and registration can choose from"
      />
      {!readOnly && (
        <Card hover={false} className="mb-5">
          <form onSubmit={add} className="flex flex-col gap-4">
            {error && <Alert tone="error">{error}</Alert>}
            <div className="grid gap-4 sm:grid-cols-[2fr_2fr_auto] sm:items-end">
              <Input
                label="Batch name"
                value={form.name}
                onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                placeholder="CSE-25"
                required
              />
              <Select
                label="Running semester"
                value={form.running_semester}
                onChange={(event) =>
                  setForm((current) => ({ ...current, running_semester: event.target.value }))
                }
                required
              >
                {SEMESTERS.map((entry) => (
                  <option key={entry} value={entry}>
                    {semesterLabel(entry)}
                  </option>
                ))}
              </Select>
              <Button type="submit" loading={saving}>
                <Plus size={15} />
                Add batch
              </Button>
            </div>
          </form>
        </Card>
      )}
      {listError && (
        <Alert tone="error" className="mb-5">
          {listError}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={5} />
      ) : items.length === 0 ? (
        <EmptyState icon={Layers} title="No batches yet" text="Add the first batch above." />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              <th className="px-4 py-3 font-semibold text-ink-soft">Batch</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Running semester</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Students</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Status</th>
              {!readOnly && (
                <th className="px-4 py-3 text-right font-semibold text-ink-soft">Actions</th>
              )}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.name} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3">
                  <div className="font-medium text-ink">{item.name}</div>
                  <div className="text-xs text-ink-soft">
                    Sections {(item.sections || []).join(' and ')}
                  </div>
                </td>
                <td className="px-4 py-3 text-ink-soft">
                  {editing === item.name ? (
                    <div className="flex flex-wrap items-end gap-2">
                      <Select
                        label="Running semester"
                        value={draft}
                        onChange={(event) => setDraft(event.target.value)}
                      >
                        {SEMESTERS.map((entry) => (
                          <option key={entry} value={entry}>
                            {semesterLabel(entry)}
                          </option>
                        ))}
                      </Select>
                      <Button size="sm" loading={busy === item.name} onClick={() => saveSemester(item)}>
                        Save
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => setEditing('')}>
                        Cancel
                      </Button>
                    </div>
                  ) : (
                    item.running_semester_label || semesterLabel(item.running_semester)
                  )}
                </td>
                <td className="px-4 py-3 tabular-nums text-ink-soft">{item.students}</td>
                <td className="px-4 py-3">
                  {item.status === 'active' ? (
                    <StatusBadge status="active" />
                  ) : (
                    <StatusBadge status="inactive" label="Inactive" tone="muted" />
                  )}
                </td>
                {!readOnly && (
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap justify-end gap-2">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => {
                          setDraft(item.running_semester)
                          setEditing(item.name)
                        }}
                      >
                        <Pencil size={14} />
                        Semester
                      </Button>
                      {item.status === 'active' ? (
                        <Button
                          variant="secondary"
                          size="sm"
                          loading={busy === item.name}
                          onClick={() => deactivate(item)}
                        >
                          <Power size={14} />
                          Deactivate
                        </Button>
                      ) : (
                        <Button size="sm" loading={busy === item.name} onClick={() => activate(item)}>
                          <Power size={14} />
                          Activate
                        </Button>
                      )}
                      <Button variant="danger" size="sm" onClick={() => setDeleting(item)}>
                        <Trash2 size={14} />
                        Delete
                      </Button>
                    </div>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      <DeleteBatchModal
        batch={deleting}
        busy={Boolean(deleting) && busy === deleting.name}
        onClose={() => setDeleting(null)}
        onConfirmed={remove}
      />
    </>
  )
}

import { useState } from 'react'
import Button from '../ui/Button'
import Modal from '../ui/Modal'
import { PERMISSION_ORDER, permissionLabel } from '../../lib/permissions'

export default function PermissionPicker({ admin, open, onClose, onSave }) {
  const [chosen, setChosen] = useState(admin?.permissions || [])
  const [saving, setSaving] = useState(false)
  const toggle = (name) => {
    setChosen((current) =>
      current.includes(name) ? current.filter((entry) => entry !== name) : [...current, name],
    )
  }
  const save = async () => {
    setSaving(true)
    try {
      await onSave(chosen)
    } finally {
      setSaving(false)
    }
  }
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`What ${admin?.username || admin?.name || 'this admin'} may do`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={save} loading={saving}>
            Save permissions
          </Button>
        </>
      }
    >
      <ul className="flex flex-col gap-2">
        {PERMISSION_ORDER.map((name) => (
          <li key={name}>
            <label className="tap-target flex items-center gap-3 rounded-tile px-2 py-2 hover:bg-wash">
              <input
                type="checkbox"
                checked={chosen.includes(name)}
                onChange={() => toggle(name)}
                className="size-4 accent-brand-600"
              />
              <span className="text-sm text-ink">{permissionLabel(name)}</span>
            </label>
          </li>
        ))}
      </ul>
      <p className="mt-3 text-xs text-muted">
        The office pages the admin may not use disappear from their navigation, and the server
        refuses those routes.
      </p>
    </Modal>
  )
}

import { useState } from 'react'
import { UserSearch } from 'lucide-react'
import { Alert, Avatar, Button, Input, Modal, Select, SkeletonLines, StatusBadge } from '../ui'
import api from '../../lib/api'
import { useDebounced, useResource } from '../../lib/hooks'
import { CONTACT_ROLES, contactDetail, scopesForRole, shortRoleLabel } from '../../lib/messages'

const TABS = [
  { value: 'person', label: 'Person' },
  { value: 'group', label: 'Group' },
]

export default function NewConversationModal({ open, onClose, role, batch, section, onOpened }) {
  const scopes = scopesForRole(role)
  const [tab, setTab] = useState('person')
  const [search, setSearch] = useState('')
  const [contactRole, setContactRole] = useState('')
  const [scope, setScope] = useState(scopes[0].value)
  const [groupBatch, setGroupBatch] = useState(batch || '')
  const [groupSection, setGroupSection] = useState(section || '')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const settled = useDebounced(search, 300)
  const chosen = scopes.find((entry) => entry.value === scope) || scopes[0]
  const needsList = chosen.batch === 'choose'
  const contacts = useResource(open && tab === 'person' ? '/api/messages/contacts' : null, {
    search: settled,
    role: contactRole,
  })
  const batches = useResource(open && tab === 'group' && needsList ? '/api/lookups/batches' : null)
  const sections = useResource(
    open && tab === 'group' && chosen.section && groupBatch ? '/api/lookups/sections' : null,
    { batch: groupBatch },
  )
  const openConversation = async (path, payload, extra) => {
    setError('')
    setBusy(true)
    try {
      const data = await api.post(path, payload)
      onOpened({ ...data, ...extra })
      setSearch('')
    } catch (failure) {
      setError(failure.message)
    } finally {
      setBusy(false)
    }
  }
  const startGroup = () => {
    if (needsList && !groupBatch) {
      setError('Choose a batch first.')
      return
    }
    if (chosen.section && !groupSection) {
      setError('Choose a section first.')
      return
    }
    openConversation('/api/messages/groups', {
      scope,
      batch: chosen.batch ? groupBatch : '',
      section: chosen.section ? groupSection : '',
    })
  }
  const items = contacts.data?.items || []
  return (
    <Modal open={open} onClose={onClose} title="New message">
      <div className="mb-4 inline-flex rounded-lg border border-brand-200 bg-white p-1">
        {TABS.map((entry) => (
          <Button
            key={entry.value}
            size="sm"
            variant={tab === entry.value ? 'primary' : 'ghost'}
            onClick={() => {
              setTab(entry.value)
              setError('')
            }}
          >
            {entry.label}
          </Button>
        ))}
      </div>
      {error && (
        <Alert tone="error" className="mb-4">
          {error}
        </Alert>
      )}
      {tab === 'person' ? (
        <div className="flex flex-col gap-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              label="Search"
              type="search"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Name or registration number"
            />
            <Select
              label="Role"
              value={contactRole}
              onChange={(event) => setContactRole(event.target.value)}
            >
              {CONTACT_ROLES.map((entry) => (
                <option key={entry.value} value={entry.value}>
                  {entry.label}
                </option>
              ))}
            </Select>
          </div>
          <div className="max-h-72 overflow-y-auto rounded-lg border border-brand-100">
            {contacts.loading ? (
              <SkeletonLines lines={4} className="p-3" />
            ) : items.length === 0 ? (
              <p className="flex items-center gap-2 p-4 text-sm text-brand-500">
                <UserSearch size={16} />
                Nobody matches this search.
              </p>
            ) : (
              <ul>
                {items.map((contact) => (
                  <li key={contact.user_id}>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() =>
                        openConversation(
                          '/api/messages/direct',
                          { user_id: contact.user_id },
                          { title: contact.name, photo_url: contact.photo_url },
                        )
                      }
                      className="flex w-full items-center gap-3 border-b border-brand-50 px-3 py-2.5 text-left transition-colors last:border-b-0 hover:bg-brand-50/70 disabled:cursor-not-allowed"
                    >
                      <Avatar src={contact.photo_url} name={contact.name} size="sm" />
                      <span className="min-w-0 flex-1">
                        <span className="block truncate text-sm font-medium text-brand-900">
                          {contact.name}
                        </span>
                        {contactDetail(contact) && (
                          <span className="block truncate text-xs text-brand-500">
                            {contactDetail(contact)}
                          </span>
                        )}
                      </span>
                      <StatusBadge status="unavailable" label={shortRoleLabel(contact.role)} />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <Select
            label="Who should it reach?"
            value={scope}
            onChange={(event) => {
              setScope(event.target.value)
              setError('')
            }}
          >
            {scopes.map((entry) => (
              <option key={entry.value} value={entry.value}>
                {entry.label}
              </option>
            ))}
          </Select>
          {chosen.batch === 'own' && <Input label="Batch" value={batch || ''} readOnly disabled />}
          {needsList && (
            <Select
              label="Batch"
              value={groupBatch}
              onChange={(event) => {
                setGroupBatch(event.target.value)
                setGroupSection('')
              }}
              required
            >
              <option value="">Choose a batch</option>
              {(batches.data?.items || []).map((entry) => (
                <option key={entry} value={entry}>
                  {entry}
                </option>
              ))}
            </Select>
          )}
          {chosen.section && (
            <Select
              label="Section"
              value={groupSection}
              onChange={(event) => setGroupSection(event.target.value)}
              required
            >
              <option value="">Choose a section</option>
              {(sections.data?.items || []).map((entry) => (
                <option key={entry} value={entry}>
                  {entry}
                </option>
              ))}
            </Select>
          )}
          <p className="text-xs text-brand-500">
            Only you can post in a group you start. Everybody else answers you privately.
          </p>
          <Button loading={busy} onClick={startGroup} className="self-end">
            Open the group
          </Button>
        </div>
      )}
    </Modal>
  )
}

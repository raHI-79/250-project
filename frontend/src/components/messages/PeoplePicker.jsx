import { X } from 'lucide-react'
import { Avatar, Field, Input, SkeletonLines, StatusBadge } from '../ui'
import { useDebounced, useResource } from '../../lib/hooks'
import { contactDetail, shortRoleLabel } from '../../lib/messages'
import cx from '../../lib/cx'

export default function PeoplePicker({ picked, onPick, onDrop, search, onSearchChange, label }) {
  const settled = useDebounced(search.trim(), 300)
  const contacts = useResource(settled ? '/api/messages/contacts' : null, { search: settled })
  const people = (contacts.data?.items || []).filter(
    (contact) => !picked.some((entry) => entry.user_id === contact.user_id),
  )
  return (
    <Field label={label} hint="Search by name or ID, then pick as many people as you need">
      <Input
        type="search"
        value={search}
        onChange={(event) => onSearchChange(event.target.value)}
        placeholder="Search students, CRs and teachers"
        aria-label="Search people"
      />
      {picked.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2">
          {picked.map((contact) => (
            <span
              key={contact.user_id}
              className="inline-flex items-center gap-1.5 rounded-md border border-brand-200 bg-brand-50 py-1 pr-1 pl-2 text-xs text-ink"
            >
              <span className="wrap-long max-w-40">{contact.name}</span>
              <button
                type="button"
                onClick={() => onDrop(contact)}
                aria-label={`Remove ${contact.name}`}
                className="grid size-6 place-items-center text-muted hover:text-accent-600"
              >
                <X size={13} />
              </button>
            </span>
          ))}
        </div>
      )}
      {settled && (
        <div className="mt-2 border border-line">
          {contacts.loading ? (
            <SkeletonLines lines={2} className="p-3" />
          ) : people.length === 0 ? (
            <p className="p-3 text-sm text-muted">Nobody else matches this search.</p>
          ) : (
            <ul className="max-h-48 overflow-y-auto">
              {people.map((contact) => (
                <li key={contact.user_id}>
                  <button
                    type="button"
                    onClick={() => onPick(contact)}
                    className={cx(
                      'flex w-full items-center gap-3 border-b border-line px-3 py-2.5 text-left transition-colors last:border-b-0 hover:bg-wash',
                    )}
                  >
                    <Avatar src={contact.photo_url} name={contact.name} size="sm" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-semibold text-ink">
                        {contact.name}
                      </span>
                      {contactDetail(contact) && (
                        <span className="block truncate text-xs text-muted">
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
      )}
    </Field>
  )
}

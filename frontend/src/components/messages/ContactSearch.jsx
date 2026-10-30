import { Search, UserSearch, X } from 'lucide-react'
import { Avatar, Input, SkeletonLines, StatusBadge } from '../ui'
import { useDebounced, useResource } from '../../lib/hooks'
import { contactChips, contactDetail, shortRoleLabel } from '../../lib/messages'
import cx from '../../lib/cx'

export default function ContactSearch({
  search,
  onSearchChange,
  roleFilter,
  onRoleFilterChange,
  onPickPerson,
  viewerRole,
  busy,
}) {
  const chips = contactChips(viewerRole)
  const text = search.trim()
  const settled = useDebounced(text, 300)
  const contacts = useResource(settled ? '/api/messages/contacts' : null, {
    search: settled,
    role: roleFilter,
  })
  const people = contacts.data?.items || []
  return (
    <div className="sticky top-0 z-20 shrink-0 border-b border-line bg-surface/95 p-3 backdrop-blur-sm">
      <Input
        type="search"
        value={search}
        onChange={(event) => onSearchChange(event.target.value)}
        placeholder="Search people and conversations"
        aria-label="Search people and conversations"
      />
      <div
        role="tablist"
        aria-label="Filter by role"
        className="no-scrollbar -mx-3 mt-2 flex items-center gap-1.5 overflow-x-auto px-3"
      >
        {chips.map((chip) => {
          const active = roleFilter === chip.value
          return (
            <button
              key={chip.value || 'all'}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => onRoleFilterChange(chip.value)}
              className={cx(
                'press min-h-9 shrink-0 rounded-full border px-3 text-xs font-semibold whitespace-nowrap transition-colors',
                active
                  ? 'border-brand-600 bg-brand-600 text-page shadow-[var(--shadow-glass)]'
                  : 'border-line-strong bg-surface font-medium text-muted hover:border-brand-300 hover:text-ink',
              )}
            >
              {chip.label}
            </button>
          )
        })}
        {text && (
          <button
            type="button"
            onClick={() => onSearchChange('')}
            className="ml-auto inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-1 text-xs text-muted hover:bg-wash"
          >
            <X size={12} />
            Clear
          </button>
        )}
      </div>
      {settled && (
        <div className="mt-3 rounded-xl border border-line">
          {contacts.loading ? (
            <SkeletonLines lines={3} className="p-3" />
          ) : people.length === 0 ? (
            <p className="flex items-center gap-2 p-3 text-sm text-muted">
              <UserSearch size={15} />
              Nobody matches this search.
            </p>
          ) : (
            <ul className="max-h-64 overflow-y-auto">
              {people.map((contact) => (
                <li key={`person-${contact.user_id}`}>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onPickPerson(contact)}
                    className="flex w-full items-center gap-3 border-b border-line px-3 py-2.5 text-left transition-colors last:border-b-0 hover:bg-wash/70 disabled:cursor-not-allowed"
                  >
                    <Avatar src={contact.photo_url} name={contact.name} size="sm" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium text-ink">
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
      {!settled && (
        <p className="mt-2 flex items-center gap-1.5 text-xs text-faint">
          <Search size={12} />
          Type a name to start a chat, or pick a conversation below.
        </p>
      )}
    </div>
  )
}

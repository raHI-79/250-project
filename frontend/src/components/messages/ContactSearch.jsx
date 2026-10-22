import { Search, UserSearch, Users, X } from 'lucide-react'
import { Avatar, Input, SkeletonLines, StatusBadge } from '../ui'
import { useDebounced, useResource } from '../../lib/hooks'
import { contactDetail, groupTitleOf, scopesForRole, shortRoleLabel } from '../../lib/messages'
import cx from '../../lib/cx'

const CHIPS = [
  { value: '', label: 'All' },
  { value: 'teacher', label: 'Teachers' },
  { value: 'cr', label: 'CRs' },
  { value: 'student', label: 'Students' },
  { value: 'admin', label: 'Office' },
]

export default function ContactSearch({
  search,
  onSearchChange,
  role,
  roleFilter,
  onRoleFilterChange,
  batch,
  section,
  onPickPerson,
  onPickGroup,
  busy,
}) {
  const text = search.trim()
  const settled = useDebounced(text, 300)
  const contacts = useResource(settled ? '/api/messages/contacts' : null, {
    search: settled,
    role: roleFilter,
  })
  const people = contacts.data?.items || []
  const scopes = scopesForRole(role).filter((scope) => scope.batch !== 'choose' || batch)
  const groups = settled
    ? scopes
        .map((scope) => ({
          scope: scope.value,
          batch: scope.batch ? batch : '',
          section: scope.section ? section : '',
          title: groupTitleOf(scope.value, batch, section) || scope.label,
          label: scope.label,
        }))
        .filter((group) =>
          `${group.title} ${group.label}`.toLowerCase().includes(settled.toLowerCase()),
        )
    : []
  return (
    <div className="border-b border-line p-3">
      <Input
        type="search"
        value={search}
        onChange={(event) => onSearchChange(event.target.value)}
        placeholder="Search people and conversations"
        aria-label="Search people and conversations"
      />
      <div className="mt-2 flex flex-wrap gap-1.5">
        {CHIPS.map((chip) => (
          <button
            key={chip.value || 'all'}
            type="button"
            onClick={() => onRoleFilterChange(chip.value)}
            className={cx(
              'rounded-full border px-2.5 py-1 text-xs font-medium transition-colors',
              roleFilter === chip.value
                ? 'border-brand-600 bg-brand-600 text-white'
                : 'border-line-strong bg-surface text-brand-600 hover:bg-wash',
            )}
          >
            {chip.label}
          </button>
        ))}
        {text && (
          <button
            type="button"
            onClick={() => onSearchChange('')}
            className="ml-auto inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs text-muted hover:bg-wash"
          >
            <X size={12} />
            Clear
          </button>
        )}
      </div>
      {settled && (
        <div className="mt-3 rounded-md border border-line">
          {contacts.loading ? (
            <SkeletonLines lines={3} className="p-3" />
          ) : people.length === 0 && groups.length === 0 ? (
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
              {groups.map((group) => (
                <li key={`group-${group.scope}-${group.section}`}>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onPickGroup(group)}
                    className="flex w-full items-center gap-3 border-b border-line px-3 py-2.5 text-left transition-colors last:border-b-0 hover:bg-wash/70 disabled:cursor-not-allowed"
                  >
                    <span className="inline-flex size-9 shrink-0 items-center justify-center rounded-full border border-line bg-wash text-brand-600">
                      <Users size={16} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium text-ink">
                        {group.title}
                      </span>
                      <span className="block text-xs text-muted">{group.label}</span>
                    </span>
                    <StatusBadge status="unavailable" label="Group" />
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

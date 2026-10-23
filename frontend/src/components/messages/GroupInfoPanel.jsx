import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { LogOut, Pencil, ShieldCheck, UserMinus, UserPlus } from 'lucide-react'
import {
  Alert,
  Avatar,
  Button,
  Input,
  SkeletonLines,
  StatusBadge,
  useConfirm,
  useToast,
} from '../ui'
import BatchPicker from './BatchPicker'
import PeoplePicker from './PeoplePicker'
import api from '../../lib/api'
import paths from '../../lib/paths'
import { shortRoleLabel } from '../../lib/messages'

export default function GroupInfoPanel({ conversation, meId, role, batch, onChanged, onLeft }) {
  const toast = useToast()
  const confirm = useConfirm()
  const [members, setMembers] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [adding, setAdding] = useState(false)
  const [search, setSearch] = useState('')
  const [picked, setPicked] = useState([])
  const [batches, setBatches] = useState([])
  const [renaming, setRenaming] = useState(false)
  const [title, setTitle] = useState(conversation.title || '')
  const [busy, setBusy] = useState(false)
  const load = useCallback(() => {
    setLoading(true)
    return api
      .get(`/api/messages/conversations/${conversation.id}/members`)
      .then((data) => {
        setError('')
        setMembers(Array.isArray(data?.items) ? data.items : [])
      })
      .catch((failure) => setError(failure.message))
      .finally(() => setLoading(false))
  }, [conversation.id])
  useEffect(() => {
    load()
  }, [load])
  const isAdmin = members.some(
    (member) => member.user_id === meId && member.role === 'admin',
  )
  const addMembers = async () => {
    if (picked.length === 0 && batches.length === 0) {
      toast.error('Pick somebody or a batch first.')
      return
    }
    setBusy(true)
    try {
      await api.post(`/api/messages/conversations/${conversation.id}/members`, {
        user_ids: picked.map((contact) => contact.user_id),
        batches,
      })
      toast.success('Members added')
      setPicked([])
      setBatches([])
      setSearch('')
      setAdding(false)
      await load()
      onChanged?.()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(false)
    }
  }
  const remove = async (member) => {
    const ok = await confirm({
      title: 'Remove this member?',
      message: `${member.name} loses access to this group and its messages from now on.`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    setBusy(true)
    try {
      await api.delete(`/api/messages/conversations/${conversation.id}/members/${member.user_id}`)
      toast.success('Member removed')
      await load()
      onChanged?.()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(false)
    }
  }
  const setAdmin = async (member, admin) => {
    setBusy(true)
    try {
      const path = `/api/messages/conversations/${conversation.id}/members/${member.user_id}/admin`
      if (admin) await api.post(path)
      else await api.delete(path)
      toast.success(admin ? 'Member is now a group admin' : 'Member is no longer a group admin')
      await load()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(false)
    }
  }
  const rename = async () => {
    const wanted = title.trim()
    if (!wanted) {
      toast.error('The group needs a name.')
      return
    }
    setBusy(true)
    try {
      await api.patch(`/api/messages/conversations/${conversation.id}`, { title: wanted })
      toast.success('Group renamed')
      setRenaming(false)
      onChanged?.()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(false)
    }
  }
  const leave = async () => {
    const ok = await confirm({
      title: 'Leave this group?',
      message: 'You stop receiving its messages. A group admin can add you back later.',
      confirmLabel: 'Leave the group',
    })
    if (!ok) return
    setBusy(true)
    try {
      await api.post(`/api/messages/conversations/${conversation.id}/leave`)
      toast.success('You left the group')
      onLeft?.()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden rounded-card border border-line bg-surface">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line px-4 py-3">
        <div className="min-w-0">
          <p className="label-caps">Group</p>
          <p className="font-display wrap-long text-lg font-bold text-ink">{conversation.title}</p>
        </div>
        {isAdmin && !renaming && (
          <Button variant="secondary" size="sm" onClick={() => setRenaming(true)}>
            <Pencil size={14} />
            Rename
          </Button>
        )}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {error && <Alert tone="error" className="mb-4">{error}</Alert>}
        {renaming && (
          <div className="mb-4 flex flex-col gap-3 rounded-xl border border-line bg-wash p-3">
            <Input
              label="Group name"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              maxLength={80}
            />
            <div className="flex flex-wrap justify-end gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  setRenaming(false)
                  setTitle(conversation.title || '')
                }}
              >
                Cancel
              </Button>
              <Button size="sm" loading={busy} onClick={rename}>
                Save the name
              </Button>
            </div>
          </div>
        )}
        {isAdmin && (
          <div className="mb-4">
            {adding ? (
              <div className="flex flex-col gap-4 rounded-xl border border-line bg-wash p-3">
                <PeoplePicker
                  label="Add people"
                  picked={picked}
                  search={search}
                  onSearchChange={setSearch}
                  onPick={(contact) => setPicked((current) => [...current, contact])}
                  onDrop={(contact) =>
                    setPicked((current) =>
                      current.filter((entry) => entry.user_id !== contact.user_id),
                    )
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
                <div className="flex flex-wrap justify-end gap-2">
                  <Button variant="secondary" size="sm" onClick={() => setAdding(false)}>
                    Cancel
                  </Button>
                  <Button size="sm" loading={busy} onClick={addMembers}>
                    Add to the group
                  </Button>
                </div>
              </div>
            ) : (
              <Button variant="secondary" size="sm" onClick={() => setAdding(true)}>
                <UserPlus size={14} />
                Add members
              </Button>
            )}
          </div>
        )}
        <p className="label-caps mb-2">
          {members.length} member{members.length === 1 ? '' : 's'}
        </p>
        {loading ? (
          <SkeletonLines lines={4} />
        ) : (
          <ul className="flex flex-col">
            {members.map((member) => (
              <li
                key={member.user_id}
                className="flex flex-wrap items-center gap-3 border-b border-dashed border-line py-3 last:border-b-0"
              >
                <Avatar src={member.photo_url} name={member.name} size="sm" />
                <div className="min-w-0 flex-1">
                  <Link
                    to={paths.person(member.user_id)}
                    className="wrap-long block text-sm font-semibold text-ink hover:text-brand-600"
                  >
                    {member.name}
                  </Link>
                  <p className="text-xs text-muted">
                    {shortRoleLabel(member.account_role)}
                    {member.batch ? ` · ${member.batch}` : ''}
                    {member.section ? ` ${member.section}` : ''}
                  </p>
                </div>
                {member.role === 'admin' && <StatusBadge status="approved" label="Admin" />}
                {isAdmin && member.user_id !== meId && (
                  <div className="flex flex-wrap gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={busy}
                      onClick={() => setAdmin(member, member.role !== 'admin')}
                      aria-label={
                        member.role === 'admin'
                          ? `Remove admin from ${member.name}`
                          : `Make ${member.name} an admin`
                      }
                    >
                      <ShieldCheck size={14} />
                      {member.role === 'admin' ? 'Remove admin' : 'Make admin'}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={busy}
                      onClick={() => remove(member)}
                      aria-label={`Remove ${member.name}`}
                    >
                      <UserMinus size={14} />
                      Remove
                    </Button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
      <div className="border-t border-line p-3">
        <Button variant="danger" size="sm" loading={busy} onClick={leave}>
          <LogOut size={14} />
          Leave the group
        </Button>
      </div>
    </div>
  )
}

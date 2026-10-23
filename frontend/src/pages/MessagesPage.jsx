import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ArrowLeft, Info, Plus, Trash2, Users } from 'lucide-react'
import ContactSearch from '../components/messages/ContactSearch'
import ConversationList from '../components/messages/ConversationList'
import GroupCreateModal from '../components/messages/GroupCreateModal'
import GroupInfoPanel from '../components/messages/GroupInfoPanel'
import MessageComposer from '../components/messages/MessageComposer'
import MessageThread from '../components/messages/MessageThread'
import { Alert, Avatar, Button, EmptyState, PageHeader, useConfirm, useToast } from '../components/ui'
import { useAuth } from '../context/auth'
import api from '../lib/api'
import cx from '../lib/cx'
import { refreshUnread, usePolling } from '../lib/hooks'
import { LIST_INTERVAL, THREAD_INTERVAL, conversationDetail } from '../lib/messages'

export default function MessagesPage() {
  const { user } = useAuth()
  const toast = useToast()
  const confirm = useConfirm()
  const [items, setItems] = useState([])
  const [listError, setListError] = useState('')
  const [listLoading, setListLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [activeId, setActiveId] = useState(0)
  const [opened, setOpened] = useState(null)
  const [messages, setMessages] = useState([])
  const [threadError, setThreadError] = useState('')
  const [threadLoading, setThreadLoading] = useState(false)
  const [roleFilter, setRoleFilter] = useState('')
  const [starting, setStarting] = useState(false)
  const [creating, setCreating] = useState(false)
  const [showInfo, setShowInfo] = useState(false)
  const [params, setParams] = useSearchParams()
  const loadList = useCallback(
    () =>
      api
        .get('/api/messages/conversations')
        .then((data) => {
          setListError('')
          setItems(Array.isArray(data?.items) ? data.items : [])
        })
        .catch((failure) => setListError(failure.message))
        .finally(() => setListLoading(false)),
    [],
  )
  const loadThread = useCallback(() => {
    if (!activeId) return undefined
    return api
      .get(`/api/messages/conversations/${activeId}/messages`)
      .then((data) => {
        setThreadError('')
        setMessages([...(data?.items || [])].reverse())
        refreshUnread()
      })
      .catch((failure) => setThreadError(failure.message))
  }, [activeId])
  useEffect(() => {
    loadList()
  }, [loadList])
  usePolling(loadList, LIST_INTERVAL)
  useEffect(() => {
    if (!activeId) return
    setThreadLoading(true)
    Promise.resolve(loadThread()).finally(() => setThreadLoading(false))
  }, [activeId, loadThread])
  usePolling(activeId ? loadThread : null, THREAD_INTERVAL)
  const canStartGroups = user?.role === 'teacher' || user?.role === 'cr'
  const active = useMemo(
    () => items.find((entry) => entry.id === activeId) || opened,
    [items, activeId, opened],
  )
  const select = (item) => {
    setMessages([])
    setThreadError('')
    setShowInfo(false)
    setOpened(item)
    setActiveId(item.id)
    setItems((current) =>
      current.map((entry) => (entry.id === item.id ? { ...entry, unread_count: 0 } : entry)),
    )
  }
  const openConversation = (conversation) => {
    select(conversation)
    setSearch('')
    loadList()
  }
  const startWith = async (path, payload, extra) => {
    setStarting(true)
    setListError('')
    try {
      const data = await api.post(path, payload)
      openConversation({ ...data, ...extra })
    } catch (failure) {
      setListError(failure.message)
    } finally {
      setStarting(false)
    }
  }
  const pickPerson = (contact) =>
    startWith(
      '/api/messages/direct',
      { user_id: contact.user_id },
      { title: contact.name, photo_url: contact.photo_url, kind: 'direct', can_post: true },
    )
  const editMessage = async (item, body) => {
    try {
      await api.patch(`/api/messages/${item.id}`, { body })
      await loadThread()
      loadList()
    } catch (failure) {
      toast.error(failure.message)
      throw failure
    }
  }
  const deleteMessage = async (item) => {
    const ok = await confirm({
      title: 'Delete this message?',
      message: 'The thread will show that the message was deleted.',
      confirmLabel: 'Delete',
      tone: 'danger',
    })
    if (!ok) return
    try {
      await api.delete(`/api/messages/${item.id}`)
      await loadThread()
      loadList()
    } catch (failure) {
      toast.error(failure.message)
    }
  }
  const clearChat = async () => {
    const ok = await confirm({
      title: 'Delete this chat?',
      message:
        'The messages so far leave your list for you only. Everybody else keeps the whole chat, you stay in a group you are a member of, and the chat comes back when a new message arrives.',
      confirmLabel: 'Delete for me',
      tone: 'danger',
    })
    if (!ok) return
    try {
      await api.delete(`/api/messages/conversations/${activeId}`)
      toast.success('Chat deleted for you')
      setMessages([])
      close()
      loadList()
      refreshUnread()
    } catch (failure) {
      toast.error(failure.message)
    }
  }
  const send = async ({ body, link, file }) => {
    const payload = new FormData()
    if (body) payload.set('body', body)
    if (link) payload.set('link', link)
    if (file) payload.set('file', file)
    await api.post(`/api/messages/conversations/${activeId}/messages`, payload)
    await loadThread()
    loadList()
  }
  useEffect(() => {
    const wanted = Number(params.get('c') || 0)
    if (!wanted || wanted === activeId) return
    const found = items.find((entry) => entry.id === wanted)
    if (found) {
      select(found)
      return
    }
    if (items.length > 0) setActiveId(wanted)
  }, [params, items, activeId])
  const close = () => {
    setActiveId(0)
    setParams(
      (current) => {
        const copy = new URLSearchParams(current)
        copy.delete('c')
        return copy
      },
      { replace: true },
    )
  }
  return (
    <>
      <PageHeader
        eyebrow="Department messages"
        title="Messages"
        subtitle="Search for a person, open a group, then write"
        actions={
          canStartGroups ? (
            <Button onClick={() => setCreating(true)}>
              <Plus size={15} />
              New group
            </Button>
          ) : null
        }
      />
      {listError && (
        <Alert tone="error" className="mb-5">
          {listError}
        </Alert>
      )}
      <div className="flex h-[calc(100dvh-11rem)] min-h-[26rem] gap-4 lg:h-[72dvh]">
        <div className={cx('w-full md:w-80 md:shrink-0', activeId && 'hidden md:block')}>
          <ConversationList
            items={items}
            activeId={activeId}
            onSelect={select}
            search={search}
            loading={listLoading}
            header={
              <ContactSearch
                search={search}
                onSearchChange={setSearch}
                roleFilter={roleFilter}
                onRoleFilterChange={setRoleFilter}
                viewerRole={user?.role}
                onPickPerson={pickPerson}
                busy={starting}
              />
            }
          />
        </div>
        <div className={cx('min-w-0 flex-1', !activeId && 'hidden md:block')}>
          {active && showInfo && active.kind === 'group' ? (
            <GroupInfoPanel
              conversation={active}
              meId={user?.id}
              role={user?.role}
              batch={user?.batch}
              onChanged={loadList}
              onLeft={() => {
                setShowInfo(false)
                close()
                loadList()
              }}
            />
          ) : active ? (
            <div className="flex h-full min-h-0 flex-col overflow-hidden rounded-card border border-line bg-page">
              <div className="flex items-center gap-2 border-b border-line bg-surface px-2 py-2">
                <button
                  type="button"
                  onClick={close}
                  aria-label="Back to the conversations"
                  className="tap-target grid shrink-0 place-items-center rounded-full text-ink-soft transition-colors hover:bg-wash md:hidden"
                >
                  <ArrowLeft size={20} />
                </button>
                {active.kind === 'group' ? (
                  <span className="inline-flex size-10 shrink-0 items-center justify-center rounded-full border border-line bg-wash text-brand-600">
                    <Users size={18} />
                  </span>
                ) : (
                  <Avatar src={active.photo_url} name={active.title} />
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-semibold text-ink">{active.title}</p>
                  <p className="text-xs text-muted">{conversationDetail(active)}</p>
                </div>
                {active.kind === 'group' && (
                  <button
                    type="button"
                    onClick={() => setShowInfo((current) => !current)}
                    aria-pressed={showInfo}
                    aria-label={showInfo ? 'Hide the members' : 'Show the members'}
                    className="tap-target grid shrink-0 place-items-center rounded-full text-ink-soft transition-colors hover:bg-wash"
                  >
                    <Info size={19} />
                  </button>
                )}
                <button
                  type="button"
                  onClick={clearChat}
                  aria-label="Delete this chat"
                  className="tap-target grid shrink-0 place-items-center rounded-full text-ink-soft transition-colors hover:bg-wash hover:text-danger-600"
                >
                  <Trash2 size={19} />
                </button>
              </div>
              {threadError && (
                <Alert tone="error" className="m-3">
                  {threadError}
                </Alert>
              )}
              <div className="min-h-0 flex-1 overflow-y-auto">
                <MessageThread
                  items={messages}
                  meId={user?.id}
                  loading={threadLoading}
                  onEdit={editMessage}
                  onDelete={deleteMessage}
                />
              </div>
              <MessageComposer onSend={send} />
            </div>
          ) : (
            <EmptyState
              icon={Users}
              title="Pick a conversation"
              text="Choose one on the left or search for a person to start a new chat."
              className="h-full"
            />
          )}
        </div>
      </div>
      <GroupCreateModal
        open={creating}
        onClose={() => setCreating(false)}
        role={user?.role}
        batch={user?.batch}
        onCreated={(created) => {
          setCreating(false)
          openConversation({ ...created, kind: 'group', can_post: true })
        }}
      />
    </>
  )
}

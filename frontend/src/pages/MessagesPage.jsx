import { useCallback, useEffect, useMemo, useState } from 'react'
import { ArrowLeft, Plus, Reply, Users } from 'lucide-react'
import ConversationList from '../components/messages/ConversationList'
import MessageComposer from '../components/messages/MessageComposer'
import MessageThread from '../components/messages/MessageThread'
import NewConversationModal from '../components/messages/NewConversationModal'
import { Alert, Avatar, Button, EmptyState, PageHeader } from '../components/ui'
import { useAuth } from '../context/auth'
import api from '../lib/api'
import cx from '../lib/cx'
import { refreshUnread, usePolling } from '../lib/hooks'
import { LIST_INTERVAL, THREAD_INTERVAL, conversationDetail } from '../lib/messages'

export default function MessagesPage() {
  const { user } = useAuth()
  const [items, setItems] = useState([])
  const [listError, setListError] = useState('')
  const [listLoading, setListLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [activeId, setActiveId] = useState(0)
  const [opened, setOpened] = useState(null)
  const [messages, setMessages] = useState([])
  const [threadError, setThreadError] = useState('')
  const [threadLoading, setThreadLoading] = useState(false)
  const [newOpen, setNewOpen] = useState(false)
  const [creator, setCreator] = useState({ id: 0, name: '' })
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
  const active = useMemo(
    () => items.find((entry) => entry.id === activeId) || opened,
    [items, activeId, opened],
  )
  const groupOwner = active && active.kind === 'group' && !active.can_post ? active.created_by : 0
  useEffect(() => {
    if (!groupOwner) {
      setCreator({ id: 0, name: '' })
      return undefined
    }
    if (creator.id === groupOwner) return undefined
    const posted = messages.find((item) => item.sender_id === groupOwner)
    if (posted) {
      setCreator({ id: groupOwner, name: posted.sender_name })
      return undefined
    }
    let live = true
    api
      .get(`/api/people/${groupOwner}`)
      .then((data) => {
        if (live) setCreator({ id: groupOwner, name: data?.full_name || data?.name || '' })
      })
      .catch(() => null)
    return () => {
      live = false
    }
  }, [groupOwner, messages, creator.id])
  const select = (item) => {
    setMessages([])
    setThreadError('')
    setOpened(item)
    setActiveId(item.id)
    setItems((current) =>
      current.map((entry) => (entry.id === item.id ? { ...entry, unread_count: 0 } : entry)),
    )
  }
  const openConversation = (conversation) => {
    setNewOpen(false)
    select(conversation)
    loadList()
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
  const replyPrivately = async () => {
    const data = await api.post('/api/messages/direct', { user_id: groupOwner }).catch((failure) => {
      setThreadError(failure.message)
      return null
    })
    if (data) openConversation({ ...data, title: creator.name, kind: 'direct', can_post: true })
  }
  return (
    <>
      <PageHeader
        title="Messages"
        subtitle="Write to one person or to a group you are allowed to start"
        actions={
          <Button onClick={() => setNewOpen(true)}>
            <Plus size={16} />
            New message
          </Button>
        }
      />
      {listError && (
        <Alert tone="error" className="mb-5">
          {listError}
        </Alert>
      )}
      <div className="flex h-[70vh] min-h-[28rem] gap-4">
        <div className={cx('w-full md:w-80 md:shrink-0', activeId && 'hidden md:block')}>
          <ConversationList
            items={items}
            activeId={activeId}
            onSelect={select}
            search={search}
            onSearchChange={setSearch}
            loading={listLoading}
          />
        </div>
        <div className={cx('min-w-0 flex-1', !activeId && 'hidden md:block')}>
          {active ? (
            <div className="flex h-full min-h-0 flex-col rounded-xl border border-brand-100 bg-page">
              <div className="flex items-center gap-3 border-b border-brand-100 bg-surface px-3 py-3">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setActiveId(0)}
                  className="md:hidden"
                  aria-label="Back to the conversations"
                >
                  <ArrowLeft size={16} />
                </Button>
                {active.kind === 'group' ? (
                  <span className="inline-flex size-10 shrink-0 items-center justify-center rounded-full border border-brand-100 bg-brand-50 text-brand-600">
                    <Users size={18} />
                  </span>
                ) : (
                  <Avatar src={active.photo_url} name={active.title} />
                )}
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-brand-900">{active.title}</p>
                  <p className="text-xs text-brand-500">{conversationDetail(active)}</p>
                </div>
              </div>
              {threadError && (
                <Alert tone="error" className="m-3">
                  {threadError}
                </Alert>
              )}
              <div className="min-h-0 flex-1 overflow-y-auto">
                <MessageThread items={messages} meId={user?.id} loading={threadLoading} />
              </div>
              {active.can_post ? (
                <MessageComposer onSend={send} />
              ) : (
                <div className="flex flex-wrap items-center gap-3 border-t border-brand-100 bg-surface p-3">
                  <p className="text-sm text-brand-600">
                    Only the person who started this group can post here.
                  </p>
                  <Button variant="secondary" onClick={replyPrivately} className="ml-auto">
                    <Reply size={15} />
                    Reply privately{creator.name ? ` to ${creator.name}` : ''}
                  </Button>
                </div>
              )}
            </div>
          ) : (
            <EmptyState
              icon={Users}
              title="Pick a conversation"
              text="Choose one on the left or start a new message."
              className="h-full"
            />
          )}
        </div>
      </div>
      <NewConversationModal
        open={newOpen}
        onClose={() => setNewOpen(false)}
        role={user?.role}
        batch={user?.batch}
        section={user?.section}
        onOpened={openConversation}
      />
    </>
  )
}

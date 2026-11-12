export const BADGE_MAX = 99

export const badgeLabel = (count, max = BADGE_MAX) => {
  const value = Math.floor(Number(count) || 0)
  if (value <= 0) return ''
  return value > max ? `${max}+` : String(value)
}

export const EMPTY_NOTIFICATIONS = { items: [], unread: 0, loaded: false }

export const received = (data) => {
  const items = Array.isArray(data?.items) ? data.items : []
  const counted = Number(data?.unread)
  return {
    items,
    unread: Number.isFinite(counted) ? Math.max(0, counted) : items.filter((item) => !item.read).length,
    loaded: true,
  }
}

export const withRead = (state, ids) => {
  const wanted = new Set(ids)
  let changed = 0
  const items = state.items.map((item) => {
    if (!wanted.has(item.id) || item.read) return item
    changed += 1
    return { ...item, read: true }
  })
  if (changed === 0) return state
  return { ...state, items, unread: Math.max(0, state.unread - changed) }
}

export const withAllRead = (state) => ({
  ...state,
  items: state.items.map((item) => (item.read ? item : { ...item, read: true })),
  unread: 0,
})

export const withUnread = (state, unread) => {
  const value = Number(unread)
  if (unread === undefined || unread === null || !Number.isFinite(value)) return state
  return { ...state, unread: Math.max(0, value) }
}

const split = (page) => {
  const [path, search = ''] = String(page || '').split('#')[0].split('?')
  return { path: path.replace(/\/+$/, '') || '/', params: new URLSearchParams(search) }
}

export const samePage = (page, target) => {
  const given = split(page)
  const wanted = split(target)
  if (given.path !== wanted.path) return false
  for (const [key, value] of wanted.params) {
    if (given.params.get(key) !== value) return false
  }
  return true
}

const ITEM_PARAMS = ['item', 'c', 'highlight', 'session', 'course', 'date']

const ITEM_PATH = /\/\d+$/

export const pointsAtAnItem = (page) => {
  const { path, params } = split(page)
  return ITEM_PATH.test(path) || ITEM_PARAMS.some((key) => params.get(key))
}

export const unreadOnPage = (state, page, targetOf) =>
  state.items
    .filter((item) => !item.read && (item.link?.ref || item.link?.date))
    .filter((item) => samePage(page, targetOf(item)))
    .map((item) => item.id)

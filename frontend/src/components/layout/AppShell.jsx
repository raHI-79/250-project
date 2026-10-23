import { useMemo, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import Sidebar from './Sidebar'
import TabBar from './TabBar'
import TopBar from './TopBar'
import PhotoReminder from './PhotoReminder'
import { ChromeContext } from '../../context/chrome'
import cx from '../../lib/cx'
import { useUnreadMessages } from '../../lib/hooks'
import { MESSAGES_BADGE } from '../../lib/nav'

export default function AppShell({ groups, tabs = [], showBell = true }) {
  const location = useLocation()
  const unread = useUnreadMessages()
  const [menuOpen, setMenuOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [collapsed, setCollapsed] = useState(false)
  const chrome = useMemo(
    () => ({ title, setTitle, collapsed, setCollapsed }),
    [title, collapsed],
  )
  const withCount = (item) => (item.badge === MESSAGES_BADGE ? { ...item, count: unread } : item)
  const sections = groups.map((group) => ({ ...group, items: group.items.map(withCount) }))
  const bar = tabs.map(withCount)
  return (
    <ChromeContext.Provider value={chrome}>
      <div className="flex min-h-dvh bg-page">
        <Sidebar groups={sections} open={menuOpen} onClose={() => setMenuOpen(false)} />
        <div className="flex min-w-0 flex-1 flex-col">
          <TopBar
            title={title}
            collapsed={collapsed}
            onOpenMenu={() => setMenuOpen(true)}
            showBell={showBell}
          />
          <main
            className={cx(
              'min-w-0 flex-1 px-4 pt-5 sm:px-6 lg:px-8 lg:pb-10',
              bar.length > 0 ? 'pb-[calc(4.5rem+env(safe-area-inset-bottom,0px))]' : 'pb-8',
            )}
          >
            <div key={location.pathname} className="animate-fade-in-up mx-auto max-w-6xl">
              <PhotoReminder />
              <Outlet />
            </div>
          </main>
          <TabBar tabs={bar} />
        </div>
      </div>
    </ChromeContext.Provider>
  )
}

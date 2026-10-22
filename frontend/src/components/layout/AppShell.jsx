import { useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import Header from './Header'
import Sidebar from './Sidebar'
import PhotoReminder from './PhotoReminder'
import PushBanner from './PushBanner'
import { useUnreadMessages } from '../../lib/hooks'
import { MESSAGES_BADGE } from '../../lib/nav'

export default function AppShell({ groups, tone = 'brand', showBell = true }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const location = useLocation()
  const unread = useUnreadMessages()
  const sections = groups.map((group) => ({
    ...group,
    items: group.items.map((item) =>
      item.badge === MESSAGES_BADGE ? { ...item, count: unread } : item,
    ),
  }))
  return (
    <div className="flex min-h-screen bg-page">
      <Sidebar
        groups={sections}
        tone={tone}
        open={menuOpen}
        onClose={() => setMenuOpen(false)}
      />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header onOpenMenu={() => setMenuOpen(true)} showBell={showBell} />
        <main className="min-w-0 flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div key={location.pathname} className="animate-fade-in-up mx-auto max-w-6xl">
            <PushBanner />
            <PhotoReminder />
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}

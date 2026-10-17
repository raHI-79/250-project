import { useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import Header from './Header'
import Sidebar from './Sidebar'
import CampusBanner from './CampusBanner'

export default function AppShell({ items, showBell = true }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const location = useLocation()
  return (
    <div className="flex min-h-screen bg-page">
      <Sidebar items={items} open={menuOpen} onClose={() => setMenuOpen(false)} />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header onOpenMenu={() => setMenuOpen(true)} showBell={showBell} />
        <main className="min-w-0 flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div key={location.pathname} className="animate-fade-in-up mx-auto max-w-6xl">
            <CampusBanner />
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}

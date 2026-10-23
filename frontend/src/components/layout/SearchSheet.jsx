import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Clock, Search, SearchX } from 'lucide-react'
import Avatar from '../ui/Avatar'
import Chip from '../ui/Chip'
import EmptyState from '../ui/EmptyState'
import ListRow from '../ui/ListRow'
import Sheet from '../ui/Sheet'
import { SkeletonRows } from '../ui/Skeleton'
import { useAuth } from '../../context/auth'
import { useDebounced } from '../../lib/hooks'
import {
  MIN_TERM,
  forgetSearches,
  recentSearches,
  rememberSearch,
  searchEverything,
} from '../../lib/search'

export default function SearchSheet({ open, onClose }) {
  const { role } = useAuth()
  const navigate = useNavigate()
  const field = useRef(null)
  const [term, setTerm] = useState('')
  const [sections, setSections] = useState([])
  const [loading, setLoading] = useState(false)
  const [recent, setRecent] = useState(recentSearches)
  const settled = useDebounced(term, 250)
  useEffect(() => {
    if (!open) return undefined
    setRecent(recentSearches())
    const timer = window.setTimeout(() => field.current?.focus(), 80)
    return () => window.clearTimeout(timer)
  }, [open])
  useEffect(() => {
    if (!open) {
      setTerm('')
      setSections([])
    }
  }, [open])
  useEffect(() => {
    if (!open) return undefined
    const value = settled.trim()
    if (value.length < MIN_TERM) {
      setSections([])
      setLoading(false)
      return undefined
    }
    let live = true
    setLoading(true)
    searchEverything(role, value)
      .then((result) => {
        if (live) setSections(result)
      })
      .catch(() => {
        if (live) setSections([])
      })
      .finally(() => {
        if (live) setLoading(false)
      })
    return () => {
      live = false
    }
  }, [open, settled, role])
  const go = (item) => {
    setRecent(rememberSearch(term))
    onClose?.()
    navigate(item.to)
  }
  const value = settled.trim()
  return (
    <Sheet open={open} onClose={onClose} title="Search" full>
      <div className="sticky top-0 z-10 -mx-5 bg-surface px-5 pb-3">
        <div className="flex h-12 items-center gap-2 rounded-xl border border-line-strong bg-wash px-3.5">
          <Search size={18} className="shrink-0 text-muted" />
          <input
            ref={field}
            type="search"
            value={term}
            onChange={(event) => setTerm(event.target.value)}
            placeholder="Courses, people, notices, pages"
            aria-label="Search"
            className="min-w-0 flex-1 bg-transparent text-base text-ink outline-none placeholder:text-faint"
          />
        </div>
      </div>
      {value.length < MIN_TERM ? (
        recent.length > 0 ? (
          <div className="pt-2">
            <div className="flex items-center justify-between gap-3">
              <p className="label-caps">Recent</p>
              <button
                type="button"
                onClick={() => setRecent(forgetSearches())}
                className="text-xs font-semibold text-brand-600"
              >
                Clear
              </button>
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              {recent.map((entry) => (
                <Chip key={entry} as="button" icon={Clock} onClick={() => setTerm(entry)}>
                  {entry}
                </Chip>
              ))}
            </div>
          </div>
        ) : (
          <p className="pt-4 text-sm text-muted">
            Type at least {MIN_TERM} characters to search courses, teachers, classmates, notices and
            pages.
          </p>
        )
      ) : loading ? (
        <SkeletonRows rows={5} className="mt-2" />
      ) : sections.length === 0 ? (
        <EmptyState
          icon={SearchX}
          title="Nothing found"
          text={`No course, person, notice or page matches "${value}".`}
          className="mt-2"
        />
      ) : (
        <div className="flex flex-col gap-5 pt-1">
          {sections.map((section) => (
            <section key={section.label}>
              <p className="label-caps">{section.label}</p>
              <div className="mt-2 divide-y divide-line overflow-hidden rounded-card border border-line">
                {section.items.map((item) => (
                  <ListRow
                    key={item.id}
                    title={item.title}
                    subtitle={item.subtitle}
                    icon={item.icon}
                    leading={
                      item.photo !== undefined ? (
                        <Avatar src={item.photo} name={item.title} size="sm" />
                      ) : undefined
                    }
                    onClick={() => go(item)}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </Sheet>
  )
}

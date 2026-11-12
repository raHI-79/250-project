import { Search } from 'lucide-react'
import Avatar from '../ui/Avatar'
import HeaderArt from '../ui/HeaderArt'
import cx from '../../lib/cx'
import { dots } from '../../lib/dots'
import { useChrome } from '../../context/chrome'
import { hourIn, longDate, shortDate } from '../../lib/time'

const greeting = (hour) => {
  if (hour < 12) return 'Good morning'
  if (hour < 17) return 'Good afternoon'
  return 'Good evening'
}

export default function TodayHero({ name, regNo, photo, meta, now = new Date(), className }) {
  const chrome = useChrome()
  const first = String(name || '')
    .trim()
    .split(/\s+/)[0]
  return (
    <section
      data-hero
      className={cx(
        'relative isolate -mx-4 -mt-5 rounded-b-sheet bg-header px-4 pt-2.5 pb-6 sm:-mx-6 sm:px-6 sm:pt-5 sm:pb-14 lg:-mx-8 lg:px-8',
        className,
      )}
    >
      <HeaderArt variant="hero" />
      <div className="flex items-center gap-3">
        <Avatar
          src={photo}
          name={name}
          size="md"
          className="shrink-0 ring-2 ring-white/40 sm:size-16 sm:text-base"
        />
        <div className="min-w-0 flex-1">
          <h1 className="font-display truncate text-lg leading-tight text-white sm:text-title">
            {greeting(hourIn(now))}
            {first ? <>,&nbsp;{first}</> : null}
          </h1>
          <p className="truncate text-xs font-semibold text-white/95 sm:mt-1 sm:text-sm">
            <span className="sm:hidden">{dots(shortDate(now), meta)}</span>
            <span className="hidden sm:inline">{dots(longDate(now), meta)}</span>
          </p>
          {regNo && (
            <p className="hidden truncate text-xs font-semibold text-white/95 tabular-nums sm:block">
              {dots(name, regNo)}
            </p>
          )}
        </div>
      </div>
      {chrome?.openSearch && (
        <button
          type="button"
          onClick={chrome.openSearch}
          className="press mt-2.5 flex h-9 w-full items-center gap-2 rounded-input bg-[rgb(4_40_66/0.3)] px-3 text-left text-xs font-semibold text-white ring-1 ring-white/25 transition-colors ring-inset hover:bg-[rgb(4_40_66/0.42)] sm:mt-4 sm:h-12 sm:gap-2.5 sm:px-3.5 sm:text-sm"
        >
          <Search size={16} className="shrink-0" />
          Search courses, people, notices
        </button>
      )}
    </section>
  )
}

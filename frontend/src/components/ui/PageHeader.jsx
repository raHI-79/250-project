import { useEffect, useRef } from 'react'
import cx from '../../lib/cx'
import { useChrome } from '../../context/chrome'
import HeaderArt from './HeaderArt'

export default function PageHeader({
  eyebrow,
  title,
  subtitle,
  actions,
  tabs,
  date,
  className,
}) {
  const chrome = useChrome()
  const anchor = useRef(null)
  const setTitle = chrome?.setTitle
  const setCollapsed = chrome?.setCollapsed
  useEffect(() => {
    if (!setTitle) return undefined
    setTitle(typeof title === 'string' ? title : '')
    return () => setTitle('')
  }, [setTitle, title])
  useEffect(() => {
    if (!setCollapsed) return undefined
    const node = anchor.current
    if (!node || typeof IntersectionObserver === 'undefined') return undefined
    const observer = new IntersectionObserver(([entry]) => setCollapsed(!entry.isIntersecting), {
      threshold: 0,
    })
    observer.observe(node)
    return () => {
      observer.disconnect()
      setCollapsed(false)
    }
  }, [setCollapsed])
  return (
    <div className={cx('mb-5', className)}>
      <div
        data-page-header
        className="relative isolate -mx-4 -mt-5 rounded-b-sheet bg-header px-4 pt-3 pb-3 sm:-mx-6 sm:px-6 sm:pt-5 sm:pb-5 lg:-mx-8 lg:px-8"
      >
        <HeaderArt variant="page" />
        <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
          <div className="min-w-0 flex-1">
            {eyebrow && (
              <p className="hidden text-2xs font-extrabold tracking-[0.12em] text-white/95 uppercase sm:block">
                {eyebrow}
              </p>
            )}
            <h1
              ref={anchor}
              className="font-display wrap-long text-section leading-tight text-white sm:mt-1 sm:text-title-lg"
            >
              {title}
            </h1>
            {subtitle && (
              <p className="mt-2 hidden max-w-3xl wrap-long text-sm text-white/95 sm:block">
                {subtitle}
              </p>
            )}
          </div>
          {date && <div className="shrink-0 pt-1">{date}</div>}
        </div>
        {tabs && <div className="mt-2.5 sm:mt-4">{tabs}</div>}
      </div>
      {actions && <div className="mt-4 flex flex-wrap gap-2 sm:justify-end">{actions}</div>}
    </div>
  )
}

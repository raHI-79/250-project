import { useEffect, useRef } from 'react'
import cx from '../../lib/cx'
import { useChrome } from '../../context/chrome'

export default function PageHeader({ eyebrow, title, subtitle, actions, className }) {
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
    const observer = new IntersectionObserver(
      ([entry]) => setCollapsed(!entry.isIntersecting),
      { threshold: 0 },
    )
    observer.observe(node)
    return () => {
      observer.disconnect()
      setCollapsed(false)
    }
  }, [setCollapsed])
  return (
    <div
      className={cx('mb-5 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between', className)}
    >
      <div className="min-w-0">
        {eyebrow && <p className="label-caps">{eyebrow}</p>}
        <h1
          ref={anchor}
          className="font-display mt-1 text-title leading-tight font-bold text-ink sm:text-title-lg"
        >
          {title}
        </h1>
        {subtitle && <p className="mt-2 max-w-3xl text-sm text-muted">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  )
}

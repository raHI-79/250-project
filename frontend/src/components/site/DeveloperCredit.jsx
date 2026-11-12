import { Link } from 'react-router-dom'
import cx from '../../lib/cx'
import { DEVELOPER } from '../../lib/siteContent'

export default function DeveloperCredit({ tone = 'dark', className }) {
  return (
    <p
      data-developer-credit
      className={cx('text-xs', tone === 'dark' ? 'text-white/60' : 'text-muted', className)}
    >
      <Link
        to={DEVELOPER.path}
        className={cx(
          'font-semibold underline-offset-2 hover:underline',
          tone === 'dark' ? 'text-white/80 hover:text-white' : 'text-brand-600',
        )}
      >
        {DEVELOPER.label}
      </Link>
    </p>
  )
}

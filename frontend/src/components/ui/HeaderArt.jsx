import cx from '../../lib/cx'

const VARIANTS = {
  hero: 'header-art-hero',
  page: 'header-art-page',
  panel: 'header-art-panel',
}

export default function HeaderArt({ variant = 'page', className }) {
  return (
    <span
      aria-hidden="true"
      data-header-art={variant}
      className={cx('header-art', VARIANTS[variant] || VARIANTS.page, className)}
    />
  )
}

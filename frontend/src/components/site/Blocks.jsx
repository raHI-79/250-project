import cx from '../../lib/cx'
import { assetURL } from '../../lib/native'
import { isSafeImage, isSafeLink } from '../../lib/siteContent'
import { formatDate } from '../../lib/time'
import RichText, { SafeLink } from './RichText'

function Picture({ src, alt, className }) {
  if (!isSafeImage(src)) return null
  return <img src={assetURL(src)} alt={alt || ''} loading="lazy" className={className} />
}

function Block({ block }) {
  if (block.type === 'heading') {
    return (
      <h3 data-block="heading" className="font-display wrap-long text-xl font-bold text-ink">
        {block.text}
      </h3>
    )
  }
  if (block.type === 'text') {
    return (
      <div data-block="text">
        <RichText text={block.text} className="text-ink-soft" />
      </div>
    )
  }
  if (block.type === 'image') {
    return (
      <figure data-block="image">
        <Picture
          src={block.image_url}
          alt={block.alt}
          className="max-h-[32rem] w-full rounded-card object-cover"
        />
      </figure>
    )
  }
  if (block.type === 'image_text') {
    return (
      <div
        data-block="image_text"
        className={cx(
          'flex flex-col gap-5 sm:items-center',
          block.side === 'right' ? 'sm:flex-row-reverse' : 'sm:flex-row',
        )}
      >
        <Picture
          src={block.image_url}
          alt={block.alt}
          className="w-full rounded-card object-cover sm:w-1/2"
        />
        <RichText text={block.text} className="min-w-0 flex-1 text-ink-soft" />
      </div>
    )
  }
  if (block.type === 'button') {
    if (!isSafeLink(block.url)) return null
    return (
      <div data-block="button">
        <SafeLink
          to={block.url}
          className="press inline-flex min-h-11 items-center rounded-xl bg-brand-600 px-5 text-sm font-semibold text-white hover:bg-brand-700"
        >
          {block.label}
        </SafeLink>
      </div>
    )
  }
  if (block.type === 'cards') {
    return (
      <div data-block="cards" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {(block.cards || []).map((card, index) => (
          <article
            key={`${card.title}-${index}`}
            className="overflow-hidden rounded-card border border-line bg-surface"
          >
            <Picture src={card.image_url} alt="" className="h-40 w-full object-cover" />
            <div className="p-4">
              {card.date && <p className="text-xs font-semibold text-brand-600">{formatDate(card.date)}</p>}
              <h4 className="font-display mt-1 wrap-long text-base font-semibold text-ink">
                {card.title}
              </h4>
              <RichText text={card.text} className="mt-2 text-muted" />
            </div>
          </article>
        ))}
      </div>
    )
  }
  if (block.type === 'faq') {
    return (
      <div data-block="faq" className="flex flex-col gap-2">
        {(block.faq || []).map((entry, index) => (
          <details
            key={`${entry.question}-${index}`}
            className="rounded-card border border-line bg-surface px-4 py-3"
          >
            <summary className="cursor-pointer wrap-long text-sm font-semibold text-ink">
              {entry.question}
            </summary>
            <RichText text={entry.answer} className="mt-2 text-ink-soft" />
          </details>
        ))}
      </div>
    )
  }
  return null
}

export default function Blocks({ blocks, className }) {
  const shown = Array.isArray(blocks) ? blocks : []
  if (shown.length === 0) return null
  return (
    <div className={cx('flex flex-col gap-6', className)}>
      {shown.map((block, index) => (
        <Block key={`${block.type}-${index}`} block={block} />
      ))}
    </div>
  )
}

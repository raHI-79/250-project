import { Link } from 'react-router-dom'
import cx from '../../lib/cx'
import { isSitePath, parseRich } from '../../lib/siteContent'

export function SafeLink({ to, className, children, ...rest }) {
  if (isSitePath(to)) {
    return (
      <Link to={to} className={className} {...rest}>
        {children}
      </Link>
    )
  }
  return (
    <a href={to} target="_blank" rel="noopener noreferrer" className={className} {...rest}>
      {children}
    </a>
  )
}

function Inline({ parts }) {
  return parts.map((part, index) => {
    const key = `${part.type}-${index}`
    if (part.type === 'bold') return <strong key={key}>{part.text}</strong>
    if (part.type === 'italic') return <em key={key}>{part.text}</em>
    if (part.type === 'link') {
      return (
        <SafeLink key={key} to={part.url} className="font-semibold text-brand-600 underline">
          {part.text}
        </SafeLink>
      )
    }
    return <span key={key}>{part.text}</span>
  })
}

export default function RichText({ text, className }) {
  const groups = parseRich(text)
  if (groups.length === 0) return null
  return (
    <div className={cx('flex flex-col gap-3 text-sm leading-relaxed', className)}>
      {groups.map((group, index) =>
        group.type === 'list' ? (
          <ul key={`list-${index}`} className="list-disc space-y-1 pl-5">
            {group.items.map((item, position) => (
              <li key={`item-${position}`} className="wrap-long">
                <Inline parts={item} />
              </li>
            ))}
          </ul>
        ) : (
          <p key={`paragraph-${index}`} className="wrap-long">
            {group.lines.map((line, position) => (
              <span key={`line-${position}`}>
                {position > 0 && <br />}
                <Inline parts={line} />
              </span>
            ))}
          </p>
        ),
      )}
    </div>
  )
}

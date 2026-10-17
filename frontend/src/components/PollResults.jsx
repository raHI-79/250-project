import cx from '../lib/cx'

const percentOf = (votes, total) => (total > 0 ? Math.round((votes / total) * 100) : 0)

export default function PollResults({ options, total, myOptionId = null }) {
  return (
    <ul className="mt-4 flex flex-col gap-3">
      {options.map((option) => {
        const mine = option.id === myOptionId
        const percent = percentOf(option.votes, total)
        return (
          <li key={option.id} className="flex flex-col gap-1">
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className={cx('text-brand-800', mine && 'font-semibold text-brand-700')}>
                {option.text}
                {mine && <span className="ml-2 text-xs text-brand-500">Your vote</span>}
              </span>
              <span className="tabular-nums text-brand-500">
                {option.votes} · {percent}%
              </span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-brand-50">
              <div
                style={{ width: `${percent}%` }}
                className={cx(
                  'h-full rounded-full transition-all duration-700 ease-out',
                  mine ? 'bg-brand-600' : 'bg-brand-300',
                )}
              />
            </div>
          </li>
        )
      })}
    </ul>
  )
}

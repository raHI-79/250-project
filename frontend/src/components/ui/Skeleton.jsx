import cx from '../../lib/cx'

export function Skeleton({ className }) {
  return <div className={cx('shimmer-bg rounded-sm', className)} />
}

export function SkeletonLines({ lines = 3, className }) {
  return (
    <div className={cx('flex flex-col gap-2', className)}>
      {Array.from({ length: lines }).map((_, index) => (
        <Skeleton
          key={index}
          className={cx('h-3.5', index === lines - 1 ? 'w-2/3' : 'w-full')}
        />
      ))}
    </div>
  )
}

export function SkeletonCard({ className }) {
  return (
    <div className={cx('border border-line bg-surface p-5', className)}>
      <Skeleton className="h-4 w-1/3" />
      <div className="mt-4">
        <SkeletonLines lines={3} />
      </div>
    </div>
  )
}

export function SkeletonTable({ rows = 5, columns = 4, className }) {
  return (
    <div className={cx('overflow-hidden border border-line bg-surface', className)}>
      <div className="flex gap-4 border-b border-line px-4 py-3">
        {Array.from({ length: columns }).map((_, index) => (
          <Skeleton key={index} className="h-3.5 flex-1" />
        ))}
      </div>
      {Array.from({ length: rows }).map((_, row) => (
        <div key={row} className="flex gap-4 border-b border-line px-4 py-3 last:border-b-0">
          {Array.from({ length: columns }).map((_, column) => (
            <Skeleton key={column} className="h-3.5 flex-1" />
          ))}
        </div>
      ))}
    </div>
  )
}

export default Skeleton

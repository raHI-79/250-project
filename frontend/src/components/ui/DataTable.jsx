import cx from '../../lib/cx'

export default function DataTable({ className, children }) {
  return (
    <div
      className={cx(
        'w-full overflow-x-auto rounded-xl border border-brand-100 bg-surface',
        className,
      )}
    >
      <table className="w-full min-w-max border-collapse text-left text-sm">{children}</table>
    </div>
  )
}

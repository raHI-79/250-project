import cx from '../../lib/cx'

export default function DataTable({ className, children }) {
  return (
    <div className={cx('w-full overflow-x-auto border border-line bg-surface', className)}>
      <table className="w-full min-w-max border-collapse text-left text-sm [&_tbody_td:first-child]:sticky [&_tbody_td:first-child]:left-0 [&_tbody_td:first-child]:bg-surface [&_thead_th:first-child]:sticky [&_thead_th:first-child]:left-0 [&_thead_th:first-child]:z-10 [&_thead_th:first-child]:bg-wash">
        {children}
      </table>
    </div>
  )
}

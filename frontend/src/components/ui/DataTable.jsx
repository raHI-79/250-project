import { Children, cloneElement, isValidElement } from 'react'
import cx from '../../lib/cx'

const textOf = (node) => {
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(textOf).join('')
  if (isValidElement(node)) return textOf(node.props.children)
  return ''
}

const headerLabels = (children) => {
  let labels = []
  Children.forEach(children, (child) => {
    if (!isValidElement(child) || child.type !== 'thead') return
    Children.forEach(child.props.children, (row) => {
      if (!isValidElement(row)) return
      const cells = []
      Children.forEach(row.props.children, (cell) => {
        if (isValidElement(cell)) cells.push(textOf(cell.props.children))
      })
      if (cells.length > 0) labels = cells
    })
  })
  return labels
}

const labelRow = (row, labels) => {
  let index = 0
  const cells = Children.map(row.props.children, (cell) => {
    if (!isValidElement(cell) || cell.type !== 'td') return cell
    const span = Number(cell.props.colSpan) || 1
    const label = span === 1 ? labels[index] : ''
    index += span
    if (!label) return cell
    return cloneElement(cell, { 'data-label': label })
  })
  return cloneElement(row, undefined, cells)
}

const labelBody = (children, labels) =>
  Children.map(children, (child) => {
    if (!isValidElement(child) || child.type !== 'tbody') return child
    const rows = Children.map(child.props.children, (row) =>
      isValidElement(row) && row.type === 'tr' ? labelRow(row, labels) : row,
    )
    return cloneElement(child, undefined, rows)
  })

export default function DataTable({ className, children }) {
  const labels = headerLabels(children)
  return (
    <div
      className={cx(
        'w-full rounded-card md:glass-card md:overflow-x-auto',
        className,
      )}
    >
      <table className="table-stack w-full border-collapse text-left text-sm md:min-w-max md:[&_tbody_td:first-child]:sticky md:[&_tbody_td:first-child]:left-0 md:[&_tbody_td:first-child]:bg-surface md:[&_thead_th:first-child]:sticky md:[&_thead_th:first-child]:left-0 md:[&_thead_th:first-child]:z-10 md:[&_thead_th:first-child]:bg-wash">
        {labels.length > 0 ? labelBody(children, labels) : children}
      </table>
    </div>
  )
}

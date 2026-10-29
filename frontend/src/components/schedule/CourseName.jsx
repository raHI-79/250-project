import cx from '../../lib/cx'
import { courseCode, courseLine } from '../../lib/schedule'

export default function CourseName({ item, className, codeClassName, as: Tag = 'span' }) {
  const code = courseCode(item)
  return (
    <Tag className={cx('block min-w-0', className)}>
      <span className="wrap-long block">{courseLine(item)}</span>
      {code && (
        <span className={cx('wrap-long mt-0.5 block text-2xs font-medium text-muted', codeClassName)}>
          {code}
        </span>
      )}
    </Tag>
  )
}

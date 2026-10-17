import cx from '../../lib/cx'
import CampusBackdrop from './CampusBackdrop'

export default function AuthScreen({ images, rotateAfter, overlay, className, children }) {
  return (
    <div className="relative flex min-h-screen items-center justify-center px-4 py-10">
      <CampusBackdrop
        images={images}
        rotateAfter={rotateAfter}
        overlay={overlay}
        className="fixed"
      />
      <div className={cx('relative w-full max-w-sm', className)}>{children}</div>
    </div>
  )
}

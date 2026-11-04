import cx from '../../lib/cx'

export default function FormSection({ title, description, children, className }) {
  return (
    <section className={cx('flex flex-col gap-4', className)}>
      <div>
        <h3 className="label-caps">{title}</h3>
        {description && <p className="mt-1 text-xs text-muted">{description}</p>}
      </div>
      {children}
    </section>
  )
}

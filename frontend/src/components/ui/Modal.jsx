import Sheet from './Sheet'

export default function Modal({ open, onClose, title, children, footer, className }) {
  return (
    <Sheet open={open} onClose={onClose} title={title} footer={footer} className={className}>
      {children}
    </Sheet>
  )
}

import {
  File,
  FileArchive,
  FileSpreadsheet,
  FileText,
  Link2,
  Presentation,
} from 'lucide-react'
import cx from '../lib/cx'
import {
  extensionOf,
  fileNameOf,
  formatFileSize,
  isImageFile,
  isPdfFile,
  linkLabel,
} from '../lib/files'

const SHEETS = ['xls', 'xlsx', 'csv']

const SLIDES = ['ppt', 'pptx']

const DOCUMENTS = ['pdf', 'doc', 'docx', 'txt']

const iconFor = (name) => {
  const extension = extensionOf(name)
  if (SHEETS.includes(extension)) return FileSpreadsheet
  if (SLIDES.includes(extension)) return Presentation
  if (extension === 'zip') return FileArchive
  if (DOCUMENTS.includes(extension)) return FileText
  return File
}

const chipClass =
  'inline-flex max-w-full items-start gap-2 rounded-md border border-line-strong bg-surface px-3 py-2 text-sm text-ink-soft transition-colors hover:border-brand-300 hover:bg-wash'

export default function AttachmentView({
  fileUrl,
  fileName,
  fileMime,
  fileSize,
  linkUrl,
  className,
}) {
  if (!fileUrl && !linkUrl) return null
  const name = fileNameOf(fileUrl, fileName)
  const Icon = iconFor(name)
  const inline = isImageFile(fileMime, name) || isPdfFile(fileMime, name)
  return (
    <div className={cx('flex flex-col items-start gap-2', className)}>
      {fileUrl && isImageFile(fileMime, name) && (
        <a href={fileUrl} target="_blank" rel="noreferrer" className="block">
          <img
            src={fileUrl}
            alt={name}
            className="max-h-52 max-w-full rounded-md border border-line object-cover"
          />
        </a>
      )}
      {fileUrl && !isImageFile(fileMime, name) && (
        <a href={fileUrl} target="_blank" rel="noreferrer" className={chipClass}>
          <Icon size={16} className="shrink-0 text-muted" />
          <span className="wrap-long">{name}</span>
          {fileSize > 0 && (
            <span className="shrink-0 text-xs text-faint">{formatFileSize(fileSize)}</span>
          )}
          <span className="shrink-0 text-xs font-medium text-brand-600">
            {inline ? 'Open' : 'Download'}
          </span>
        </a>
      )}
      {linkUrl && (
        <a href={linkUrl} target="_blank" rel="noreferrer" className={chipClass}>
          <Link2 size={16} className="shrink-0 text-muted" />
          <span className="wrap-long">{linkLabel(linkUrl)}</span>
        </a>
      )}
    </div>
  )
}

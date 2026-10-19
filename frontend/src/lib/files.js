export const MAX_FILE_MB = 10

export const MAX_FILE_BYTES = MAX_FILE_MB * 1024 * 1024

export const ALLOWED_EXTENSIONS = [
  'pdf',
  'jpg',
  'jpeg',
  'png',
  'webp',
  'gif',
  'doc',
  'docx',
  'xls',
  'xlsx',
  'ppt',
  'pptx',
  'csv',
  'txt',
  'zip',
]

export const FILE_ACCEPT = ALLOWED_EXTENSIONS.map((entry) => `.${entry}`).join(',')

export const FILE_TYPE_MESSAGE = 'This file type is not allowed.'

export const FILE_SIZE_MESSAGE = `The file must be at most ${MAX_FILE_MB} MB.`

export const LINK_MESSAGE = 'Link must be a full link that starts with http:// or https://.'

const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'webp', 'gif']

const LINK_PATTERN = /^https?:\/\/\S+$/i

export const extensionOf = (name) => {
  const text = String(name || '')
    .trim()
    .toLowerCase()
  const dot = text.lastIndexOf('.')
  return dot === -1 ? '' : text.slice(dot + 1)
}

export const fileError = (file) => {
  if (!file) return ''
  if (!ALLOWED_EXTENSIONS.includes(extensionOf(file.name))) return FILE_TYPE_MESSAGE
  if (file.size > MAX_FILE_BYTES) return FILE_SIZE_MESSAGE
  return ''
}

export const formatFileSize = (bytes) => {
  const size = Number(bytes) || 0
  if (size <= 0) return ''
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${Math.round(size / 1024)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

export const isImageFile = (mime, name) =>
  String(mime || '').startsWith('image/') || IMAGE_EXTENSIONS.includes(extensionOf(name))

export const isPdfFile = (mime, name) =>
  String(mime || '') === 'application/pdf' || extensionOf(name) === 'pdf'

export const linkError = (value) => {
  const text = String(value || '').trim()
  if (!text) return ''
  return LINK_PATTERN.test(text) ? '' : LINK_MESSAGE
}

export const linkLabel = (value) => {
  const text = String(value || '').trim()
  try {
    const url = new URL(text)
    const tail = url.pathname === '/' ? '' : url.pathname
    return `${url.host}${tail}`
  } catch {
    return text
  }
}

export const fileNameOf = (url, fallback) => {
  const text = String(fallback || '').trim()
  if (text) return text
  const parts = String(url || '').split('/')
  return parts[parts.length - 1] || 'attachment'
}

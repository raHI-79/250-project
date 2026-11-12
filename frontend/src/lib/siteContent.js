export const DEVELOPER = Object.freeze({
  name: 'CoordiNet',
  label: 'Developed by CoordiNet',
  path: '/about-coordinet',
  people: Object.freeze(['Ratul Islam', 'Rakibul Hasan']),
})

export const BLOCK_TYPES = [
  { value: 'heading', label: 'Heading' },
  { value: 'text', label: 'Text' },
  { value: 'image', label: 'Image' },
  { value: 'image_text', label: 'Image and text' },
  { value: 'button', label: 'Button' },
  { value: 'cards', label: 'Cards (news or events)' },
  { value: 'faq', label: 'Questions and answers' },
]

export const SECTION_LABELS = {
  stats: 'Numbers of the department',
  about: 'About the department',
  notices: 'Notices preview',
  app: 'Download the Android app',
}

export const CUSTOM_PREFIX = 'custom:'

export const RESERVED_SLUGS = [
  'student', 'teacher', 'cr', 'office-portal', 'control-room', 'profile', 'people', 'api',
  'uploads', 'download', 'p', 'about-coordinet', 'login', 'register', 'admin', 'brand', 'campus',
  'assets', 'index', 'sw', 'manifest',
]

export const DEFAULT_CONTENT = Object.freeze({
  header: {
    site_name: 'SUST CSE',
    tagline: 'Department of Computer Science and Engineering',
    logo_url: '/brand/sust-logo.png',
    menu: [],
  },
  home: {
    hero_eyebrow: 'Established 1991 · Sylhet, Bangladesh',
    hero_title: 'Class and exam coordination for the CSE department',
    hero_text:
      'One record for the weekly routine, rescheduled classes, exams and notices. Class representatives draft the plan, teachers confirm it, and every student reads the published result.',
    slides: [
      { image_url: '/campus/hero-oldGate.jpg', caption: 'SUST Iconic Old Gate, Kumargaon' },
      { image_url: '/campus/hero-building.jpg', caption: 'IICT Building, SUST campus' },
      { image_url: '/campus/hero-monument.jpg', caption: 'Shaheed Minar, SUST campus' },
      { image_url: '/campus/hero-entrance.jpg', caption: 'University main entrance, Kumargaon' },
    ],
    primary_cta: { label: 'Student sign in', url: '/student/login' },
    secondary_cta: { label: 'Teacher sign in', url: '/teacher/login' },
    about_title: 'Everything a class day needs, in one record',
    about_text:
      'The department runs its weekly routine, the classes and exams ahead, teacher confirmations, notices, materials, polls and messages here.',
    sections: [
      { key: 'stats', visible: true },
      { key: 'about', visible: true },
      { key: 'notices', visible: false },
      { key: 'app', visible: true },
    ],
    custom: [],
  },
  footer: {
    visible: true,
    address: 'Academic Building A, University Avenue\nKumargaon, Sylhet 3114\nBangladesh',
    phone: '',
    email: '',
    office_hours: '',
    note: 'For class and exam coordination inside the department only.',
    quick_links: [],
    social_links: [],
  },
  sign_in: { title: '', text: '' },
  seo: {
    title: 'SUST CSE · Class and exam coordination',
    description: 'The weekly routine, classes, exams and notices of the CSE department of SUST.',
    image_url: '',
  },
  pages: [],
})

export const cloneContent = (content) => JSON.parse(JSON.stringify(content || DEFAULT_CONTENT))

const list = (value) => (Array.isArray(value) ? value : [])

export const withDefaults = (content) => {
  const base = cloneContent(DEFAULT_CONTENT)
  if (!content || typeof content !== 'object') return base
  const merged = {
    header: { ...base.header, ...(content.header || {}) },
    home: { ...base.home, ...(content.home || {}) },
    footer: { ...base.footer, ...(content.footer || {}) },
    sign_in: { ...base.sign_in, ...(content.sign_in || {}) },
    seo: { ...base.seo, ...(content.seo || {}) },
    pages: list(content.pages),
  }
  merged.header.menu = list(merged.header.menu)
  merged.home.slides = list(merged.home.slides).length > 0 ? merged.home.slides : base.home.slides
  merged.home.sections = list(merged.home.sections)
  merged.home.custom = list(merged.home.custom)
  merged.footer.quick_links = list(merged.footer.quick_links)
  merged.footer.social_links = list(merged.footer.social_links)
  if (!merged.header.logo_url) merged.header.logo_url = base.header.logo_url
  return merged
}

export const isSitePath = (value) => {
  const link = String(value || '').trim()
  return link.startsWith('/') && !link.startsWith('//')
}

export const isSafeLink = (value) => {
  const link = String(value || '').trim()
  if (!link || link.length > 300 || /[\s"'<>\\]/.test(link)) return false
  const lowered = link.toLowerCase()
  if (lowered.startsWith('https://') || lowered.startsWith('http://')) {
    return link.length > 'https://'.length
  }
  return isSitePath(link)
}

export const isSafeImage = (value) => {
  const image = String(value || '').trim()
  if (!image || /[\s"'<>\\]/.test(image) || image.includes('..') || image.startsWith('//')) {
    return false
  }
  return ['/uploads/photos/', '/brand/', '/campus/'].some(
    (prefix) => image.startsWith(prefix) && image.length > prefix.length,
  )
}

const INLINE = /(\*\*[^*\n]+\*\*|\*[^*\n]+\*|\[[^\]\n]*\]\([^)\s]*\))/

const LINK = /^\[([^\]\n]*)\]\(([^)\s]*)\)$/

export const parseInline = (text) =>
  String(text || '')
    .split(INLINE)
    .filter((part) => part !== '')
    .map((part) => {
      if (part.startsWith('**') && part.endsWith('**') && part.length > 4) {
        return { type: 'bold', text: part.slice(2, -2) }
      }
      const link = LINK.exec(part)
      if (link) {
        if (!isSafeLink(link[2]) || !link[1]) return { type: 'text', text: link[1] || '' }
        return { type: 'link', text: link[1], url: link[2] }
      }
      if (part.startsWith('*') && part.endsWith('*') && part.length > 2) {
        return { type: 'italic', text: part.slice(1, -1) }
      }
      return { type: 'text', text: part }
    })

export const parseRich = (text) => {
  const groups = []
  let paragraph = []
  let items = []
  const flush = () => {
    if (paragraph.length > 0) groups.push({ type: 'paragraph', lines: paragraph.map(parseInline) })
    if (items.length > 0) groups.push({ type: 'list', items: items.map(parseInline) })
    paragraph = []
    items = []
  }
  String(text || '')
    .replace(/\r\n/g, '\n')
    .split('\n')
    .forEach((raw) => {
      const line = raw.trim()
      if (line === '') {
        flush()
        return
      }
      if (line.startsWith('- ')) {
        if (paragraph.length > 0) flush()
        items.push(line.slice(2).trim())
        return
      }
      if (items.length > 0) flush()
      paragraph.push(line)
    })
  flush()
  return groups
}

export const homeSections = (content) => {
  const home = withDefaults(content).home
  const custom = new Map(home.custom.map((section) => [`${CUSTOM_PREFIX}${section.id}`, section]))
  const known = (key) => Boolean(SECTION_LABELS[key]) || custom.has(key)
  return home.sections
    .filter((entry) => known(entry.key))
    .map((entry) => ({
      key: entry.key,
      visible: Boolean(entry.visible),
      label: SECTION_LABELS[entry.key] || custom.get(entry.key).title,
      custom: custom.get(entry.key) || null,
    }))
}

export const visibleSections = (content) => homeSections(content).filter((entry) => entry.visible)

export const pageBySlug = (content, slug) =>
  list(content?.pages).find((page) => page.slug === slug) || null

export const slugProblem = (slug, others = []) => {
  const value = String(slug || '')
  if (!/^[a-z0-9]+(-[a-z0-9]+)*$/.test(value) || value.length > 60) {
    return 'Use small letters, digits and dashes, like admission-2027.'
  }
  if (RESERVED_SLUGS.includes(value)) return 'The portal already uses this name. Pick another.'
  if (others.includes(value)) return 'Another page already has this address.'
  return ''
}

export const slugFrom = (title) =>
  String(title || '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60)

export const newId = (taken = []) => {
  for (let attempt = 0; attempt < 50; attempt += 1) {
    const id = `s${Math.random().toString(36).slice(2, 10)}`.replace(/[^a-z0-9]/g, '')
    if (id.length > 1 && !taken.includes(id)) return id
  }
  return `s${Date.now().toString(36)}`
}

export const newBlock = (type) => {
  if (type === 'heading') return { type, text: '' }
  if (type === 'text') return { type, text: '' }
  if (type === 'image') return { type, image_url: '', alt: '' }
  if (type === 'image_text') return { type, image_url: '', alt: '', text: '', side: 'left' }
  if (type === 'button') return { type, label: '', url: '' }
  if (type === 'cards') return { type, cards: [{ title: '', text: '', date: '', image_url: '' }] }
  return { type: 'faq', faq: [{ question: '', answer: '' }] }
}

export const move = (items, index, step) => {
  const target = index + step
  if (target < 0 || target >= items.length) return items
  const copy = [...items]
  const [entry] = copy.splice(index, 1)
  copy.splice(target, 0, entry)
  return copy
}

export const footerParts = (content) => {
  const footer = withDefaults(content).footer
  return {
    shown: footer.visible !== false,
    footer,
    credit: DEVELOPER,
  }
}

export const seoFor = (content, page) => {
  const seo = withDefaults(content).seo
  return {
    title: page ? page.seo_title || `${page.title} · ${seo.title}` : seo.title,
    description: page ? page.seo_description || seo.description : seo.description,
    image: seo.image_url || '',
  }
}

const CREDIT_WORDS = ['credit', 'developer', 'developed_by', 'developedby', 'coordinet', 'powered_by']

export const stripCredit = (value) => {
  if (Array.isArray(value)) return value.map(stripCredit)
  if (!value || typeof value !== 'object') return value
  return Object.fromEntries(
    Object.entries(value)
      .filter(([key]) => !CREDIT_WORDS.some((word) => key.toLowerCase().includes(word)))
      .map(([key, inner]) => [key, stripCredit(inner)]),
  )
}

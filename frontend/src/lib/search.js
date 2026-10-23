import api from './api'
import { navForRole } from './nav'
import { noticesFor, paths } from './paths'
import { readJson, writeJson } from './storage'

export const RECENT_KEY = 'sustcse.search.recent'

export const MAX_RECENT = 6

export const MIN_TERM = 2

const NOTICE_ENDPOINTS = {
  student: '/api/students/notices',
  cr: '/api/students/notices',
  teacher: '/api/teacher/notices',
}

const hit = (value, term) => String(value || '').toLowerCase().includes(term)

const roleSubtitle = (person) => {
  if (person.role === 'teacher') return person.designation || 'Teacher'
  if (person.batch) return `${person.batch} · Section ${person.section}`
  return 'Student'
}

const pageResults = (role, term) =>
  navForRole(role)
    .flatMap((group) => group.items.map((item) => ({ group: group.label, ...item })))
    .filter((item) => hit(item.label, term))
    .map((item) => ({
      id: `page-${item.label}`,
      title: item.label,
      subtitle: item.group || 'Page',
      to: item.to(),
      icon: item.icon,
    }))

const courseResults = async (term) => {
  const data = await api.get('/api/lookups/courses')
  return (data?.items || [])
    .filter((course) => hit(course.code, term) || hit(course.title, term))
    .slice(0, 6)
    .map((course) => ({
      id: `course-${course.id}`,
      title: course.code,
      subtitle: course.title,
      to: paths.student.courses(),
    }))
}

const teacherResults = async (term) => {
  const data = await api.get('/api/teachers-info', { query: { search: term } })
  return (data?.items || []).slice(0, 6).map((teacher) => ({
    id: `teacher-${teacher.user_id}`,
    title: teacher.name,
    subtitle: teacher.designation || 'Teacher',
    photo: teacher.photo_url,
    to: paths.person(teacher.user_id),
  }))
}

const peopleResults = async (term) => {
  const data = await api.get('/api/messages/contacts', { query: { search: term } })
  return (data?.items || [])
    .filter((person) => person.role !== 'teacher')
    .slice(0, 6)
    .map((person) => ({
      id: `person-${person.user_id}`,
      title: person.name,
      subtitle: roleSubtitle(person),
      photo: person.photo_url,
      to: paths.person(person.user_id),
    }))
}

const noticeResults = async (role, term) => {
  const endpoint = NOTICE_ENDPOINTS[role]
  if (!endpoint) return []
  const data = await api.get(endpoint)
  return (data?.items || [])
    .filter((notice) => hit(notice.title, term) || hit(notice.description, term))
    .slice(0, 6)
    .map((notice) => ({
      id: `notice-${notice.id}`,
      title: notice.title,
      subtitle: notice.created_by_name || 'Notice',
      to: noticesFor(role, { highlight: notice.id }),
    }))
}

export const searchEverything = async (role, raw) => {
  const term = String(raw || '').trim().toLowerCase()
  if (term.length < MIN_TERM) return []
  const sources = [
    { label: 'Pages', run: async () => pageResults(role, term) },
    { label: 'Courses', run: () => courseResults(term) },
    { label: 'Teachers', run: () => teacherResults(term) },
    { label: 'People', run: () => peopleResults(term) },
    { label: 'Notices', run: () => noticeResults(role, term) },
  ]
  const settled = await Promise.allSettled(sources.map((source) => source.run()))
  return sources
    .map((source, index) => ({
      label: source.label,
      items: settled[index].status === 'fulfilled' ? settled[index].value : [],
    }))
    .filter((section) => section.items.length > 0)
}

export const recentSearches = () => {
  const stored = readJson(RECENT_KEY)
  return Array.isArray(stored) ? stored.filter((entry) => typeof entry === 'string') : []
}

export const rememberSearch = (term) => {
  const value = String(term || '').trim()
  if (value.length < MIN_TERM) return recentSearches()
  const next = [value, ...recentSearches().filter((entry) => entry !== value)].slice(0, MAX_RECENT)
  writeJson(RECENT_KEY, next)
  return next
}

export const forgetSearches = () => {
  writeJson(RECENT_KEY, [])
  return []
}

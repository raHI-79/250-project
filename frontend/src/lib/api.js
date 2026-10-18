import { readJson, remove } from './storage'
import { loginFor } from './paths'

export const SESSION_KEY = 'sustcse.session'

export class ApiError extends Error {
  constructor(status, message) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const PUBLIC_ENDPOINTS = [
  '/api/auth/config',
  '/api/auth/login',
  '/api/auth/admin/login',
  '/api/auth/register',
  '/api/auth/verify-email',
  '/api/auth/resend-verification',
  '/api/auth/drop-course-options',
  '/api/auth/google/verify-email',
  '/api/auth/invite',
  '/api/auth/activate',
  '/api/auth/forgot-password',
  '/api/auth/verify-reset-code',
  '/api/auth/reset-password',
]

const isPublicEndpoint = (path) => PUBLIC_ENDPOINTS.some((entry) => path.startsWith(entry))

const currentToken = () => readJson(SESSION_KEY)?.token || null

const currentRole = () => readJson(SESSION_KEY)?.user?.role || null

const expireSession = () => {
  const target = loginFor(currentRole())
  remove(SESSION_KEY)
  if (window.location.pathname !== target) {
    window.location.assign(target)
  }
}

const withQuery = (path, query) => {
  if (!query) return path
  const search = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    search.set(key, String(value))
  })
  const text = search.toString()
  return text ? `${path}?${text}` : path
}

const parseBody = async (response) => {
  if (response.status === 204) return null
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

const request = async (method, path, { body, query, signal, auth = true } = {}) => {
  const headers = {}
  const isForm = typeof FormData !== 'undefined' && body instanceof FormData
  let payload
  if (body !== undefined && body !== null) {
    if (isForm) {
      payload = body
    } else {
      headers['Content-Type'] = 'application/json'
      payload = JSON.stringify(body)
    }
  }
  if (auth) {
    const token = currentToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }
  const url = withQuery(path, query)
  let response
  try {
    response = await fetch(url, { method, headers, body: payload, signal })
  } catch (error) {
    if (error?.name === 'AbortError') throw error
    throw new ApiError(0, 'Cannot reach the server. Please check your connection.')
  }
  const data = await parseBody(response)
  if (response.ok) return data
  if (response.status === 401 && !isPublicEndpoint(path)) {
    expireSession()
  }
  const message =
    (data && typeof data === 'object' && data.error) ||
    (typeof data === 'string' && data) ||
    'Something went wrong. Please try again.'
  throw new ApiError(response.status, message)
}

export const api = {
  get: (path, options) => request('GET', path, options),
  post: (path, body, options) => request('POST', path, { ...options, body }),
  put: (path, body, options) => request('PUT', path, { ...options, body }),
  patch: (path, body, options) => request('PATCH', path, { ...options, body }),
  delete: (path, options) => request('DELETE', path, options),
}

export default api

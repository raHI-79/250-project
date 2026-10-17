export const readJson = (key) => {
  try {
    const raw = window.localStorage.getItem(key)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export const writeJson = (key, value) => {
  try {
    window.localStorage.setItem(key, JSON.stringify(value))
    return true
  } catch {
    return false
  }
}

export const readRaw = (key) => {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

export const writeRaw = (key, value) => {
  try {
    window.localStorage.setItem(key, value)
    return true
  } catch {
    return false
  }
}

export const remove = (key) => {
  try {
    window.localStorage.removeItem(key)
    return true
  } catch {
    return false
  }
}

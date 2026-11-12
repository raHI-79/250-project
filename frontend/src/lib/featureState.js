import { useSyncExternalStore } from 'react'
import { readJson, writeJson } from './storage'

const STORE_KEY = 'sustcse.features'

let features = null

let loaded = false

const listeners = new Set()

const same = (left, right) => JSON.stringify(left || {}) === JSON.stringify(right || {})

const announce = () => listeners.forEach((listener) => listener())

const stored = () => {
  const value = readJson(STORE_KEY)
  return value && typeof value === 'object' ? value : null
}

export const currentFeatures = () => {
  if (features === null) features = stored() || {}
  return features
}

export const featuresLoaded = () => loaded

export const setFeatures = (next) => {
  const value = next && typeof next === 'object' ? next : {}
  const changed = !same(currentFeatures(), value) || !loaded
  loaded = true
  if (!changed) return
  features = value
  writeJson(STORE_KEY, value)
  announce()
}

export const turnFeatureOff = (feature) => {
  if (!feature || currentFeatures()[feature] === false) return
  features = { ...currentFeatures(), [feature]: false }
  writeJson(STORE_KEY, features)
  announce()
}

const subscribe = (listener) => {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export const useFeatures = () => useSyncExternalStore(subscribe, currentFeatures, currentFeatures)

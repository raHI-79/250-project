import { useCallback, useEffect, useRef, useState } from 'react'
import {
  PLACE_ASKING,
  PLACE_DENIED,
  PLACE_GRANTED,
  PLACE_IDLE,
  geolocationPermission,
  readBrowserPosition,
} from './webLocation.js'

export default function useWebLocation({ active, every = 0, onPosition }) {
  const [place, setPlace] = useState({ state: PLACE_IDLE })
  const sender = useRef(onPosition)
  useEffect(() => {
    sender.current = onPosition
  }, [onPosition])
  const locate = useCallback(async () => {
    setPlace((current) =>
      current.state === PLACE_GRANTED ? current : { ...current, state: PLACE_ASKING },
    )
    const result = await readBrowserPosition()
    if (!result.ok) {
      setPlace({ state: result.state })
      return null
    }
    setPlace({ state: PLACE_GRANTED, accuracy: result.position.accuracy_m })
    if (sender.current) {
      try {
        await sender.current(result.position)
      } catch {
        return result.position
      }
    }
    return result.position
  }, [])
  useEffect(() => {
    if (!active) {
      setPlace({ state: PLACE_IDLE })
      return undefined
    }
    let gone = false
    geolocationPermission().then((state) => {
      if (gone) return
      if (state === 'granted') locate()
      else if (state === 'denied') setPlace({ state: PLACE_DENIED })
    })
    return () => {
      gone = true
    }
  }, [active, locate])
  useEffect(() => {
    if (!active || !every || place.state !== PLACE_GRANTED) return undefined
    const timer = window.setInterval(() => {
      locate()
    }, every)
    return () => window.clearInterval(timer)
  }, [active, every, place.state, locate])
  return { place, locate }
}

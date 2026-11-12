import { useCallback, useEffect, useRef, useState } from 'react'
import {
  PLACE_ASKING,
  PLACE_DENIED,
  PLACE_GRANTED,
  PLACE_IDLE,
  PLACE_WEAK,
  geolocationPermission,
  readBestPosition,
} from './webLocation.js'

export default function useWebLocation({ active, every = 0, limit = 0, onPosition }) {
  const [place, setPlace] = useState({ state: PLACE_IDLE })
  const sender = useRef(onPosition)
  const running = useRef(null)
  useEffect(() => {
    sender.current = onPosition
  }, [onPosition])
  const read = useCallback(async () => {
    setPlace((current) =>
      current.state === PLACE_GRANTED ? current : { state: PLACE_ASKING, accuracy: 0 },
    )
    const result = await readBestPosition({
      onReading: (best) =>
        setPlace((current) =>
          current.state === PLACE_GRANTED
            ? current
            : { state: PLACE_ASKING, accuracy: best.accuracy_m },
        ),
    })
    if (!result.ok) {
      setPlace({ state: result.state })
      return null
    }
    const accuracy = result.position.accuracy_m
    if (limit > 0 && accuracy > limit) {
      setPlace({ state: PLACE_WEAK, accuracy })
      return null
    }
    setPlace({ state: PLACE_GRANTED, accuracy })
    if (sender.current) {
      try {
        await sender.current(result.position)
      } catch {
        return result.position
      }
    }
    return result.position
  }, [limit])
  const locate = useCallback(() => {
    if (!running.current) {
      running.current = read().finally(() => {
        running.current = null
      })
    }
    return running.current
  }, [read])
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

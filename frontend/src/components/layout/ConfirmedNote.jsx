import { useEffect, useState } from 'react'
import { ShieldCheck } from 'lucide-react'
import { confirmedUntil, watchConfirmed } from '../../lib/confirmgate'

const TICK = 15000

const minutesLeft = (until) => Math.max(1, Math.ceil((until - Date.now()) / 60000))

export default function ConfirmedNote() {
  const [until, setUntil] = useState(confirmedUntil)
  const [, setTick] = useState(0)
  useEffect(() => watchConfirmed(setUntil), [])
  useEffect(() => {
    if (!until) return undefined
    const timer = window.setInterval(() => setTick((current) => current + 1), TICK)
    return () => window.clearInterval(timer)
  }, [until])
  if (!until || until <= Date.now()) return null
  return (
    <span className="hidden items-center gap-1.5 rounded-full bg-white/15 px-2.5 py-1 text-2xs font-semibold text-white sm:inline-flex">
      <ShieldCheck size={13} aria-hidden="true" />
      Confirmed for {minutesLeft(until)} minutes
    </span>
  )
}

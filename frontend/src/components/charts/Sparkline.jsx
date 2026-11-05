const WIDTH = 96

const HEIGHT = 28

export default function Sparkline({ points = [], color = 'var(--color-brand-600)', label }) {
  const values = points.filter((value) => Number.isFinite(value))
  if (values.length < 2) return null
  const top = Math.max(...values)
  const floor = Math.min(...values)
  const span = top - floor || 1
  const step = WIDTH / (values.length - 1)
  const path = values
    .map((value, index) => {
      const x = index * step
      const y = HEIGHT - ((value - floor) / span) * (HEIGHT - 4) - 2
      return `${index === 0 ? 'M' : 'L'}${x.toFixed(1)} ${y.toFixed(1)}`
    })
    .join(' ')
  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      width={WIDTH}
      height={HEIGHT}
      role="img"
      aria-label={label || 'Trend'}
      className="shrink-0 overflow-visible"
    >
      <path d={path} fill="none" stroke={color} strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}

export const SERIES = [
  'var(--color-brand-600)',
  'var(--color-gold-500)',
  'var(--color-purple-600)',
  'var(--color-success-500)',
]

export const SERIES_NAMES = ['Ocean', 'Amber', 'Violet', 'Green']

export const GRID = 'var(--color-line)'

export const AXIS_INK = 'var(--color-muted)'

export const SURFACE = 'var(--color-surface)'

export const axis = {
  stroke: GRID,
  tick: { fill: AXIS_INK, fontSize: 12 },
  tickLine: false,
  axisLine: { stroke: GRID },
}

export const tooltipStyle = {
  contentStyle: {
    background: 'var(--color-surface)',
    border: '1px solid var(--color-line)',
    borderRadius: '0.75rem',
    boxShadow: 'var(--shadow-lift)',
    fontSize: '0.8125rem',
    color: 'var(--color-ink)',
  },
  labelStyle: { color: 'var(--color-ink-soft)', fontWeight: 700 },
  itemStyle: { color: 'var(--color-ink)' },
  cursor: { stroke: GRID, strokeWidth: 1 },
}

export const legendStyle = { fontSize: '0.8125rem', color: 'var(--color-ink-soft)' }

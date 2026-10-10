import { formatNumber } from './format'

export interface Point {
  x: number
  y: number
}

/**
 * Places values on a width by height canvas, left to right and evenly
 * spaced, with pad kept clear on every side so markers are not clipped.
 * A flat series sits in the middle rather than on an edge.
 */
export function sparkPoints(values: number[], width: number, height: number, pad = 4): Point[] {
  if (values.length === 0) return []
  const lo = Math.min(...values)
  const hi = Math.max(...values)
  const span = hi - lo
  const step = values.length > 1 ? (width - pad * 2) / (values.length - 1) : 0
  return values.map((v, i) => ({
    x: values.length > 1 ? pad + i * step : width / 2,
    y: span === 0 ? height / 2 : pad + (1 - (v - lo) / span) * (height - pad * 2),
  }))
}

/** The index of the evenly spaced point nearest to x. */
export function nearestIndex(x: number, count: number, width: number, pad = 4): number {
  if (count <= 1) return 0
  const step = (width - pad * 2) / (count - 1)
  return Math.min(count - 1, Math.max(0, Math.round((x - pad) / step)))
}

/** "+1,204 since the last scan", "−3 since the last scan" or "No change since the last scan". */
export function describeChange(values: number[]): string {
  if (values.length < 2) return 'First scan'
  const change = values[values.length - 1] - values[values.length - 2]
  if (change === 0) return 'No change since the last scan'
  const sign = change > 0 ? '+' : '−'
  return `${sign}${formatNumber(Math.abs(change))} since the last scan`
}

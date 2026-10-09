import type { DayActivity } from './types'

export type Level = 0 | 1 | 2 | 3 | 4

export interface CalendarCell {
  date: string // YYYY-MM-DD
  count: number
  level: Level
}

export interface Calendar {
  /** Columns of seven days, Sunday first. Days after the end are null. */
  weeks: (CalendarCell | null)[][]
  /** Month labels and the week column each starts in. */
  months: { label: string; week: number }[]
  max: number
  total: number
  start: string
  end: string
}

const dayMs = 86_400_000
const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

const parse = (iso: string) => Date.UTC(+iso.slice(0, 4), +iso.slice(5, 7) - 1, +iso.slice(8, 10))
const format = (ms: number) => new Date(ms).toISOString().slice(0, 10)

/** Buckets a count against the busiest day into five intensity levels. */
export function levelFor(count: number, max: number): Level {
  if (count <= 0 || max <= 0) return 0
  return Math.min(4, Math.max(1, Math.ceil((count / max) * 4))) as Level
}

/**
 * Lays sparse daily counts out as a week-column calendar ending on `end`,
 * covering `weeks` columns. The first column starts on a Sunday.
 */
export function buildCalendar(daily: DayActivity[], end: string, weeks = 53): Calendar {
  const counts = new Map(daily.map((d) => [d.date, d.count]))
  const endMs = parse(end)
  const endDow = new Date(endMs).getUTCDay()
  const startMs = endMs - endDow * dayMs - (weeks - 1) * 7 * dayMs

  let max = 0
  let total = 0
  for (let t = startMs; t <= endMs; t += dayMs) {
    const c = counts.get(format(t)) ?? 0
    max = Math.max(max, c)
    total += c
  }

  const columns: (CalendarCell | null)[][] = []
  const months: Calendar['months'] = []
  let lastMonth = -1
  for (let w = 0; w < weeks; w++) {
    const column: (CalendarCell | null)[] = []
    for (let d = 0; d < 7; d++) {
      const t = startMs + (w * 7 + d) * dayMs
      if (t > endMs) {
        column.push(null)
        continue
      }
      const date = format(t)
      const count = counts.get(date) ?? 0
      column.push({ date, count, level: levelFor(count, max) })
    }
    const month = new Date(startMs + w * 7 * dayMs).getUTCMonth()
    if (month !== lastMonth) {
      months.push({ label: monthNames[month], week: w })
      lastMonth = month
    }
    columns.push(column)
  }
  // A label squeezed into the first column usually collides with the next one.
  if (months.length > 1 && months[1].week - months[0].week < 2) months.shift()

  return { weeks: columns, months, max, total, start: format(startMs), end }
}

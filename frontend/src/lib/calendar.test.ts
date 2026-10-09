import { describe, expect, it } from 'vitest'
import { buildCalendar, levelFor } from './calendar'

describe('levelFor', () => {
  it('maps counts to five levels relative to the max', () => {
    expect(levelFor(0, 10)).toBe(0)
    expect(levelFor(1, 10)).toBe(1)
    expect(levelFor(5, 10)).toBe(2)
    expect(levelFor(7, 10)).toBe(3)
    expect(levelFor(10, 10)).toBe(4)
    expect(levelFor(3, 0)).toBe(0)
  })
})

describe('buildCalendar', () => {
  // 2026-10-09 is a Friday.
  const cal = buildCalendar(
    [
      { date: '2026-10-09', count: 4 },
      { date: '2026-10-04', count: 1 },
      { date: '2025-01-01', count: 99 }, // outside the window
    ],
    '2026-10-09',
    4,
  )

  it('starts on a Sunday and fills whole weeks up to the end date', () => {
    expect(cal.weeks).toHaveLength(4)
    expect(cal.start).toBe('2026-09-13')
    expect(new Date(cal.start + 'T00:00:00Z').getUTCDay()).toBe(0)
    const last = cal.weeks[3]
    expect(last[5]?.date).toBe('2026-10-09')
    expect(last[6]).toBeNull()
  })

  it('fills gaps with zero days and ignores days outside the window', () => {
    const cells = cal.weeks.flat().filter((c) => c !== null)
    expect(cells).toHaveLength(27)
    expect(cal.total).toBe(5)
    expect(cal.max).toBe(4)
    expect(cal.weeks[3][0]).toEqual({ date: '2026-10-04', count: 1, level: 1 })
    expect(cal.weeks[3][5]?.level).toBe(4)
  })

  it('labels months at the week they begin', () => {
    expect(cal.months).toEqual([
      { label: 'Sep', week: 0 },
      { label: 'Oct', week: 3 },
    ])
  })
})

import { describe, expect, it } from 'vitest'
import { describeChange, nearestIndex, sparkPoints } from './trend'

describe('sparkPoints', () => {
  it('spans the canvas inside the padding', () => {
    const pts = sparkPoints([0, 5, 10], 108, 48, 4)
    expect(pts).toEqual([
      { x: 4, y: 44 },
      { x: 54, y: 24 },
      { x: 104, y: 4 },
    ])
  })

  it('centers flat and single-value series', () => {
    expect(sparkPoints([7, 7], 100, 40).map((p) => p.y)).toEqual([20, 20])
    expect(sparkPoints([3], 100, 40)).toEqual([{ x: 50, y: 20 }])
    expect(sparkPoints([], 100, 40)).toEqual([])
  })
})

describe('nearestIndex', () => {
  it('snaps to the closest point and clamps to the ends', () => {
    expect(nearestIndex(50, 3, 108, 4)).toBe(1)
    expect(nearestIndex(-20, 3, 108, 4)).toBe(0)
    expect(nearestIndex(500, 3, 108, 4)).toBe(2)
    expect(nearestIndex(30, 1, 108, 4)).toBe(0)
  })
})

describe('describeChange', () => {
  it('compares the last two scans', () => {
    expect(describeChange([5])).toBe('First scan')
    expect(describeChange([5, 5])).toBe('No change since the last scan')
    expect(describeChange([100, 1304])).toBe('+1,204 since the last scan')
    expect(describeChange([10, 7])).toBe('−3 since the last scan')
  })
})

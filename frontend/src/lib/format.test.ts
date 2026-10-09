import { describe, expect, it } from 'vitest'
import {
  formatBytes,
  formatCompact,
  formatNumber,
  formatPercent,
  formatRelative,
  plural,
  shortHash,
} from './format'

describe('formatBytes', () => {
  it('keeps bytes whole and scales by 1024', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(630)).toBe('630 B')
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1024 * 1024)).toBe('1.0 MB')
    expect(formatBytes(1572036)).toBe('1.5 MB')
  })
})

describe('numbers', () => {
  it('groups thousands', () => {
    expect(formatNumber(14916)).toBe('14,916')
  })
  it('compacts only large values', () => {
    expect(formatCompact(9999)).toBe('9,999')
    expect(formatCompact(14916)).toBe('14.9K')
    expect(formatCompact(2_400_000)).toBe('2.4M')
  })
  it('formats shares', () => {
    expect(formatPercent(0.4567)).toBe('45.7%')
    expect(formatPercent(0.0004)).toBe('<0.1%')
    expect(formatPercent(0)).toBe('0.0%')
  })
  it('pluralizes', () => {
    expect(plural(1, 'file')).toBe('1 file')
    expect(plural(1200, 'file')).toBe('1,200 files')
    expect(plural(2, 'dependency', 'dependencies')).toBe('2 dependencies')
  })
})

describe('formatRelative', () => {
  const now = new Date('2026-10-09T12:00:00Z')
  it('describes recent moments', () => {
    expect(formatRelative('2026-10-09T11:59:40Z', now)).toBe('just now')
    expect(formatRelative('2026-10-09T11:55:00Z', now)).toBe('5 min ago')
    expect(formatRelative('2026-10-09T11:00:00Z', now)).toBe('1 hour ago')
    expect(formatRelative('2026-10-08T10:00:00Z', now)).toBe('yesterday')
    expect(formatRelative('2026-09-29T12:00:00Z', now)).toBe('10 days ago')
  })
  it('falls back to a date after a month', () => {
    expect(formatRelative('2026-08-07T12:00:00Z', now)).toBe('Aug 7, 2026')
  })
})

it('shortens hashes to seven characters', () => {
  expect(shortHash('41a95d483d833ce6bc8662e443870695dc1b7f76')).toBe('41a95d4')
})

import { describe, expect, it } from 'vitest'
import { highlight, matcherFor } from './highlight'

const opts = { regex: false, caseSensitive: false, wholeWord: false }

describe('matcherFor', () => {
  it('escapes plain text and honors case', () => {
    const m = matcherFor({ ...opts, query: 'a.b' })!
    expect('A.B'.match(m)).toBeTruthy()
    expect('axb'.match(m)).toBeNull()
    expect('A.B'.match(matcherFor({ ...opts, query: 'a.b', caseSensitive: true })!)).toBeNull()
  })
  it('groups whole-word alternations', () => {
    const m = matcherFor({ ...opts, query: 'foo|bar', regex: true, wholeWord: true })!
    expect('foobar'.match(m)).toBeNull()
    expect('a bar'.match(m)?.[0]).toBe('bar')
  })
  it('returns null for empty or invalid patterns', () => {
    expect(matcherFor({ ...opts, query: '' })).toBeNull()
    expect(matcherFor({ ...opts, query: '(', regex: true })).toBeNull()
  })
})

describe('highlight', () => {
  it('splits text around every match', () => {
    expect(highlight('let x = useQuery(useQuery)', matcherFor({ ...opts, query: 'usequery' }))).toEqual([
      { text: 'let x = ', match: false },
      { text: 'useQuery', match: true },
      { text: '(', match: false },
      { text: 'useQuery', match: true },
      { text: ')', match: false },
    ])
  })
  it('ignores zero-length matches and passes text through without a matcher', () => {
    expect(highlight('abc', matcherFor({ ...opts, query: 'x*', regex: true }))).toEqual([{ text: 'abc', match: false }])
    expect(highlight('abc', null)).toEqual([{ text: 'abc', match: false }])
  })
})

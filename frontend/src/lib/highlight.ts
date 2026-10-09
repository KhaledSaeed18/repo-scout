export interface Segment {
  text: string
  match: boolean
}

export interface MatchOptions {
  query: string
  regex: boolean
  caseSensitive: boolean
  wholeWord: boolean
}

const escape = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/**
 * Builds a browser regex mirroring the server's matcher, for highlighting.
 * Returns null when the pattern is empty or not valid JavaScript.
 */
export function matcherFor({ query, regex, caseSensitive, wholeWord }: MatchOptions): RegExp | null {
  if (!query) return null
  let source = regex ? query : escape(query)
  if (wholeWord) source = `\\b(?:${source})\\b`
  try {
    return new RegExp(source, caseSensitive ? 'g' : 'gi')
  } catch {
    return null
  }
}

/** Splits text into alternating plain and matched segments. */
export function highlight(text: string, matcher: RegExp | null): Segment[] {
  if (!matcher) return [{ text, match: false }]
  const out: Segment[] = []
  let last = 0
  for (const m of text.matchAll(new RegExp(matcher.source, matcher.flags))) {
    const start = m.index
    if (m[0] === '') continue
    if (start > last) out.push({ text: text.slice(last, start), match: false })
    out.push({ text: m[0], match: true })
    last = start + m[0].length
  }
  if (last < text.length) out.push({ text: text.slice(last), match: false })
  return out.length ? out : [{ text, match: false }]
}

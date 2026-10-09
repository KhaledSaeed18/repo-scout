import { describe, expect, it } from 'vitest'
import { collapse, folderAt, layer, positions } from './graph'

describe('folderAt', () => {
  it('cuts file folders and folder paths to a depth', () => {
    expect(folderAt('src/app/main.ts', 1, false)).toBe('src')
    expect(folderAt('src/app/main.ts', 5, false)).toBe('src/app')
    expect(folderAt('internal/api', 1, true)).toBe('internal')
    expect(folderAt('main.go', 2, false)).toBe('(root)')
  })
})

describe('collapse', () => {
  const edges = [
    { from: 'src/a/x.ts', to: 'src/b/y.ts' },
    { from: 'src/a/z.ts', to: 'src/b/y.ts' },
    { from: 'src/a/x.ts', to: 'src/a/z.ts' },
    { from: 'cmd/main.go', to: 'internal/api' },
  ]
  it('keeps files as nodes without a depth', () => {
    const g = collapse(edges, new Set(['internal/api']))
    expect(g.nodes).toContain('src/a/x.ts')
    expect(g.edges).toHaveLength(4)
  })
  it('merges parallel edges and drops self-loops at a folder depth', () => {
    const g = collapse(edges, new Set(['internal/api']), 2)
    expect(g.nodes).toEqual(['cmd', 'internal/api', 'src/a', 'src/b'])
    expect(g.edges).toContainEqual({ from: 'src/a', to: 'src/b', weight: 2 })
    expect(g.edges.find((e) => e.from === e.to)).toBeUndefined()
  })
})

describe('layer', () => {
  it('places importers left of what they import, using the longest path', () => {
    const cols = layer({
      nodes: ['a', 'b', 'c', 'd'],
      edges: [
        { from: 'a', to: 'b', weight: 1 },
        { from: 'b', to: 'c', weight: 1 },
        { from: 'a', to: 'c', weight: 1 },
        { from: 'd', to: 'c', weight: 1 },
      ],
    })
    expect(Object.fromEntries(cols)).toEqual({ a: 0, b: 1, c: 2, d: 0 })
  })
  it('terminates on cycles and still layers every node', () => {
    const cols = layer({
      nodes: ['a', 'b', 'c'],
      edges: [
        { from: 'a', to: 'b', weight: 1 },
        { from: 'b', to: 'c', weight: 1 },
        { from: 'c', to: 'a', weight: 1 },
      ],
    })
    expect(cols.size).toBe(3)
    expect(new Set(cols.values()).size).toBe(3)
  })
  it('handles a long chain without blowing the stack', () => {
    const n = 20000
    const nodes = Array.from({ length: n }, (_, i) => `n${i}`)
    const edges = nodes.slice(1).map((to, i) => ({ from: nodes[i], to, weight: 1 }))
    expect(layer({ nodes, edges }).get(`n${n - 1}`)).toBe(n - 1)
  })
})

it('positions nodes on a column grid', () => {
  const pos = positions({ nodes: ['a', 'b', 'c'], edges: [{ from: 'a', to: 'b', weight: 1 }, { from: 'a', to: 'c', weight: 1 }] }, 200, 40)
  expect(pos).toEqual([
    { id: 'a', x: 0, y: 0 },
    { id: 'b', x: 200, y: 0 },
    { id: 'c', x: 200, y: 40 },
  ])
})

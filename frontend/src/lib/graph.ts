export interface GraphEdge {
  from: string
  to: string
  weight: number
}

export interface Graph {
  nodes: string[]
  edges: GraphEdge[]
}

/** Folder prefix of a path, at most `depth` segments; files map to their folder. */
export function folderAt(path: string, depth: number, isFolder: boolean): string {
  const parts = path.split('/')
  const dirParts = isFolder ? parts : parts.slice(0, -1)
  return dirParts.length ? dirParts.slice(0, depth).join('/') : '(root)'
}

/**
 * Collapses file-level import edges into a weighted graph. With a depth,
 * endpoints become folders cut to that depth; without one, files stay files.
 * Self-loops are dropped and parallel edges merged into a weight.
 */
export function collapse(
  edges: { from: string; to: string }[],
  folders: Set<string>,
  depth?: number,
): Graph {
  const key = (p: string) => (depth ? folderAt(p, depth, folders.has(p)) : p)
  const weights = new Map<string, GraphEdge>()
  const nodes = new Set<string>()
  for (const e of edges) {
    const from = key(e.from)
    const to = key(e.to)
    nodes.add(from)
    nodes.add(to)
    if (from === to) continue
    const id = `${from}\u0000${to}`
    const existing = weights.get(id)
    if (existing) existing.weight++
    else weights.set(id, { from, to, weight: 1 })
  }
  return { nodes: [...nodes].sort(), edges: [...weights.values()] }
}

/**
 * Assigns each node a column so imports flow left to right: a node sits one
 * column right of the furthest node importing it. Back edges found by DFS are
 * ignored, so cycles cannot cause unbounded work. Runs in O(V + E).
 */
export function layer(graph: Graph): Map<string, number> {
  const out = new Map<string, string[]>()
  for (const n of graph.nodes) out.set(n, [])
  for (const e of graph.edges) out.get(e.from)?.push(e.to)

  // Iterative DFS marks back edges (edges into a node still on the stack).
  const state = new Map<string, 0 | 1 | 2>()
  const back = new Set<string>()
  for (const root of graph.nodes) {
    if (state.get(root)) continue
    const stack: [string, number][] = [[root, 0]]
    state.set(root, 1)
    while (stack.length) {
      const top = stack[stack.length - 1]
      const next = out.get(top[0])![top[1]++]
      if (next === undefined) {
        state.set(top[0], 2)
        stack.pop()
      } else if (state.get(next) === 1) {
        back.add(`${top[0]}\u0000${next}`)
      } else if (!state.get(next)) {
        state.set(next, 1)
        stack.push([next, 0])
      }
    }
  }

  // Longest path over the remaining DAG, in topological (Kahn) order.
  const indegree = new Map(graph.nodes.map((n) => [n, 0]))
  const forward = graph.edges.filter((e) => !back.has(`${e.from}\u0000${e.to}`))
  for (const e of forward) indegree.set(e.to, indegree.get(e.to)! + 1)
  const adj = new Map<string, string[]>(graph.nodes.map((n) => [n, []]))
  for (const e of forward) adj.get(e.from)!.push(e.to)
  const column = new Map(graph.nodes.map((n) => [n, 0]))
  const queue = graph.nodes.filter((n) => indegree.get(n) === 0)
  for (let i = 0; i < queue.length; i++) {
    const n = queue[i]
    for (const m of adj.get(n)!) {
      column.set(m, Math.max(column.get(m)!, column.get(n)! + 1))
      indegree.set(m, indegree.get(m)! - 1)
      if (indegree.get(m) === 0) queue.push(m)
    }
  }
  return column
}

export interface Positioned {
  id: string
  x: number
  y: number
}

/**
 * Lays nodes out in columns, ordering each column by the average row of the
 * nodes importing it to cut down on crossing edges.
 */
export function positions(graph: Graph, colWidth: number, rowHeight: number): Positioned[] {
  const column = layer(graph)
  const incoming = new Map<string, string[]>(graph.nodes.map((n) => [n, []]))
  for (const e of graph.edges) incoming.get(e.to)?.push(e.from)

  const byColumn = new Map<number, string[]>()
  for (const n of graph.nodes) {
    const c = column.get(n)!
    if (!byColumn.has(c)) byColumn.set(c, [])
    byColumn.get(c)!.push(n)
  }
  const row = new Map<string, number>()
  const out: Positioned[] = []
  for (const c of [...byColumn.keys()].sort((a, b) => a - b)) {
    const ids = byColumn.get(c)!
    const score = (n: string) => {
      const rows = incoming.get(n)!.map((p) => row.get(p)).filter((r): r is number => r !== undefined)
      return rows.length ? rows.reduce((a, b) => a + b, 0) / rows.length : Number.MAX_SAFE_INTEGER
    }
    ids.sort((a, b) => score(a) - score(b) || a.localeCompare(b))
    ids.forEach((id, i) => {
      row.set(id, i)
      out.push({ id, x: c * colWidth, y: i * rowHeight })
    })
  }
  return out
}

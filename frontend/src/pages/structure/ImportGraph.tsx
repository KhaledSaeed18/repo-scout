import { useMemo, useState } from 'react'
import ReactFlow, { Background, Controls, MarkerType, Position, type Edge, type Node } from 'reactflow'
import 'reactflow/dist/style.css'
import { collapse, positions } from '@/lib/graph'
import { plural } from '@/lib/format'

const nodeWidth = 190
const colWidth = 250
const rowHeight = 52

export interface GraphView {
  edges: { from: string; to: string }[]
  folders: Set<string>
  depth?: number
}

/** Interactive import graph. Clicking a node highlights what it touches. */
export default function ImportGraph({ edges, folders, depth }: GraphView) {
  const [selected, setSelected] = useState<string | null>(null)
  const graph = useMemo(() => collapse(edges, folders, depth), [edges, folders, depth])
  const layout = useMemo(() => positions(graph, colWidth, rowHeight), [graph])

  const neighbors = useMemo(() => {
    if (!selected) return null
    const set = new Set([selected])
    for (const e of graph.edges) {
      if (e.from === selected) set.add(e.to)
      if (e.to === selected) set.add(e.from)
    }
    return set
  }, [graph, selected])

  const maxWeight = Math.max(1, ...graph.edges.map((e) => e.weight))
  // Size the canvas to the layout so fitting it does not shrink small graphs.
  const tallest = Math.max(1, ...layout.map((p) => p.y / rowHeight + 1))
  const height = Math.min(640, Math.max(300, tallest * rowHeight + 120))
  const nodes: Node[] = layout.map((p) => {
    const dim = neighbors && !neighbors.has(p.id)
    const label = depth ? p.id : p.id.split('/').pop()
    return {
      id: p.id,
      position: { x: p.x, y: p.y },
      data: { label },
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      style: {
        width: nodeWidth,
        padding: '6px 10px',
        fontSize: 12,
        textAlign: 'left',
        borderRadius: 4,
        background: p.id === selected ? 'var(--accent)' : 'var(--card)',
        color: p.id === selected ? 'var(--accent-foreground)' : 'var(--card-foreground)',
        border: `1px solid ${p.id === selected ? 'var(--primary)' : 'var(--border)'}`,
        opacity: dim ? 0.25 : 1,
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
      },
    }
  })
  const flowEdges: Edge[] = graph.edges.map((e) => {
    const active = neighbors ? e.from === selected || e.to === selected : false
    const color = active ? 'var(--primary)' : 'var(--muted-foreground)'
    return {
      id: `${e.from}->${e.to}`,
      source: e.from,
      target: e.to,
      label: depth && e.weight > 1 ? String(e.weight) : undefined,
      labelStyle: { fontSize: 10, fill: 'var(--muted-foreground)' },
      labelBgStyle: { fill: 'var(--background)' },
      markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color },
      style: {
        stroke: color,
        strokeWidth: 1 + (depth ? (e.weight / maxWeight) * 2.5 : 0),
        opacity: neighbors && !active ? 0.12 : active ? 1 : 0.45,
      },
    }
  })

  return (
    <div className="flex flex-col gap-2">
      <div className="overflow-hidden rounded-md border bg-background" style={{ height }}>
        <ReactFlow
          nodes={nodes}
          edges={flowEdges}
          fitView
          fitViewOptions={{ padding: 0.08 }}
          minZoom={0.1}
          nodesConnectable={false}
          onNodeClick={(_, n) => setSelected((cur) => (cur === n.id ? null : n.id))}
          onPaneClick={() => setSelected(null)}
        >
          <Background gap={20} size={1} color="var(--border)" />
          <Controls showInteractive={false} />
        </ReactFlow>
      </div>
      <p className="text-xs text-muted-foreground">
        {plural(graph.nodes.length, depth ? 'folder' : 'file')} and {plural(graph.edges.length, 'import')}. Arrows point
        from the importer to what it imports. Click a node to trace its connections.
        {depth ? ' Numbers count the file-level imports between two folders.' : ''}
      </p>
    </div>
  )
}

import { useMemo } from 'react'
import ReactFlow, {
  Background,
  Controls,
  type Edge,
  type Node,
} from 'reactflow'
import 'reactflow/dist/style.css'
import { Button, Card, Empty, Spinner, Table } from '../components/ui'
import RepoSelector from '../components/RepoSelector'
import { api, useArchitecture, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

const nodeWidth = 170
const nodeHeight = 36
const colSpace = 220
const rowSpace = 48

function layout(edges: { from: string; to: string }[]): {
  nodes: Node[]
  edges: Edge[]
} {
  const adj = new Map<string, string[]>()
  const seen = new Set<string>()
  for (const e of edges) {
    seen.add(e.from)
    seen.add(e.to)
    if (!adj.has(e.from)) adj.set(e.from, [])
    adj.get(e.from)!.push(e.to)
  }
  const depthOf = (node: string): number => {
    let depth = 0
    const visit = (n: string, d: number) => {
      if (d > depth) depth = d
      for (const next of adj.get(n) ?? []) visit(next, d + 1)
    }
    visit(node, 0)
    return depth
  }
  const byDepth = new Map<number, string[]>()
  for (const n of seen) {
    const d = depthOf(n)
    if (!byDepth.has(d)) byDepth.set(d, [])
    byDepth.get(d)!.push(n)
  }
  const positions = new Map<string, { x: number; y: number }>()
  for (const [depth, names] of byDepth) {
    names.forEach((n, i) => positions.set(n, { x: 40 + depth * colSpace, y: 40 + i * rowSpace }))
  }
  const nodes: Node[] = [...seen].map((n) => ({
    id: n,
    position: positions.get(n)!,
    data: { label: n.split('/').pop() },
    style: {
      background: '#111a2e',
      border: '1px solid #334155',
      borderRadius: 6,
      color: '#e2e8f0',
      fontSize: 11,
      width: nodeWidth,
      height: nodeHeight,
    },
  }))
  const flowEdges: Edge[] = edges.map((e) => ({
    id: `${e.from}->${e.to}`,
    source: e.from,
    target: e.to,
    animated: false,
    style: { stroke: '#38bdf8', strokeOpacity: 0.5 },
  }))
  return { nodes, edges: flowEdges }
}

function ArchitecturePage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const { data, isLoading } = useArchitecture(repoId)
  const graph = useMemo(
    () => layout((data?.edges ?? []).map((e) => ({ from: e.from, to: e.to }))),
    [data?.edges],
  )
  if (repoId === 0) return <Empty message="Scan a repository first" />
  if (isLoading) return <Spinner />
  if (!data) return <Empty message="No architecture data" />

  const nodeCount = graph.nodes.length
  const edges = graph.edges.length

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Architecture</h1>
        <div className="flex items-center gap-2">
          <a href={api.svgUrl(repoId)} download>
            <Button variant="outline">Download SVG</Button>
          </a>
          <RepoSelector />
        </div>
      </div>

      {graph.nodes.length === 0 ? (
        <Empty message="No import edges to visualize" />
      ) : (
        <>
          <Card className="p-0">
            <div className="h-[520px]">
              <ReactFlow nodes={graph.nodes} edges={graph.edges} fitView nodesDraggable>
                <Background gap={16} color="#1e293b" />
                <Controls />
              </ReactFlow>
            </div>
          </Card>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Card>
              <h3 className="mb-3 text-sm font-medium text-slate-300">
                Graph summary
              </h3>
              <Table
                headers={['Measure', 'Value']}
                rows={[
                  ['Nodes', nodeCount],
                  ['Edges', edges],
                  ['Entry points', (data.entryPoints ?? []).length],
                  ['Dead files', (data.deadFiles ?? []).length],
                  ['Unused modules', (data.unusedModules ?? []).length],
                  ['Cycles', (data.cycles ?? []).length],
                ]}
              />
            </Card>
            <Card>
              <h3 className="mb-3 text-sm font-medium text-slate-300">Issues</h3>
              <Table
                headers={['Type', 'Items']}
                rows={[
                  [
                    'Cycles',
                    (data.cycles ?? []).length
                      ? data.cycles.map((c) => c.join(' → ')).join(' | ')
                      : 'none',
                  ],
                  [
                    'Dead files',
                    (data.deadFiles ?? []).length
                      ? data.deadFiles.slice(0, 15).join(', ')
                      : 'none',
                  ],
                  [
                    'Unused modules',
                    (data.unusedModules ?? []).length
                      ? data.unusedModules.slice(0, 15).join(', ')
                      : 'none',
                  ],
                ]}
              />
            </Card>
          </div>
        </>
      )}
    </div>
  )
}

export default function Architecture() {
  return (
    <RepoProvider>
      <ArchitecturePage />
    </RepoProvider>
  )
}

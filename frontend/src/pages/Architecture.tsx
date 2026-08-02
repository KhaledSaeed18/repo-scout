import { FolderGit2, Network, Workflow } from 'lucide-react'
import { useMemo } from 'react'
import ReactFlow, {
  Background,
  Controls,
  type Edge,
  type Node,
} from 'reactflow'
import 'reactflow/dist/style.css'
import { Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
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
      background: 'var(--card)',
      border: '1px solid var(--border)',
      borderRadius: 6,
      color: 'var(--card-foreground)',
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
    style: { stroke: 'var(--chart-1)', strokeOpacity: 0.5 },
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
  if (repoId === 0) return <EmptyState icon={FolderGit2} title="Scan a repository first" />
  if (isLoading) return <Spinner />
  if (!data) return <EmptyState icon={Network} title="No architecture data" />

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
        <EmptyState icon={Workflow} title="No import edges to visualize" />
      ) : (
        <>
          <Card className="p-0">
            <div className="h-[520px]">
              <ReactFlow nodes={graph.nodes} edges={graph.edges} fitView nodesDraggable>
                <Background gap={16} color="var(--border)" />
                <Controls />
              </ReactFlow>
            </div>
          </Card>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Graph summary</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Measure</TableHead>
                      <TableHead>Value</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow key="nodes">
                      <TableCell>Nodes</TableCell>
                      <TableCell>{nodeCount}</TableCell>
                    </TableRow>
                    <TableRow key="edges">
                      <TableCell>Edges</TableCell>
                      <TableCell>{edges}</TableCell>
                    </TableRow>
                    <TableRow key="entry-points">
                      <TableCell>Entry points</TableCell>
                      <TableCell>{(data.entryPoints ?? []).length}</TableCell>
                    </TableRow>
                    <TableRow key="dead-files">
                      <TableCell>Dead files</TableCell>
                      <TableCell>{(data.deadFiles ?? []).length}</TableCell>
                    </TableRow>
                    <TableRow key="unused-modules">
                      <TableCell>Unused modules</TableCell>
                      <TableCell>{(data.unusedModules ?? []).length}</TableCell>
                    </TableRow>
                    <TableRow key="cycles">
                      <TableCell>Cycles</TableCell>
                      <TableCell>{(data.cycles ?? []).length}</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Issues</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Type</TableHead>
                      <TableHead>Items</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow key="cycles-row">
                      <TableCell>Cycles</TableCell>
                      <TableCell>{(data.cycles ?? []).length ? data.cycles.map((c) => c.join(' → ')).join(' | ') : 'none'}</TableCell>
                    </TableRow>
                    <TableRow key="dead-files-row">
                      <TableCell>Dead files</TableCell>
                      <TableCell>{(data.deadFiles ?? []).length ? data.deadFiles.slice(0, 15).join(', ') : 'none'}</TableCell>
                    </TableRow>
                    <TableRow key="unused-modules-row">
                      <TableCell>Unused modules</TableCell>
                      <TableCell>{(data.unusedModules ?? []).length ? data.unusedModules.slice(0, 15).join(', ') : 'none'}</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </CardContent>
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

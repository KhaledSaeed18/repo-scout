import { FileBarChart2, FolderGit2 } from 'lucide-react'
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner, Stat, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import RepoSelector from '../components/RepoSelector'
import { useMetrics, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

const colors = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
]

function MetricsPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const { data, isLoading } = useMetrics(repoId)
  if (repoId === 0) return <EmptyState icon={FolderGit2} title="Scan a repository first" />
  if (isLoading) return <Spinner />
  if (!data) return <EmptyState icon={FileBarChart2} title="No metrics yet" />

  const langData = Object.entries(data.languages)
    .filter(([, v]) => v.files > 0)
    .sort((a, b) => b[1].code - a[1].code)
    .slice(0, 8)
    .map(([name, v]) => ({ name, code: v.code }))

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Metrics</h1>
        <RepoSelector />
      </div>
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <Stat label="Files" value={data.totals.files} />
        <Stat label="Code lines" value={data.totals.code.toLocaleString()} />
        <Stat label="Complexity" value={data.totals.complexity} />
        <Stat label="Functions" value={data.totals.funcs} />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Lines of code by language</CardTitle>
          </CardHeader>
          <CardContent>
            {langData.length ? (
              <ResponsiveContainer width="100%" height={240}>
                <BarChart data={langData}>
                  <XAxis dataKey="name" stroke="var(--muted-foreground)" fontSize={11} />
                  <YAxis stroke="var(--muted-foreground)" fontSize={12} />
                  <Tooltip
                    contentStyle={{ background: 'var(--popover)', border: '1px solid var(--border)' }}
                    labelStyle={{ color: 'var(--popover-foreground)' }}
                  />
                  <Bar dataKey="code" radius={[4, 4, 0, 0]} isAnimationActive={false}>
                    {langData.map((_, i) => (
                      <Cell key={i} fill={colors[i % colors.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <EmptyState icon={FileBarChart2} title="No source files" />
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Aggregates</CardTitle>
          </CardHeader>
          <CardContent className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Metric</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow key="comments">
                  <TableCell>Comments</TableCell>
                  <TableCell>{data.totals.comments}</TableCell>
                </TableRow>
                <TableRow key="blank-lines">
                  <TableCell>Blank lines</TableCell>
                  <TableCell>{data.totals.blank}</TableCell>
                </TableRow>
                <TableRow key="imports">
                  <TableCell>Imports</TableCell>
                  <TableCell>{data.totals.imports}</TableCell>
                </TableRow>
                <TableRow key="exports">
                  <TableCell>Exports</TableCell>
                  <TableCell>{data.totals.exports}</TableCell>
                </TableRow>
                <TableRow key="max-depth">
                  <TableCell>Deepest folder depth</TableCell>
                  <TableCell>{data.maxDepth}</TableCell>
                </TableRow>
                <TableRow key="deepest-file">
                  <TableCell>Deepest file</TableCell>
                  <TableCell><span className="break-all text-xs">{data.deepestFile || '—'}</span></TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Largest files</CardTitle>
        </CardHeader>
        <CardContent>
          {(data.largestFiles ?? []).length === 0 ? (
            <EmptyState icon={FileBarChart2} title="No files" />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>File</TableHead>
                    <TableHead>Code lines</TableHead>
                    <TableHead>Complexity</TableHead>
                    <TableHead>Functions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.largestFiles.map((f) => (
                    <TableRow key={f.path}>
                      <TableCell><span className="text-primary">{f.path}</span></TableCell>
                      <TableCell>{f.linesCode}</TableCell>
                      <TableCell>{f.complexity}</TableCell>
                      <TableCell>{f.funcCount}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Most complex files</CardTitle>
        </CardHeader>
        <CardContent>
          {(data.mostComplexFiles ?? []).length === 0 ? (
            <EmptyState icon={FileBarChart2} title="No files" />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>File</TableHead>
                    <TableHead>Complexity</TableHead>
                    <TableHead>Functions</TableHead>
                    <TableHead>Max nesting</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.mostComplexFiles.map((f) => (
                    <TableRow key={f.path}>
                      <TableCell><span className="text-primary">{f.path}</span></TableCell>
                      <TableCell>{f.complexity}</TableCell>
                      <TableCell>{f.funcCount}</TableCell>
                      <TableCell>{f.maxNesting}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function Metrics() {
  return (
    <RepoProvider>
      <MetricsPage />
    </RepoProvider>
  )
}

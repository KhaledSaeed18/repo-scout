import { Badge, Card, Empty, Spinner, Table } from '../components/ui'
import RepoSelector from '../components/RepoSelector'
import { useDuplicates, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

function DuplicatesPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const { data, isLoading } = useDuplicates(repoId)
  if (repoId === 0) return <Empty message="Scan a repository first" />
  if (isLoading) return <Spinner />
  const groups = data?.groups ?? []
  if (!groups.length) return <Empty message="No duplicate code blocks found" />

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Duplicates</h1>
        <RepoSelector />
      </div>
      {groups.map((g) => (
        <Card key={g.id} className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <Badge className="border-sky-800 bg-sky-500/20 text-sky-300">
              {g.lines} lines
            </Badge>
            <Badge className="border-sky-800 bg-sky-500/20 text-sky-300">
              {Math.round(g.similarity * 100)}% similar
            </Badge>
            <Badge>{g.fileCount} files</Badge>
          </div>
          <pre className="overflow-x-auto rounded border border-slate-800 bg-slate-950 p-3 text-xs text-slate-300">
            {g.fragment}
          </pre>
          <Table
            headers={['File', 'Start', 'End']}
            rows={g.blocks.map((b) => [
              <span key="p" className="text-sky-400">
                {b.filePath}
              </span>,
              b.startLine,
              b.endLine,
            ])}
          />
        </Card>
      ))}
    </div>
  )
}

export default function Duplicates() {
  return (
    <RepoProvider>
      <DuplicatesPage />
    </RepoProvider>
  )
}

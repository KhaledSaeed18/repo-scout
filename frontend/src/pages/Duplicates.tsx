import { Copy, FolderGit2 } from 'lucide-react'
import { Badge, Card, CardContent, EmptyState, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import RepoSelector from '../components/RepoSelector'
import { useDuplicates, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'

function DuplicatesPage() {
  const { repoId } = useRepoContext()
  const repo = useRepo(repoId).data
  const { data, isLoading } = useDuplicates(repoId)
  if (repoId === 0) return <EmptyState icon={FolderGit2} title="Scan a repository first" />
  if (isLoading) return <Spinner />
  const groups = data?.groups ?? []
  if (!groups.length) return <EmptyState icon={Copy} title="No duplicate code blocks found" />

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Duplicates</h1>
        <RepoSelector />
      </div>
      {groups.map((g) => (
        <Card key={g.id}>
          <CardContent className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="secondary">{g.lines} lines</Badge>
              <Badge variant="secondary">{Math.round(g.similarity * 100)}% similar</Badge>
              <Badge>{g.fileCount} files</Badge>
            </div>
            <pre className="overflow-x-auto rounded border border-border bg-background p-3 text-xs text-foreground">
              {g.fragment}
            </pre>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>File</TableHead>
                    <TableHead>Start</TableHead>
                    <TableHead>End</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {g.blocks.map((b) => (
                    <TableRow key={b.filePath + b.startLine}>
                      <TableCell><span className="text-primary">{b.filePath}</span></TableCell>
                      <TableCell>{b.startLine}</TableCell>
                      <TableCell>{b.endLine}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
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

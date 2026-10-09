import { GitCommitHorizontal, GitMerge } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Badge, Button, Input, Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui'
import DiffStat from '@/components/DiffStat'
import ExportLinks from '@/components/ExportLinks'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader } from '@/components/layout'
import { Empty, ErrorNotice, Loading, QueryView } from '@/components/states'
import { useCommits, useLargestCommits } from '@/lib/api'
import { formatDate, formatRelative, plural, shortHash } from '@/lib/format'
import type { Repository } from '@/lib/types'

interface CommitLike {
  hash: string
  author: string
  email: string
  date: string
  message: string
  filesChanged: number
  insertions: number
  deletions: number
  isMerge?: boolean
}

function CommitRow({ c }: { c: CommitLike }) {
  return (
    <li className="flex flex-wrap items-start justify-between gap-x-6 gap-y-1 border-b border-border/70 py-3 last:border-b-0">
      <div className="min-w-0 flex-1">
        <p className="flex items-center gap-2">
          <span className="truncate font-medium" title={c.message}>
            {c.message || 'No message'}
          </span>
          {c.isMerge && (
            <Badge variant="outline">
              <GitMerge />
              Merge
            </Badge>
          )}
        </p>
        <p className="mt-0.5 flex flex-wrap gap-x-3 text-sm text-muted-foreground">
          <span className="font-mono text-xs leading-5 text-foreground/80">{shortHash(c.hash)}</span>
          <span title={c.email}>{c.author}</span>
          <time dateTime={c.date} title={formatDate(c.date)}>
            {formatRelative(c.date)}
          </time>
        </p>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1 pt-0.5">
        <DiffStat insertions={c.insertions} deletions={c.deletions} />
        <span className="text-xs text-muted-foreground">{plural(c.filesChanged, 'file')}</span>
      </div>
    </li>
  )
}

function matches(c: CommitLike, q: string) {
  const needle = q.toLowerCase()
  return (
    c.message.toLowerCase().includes(needle) ||
    c.author.toLowerCase().includes(needle) ||
    c.email.toLowerCase().includes(needle) ||
    c.hash.startsWith(needle)
  )
}

function Newest({ repoId, filter }: { repoId: number; filter: string }) {
  const q = useCommits(repoId)
  const all = useMemo(() => q.data?.pages.flatMap((p) => p.commits) ?? [], [q.data])
  if (q.isPending) return <Loading label="Loading commits…" />
  if (q.isError) return <ErrorNotice title="Couldn't load commits" error={q.error} onRetry={() => void q.refetch()} />
  const shown = filter ? all.filter((c) => matches(c, filter)) : all
  return (
    <>
      {shown.length ? (
        <ol>
          {shown.map((c) => (
            <CommitRow key={c.hash} c={c} />
          ))}
        </ol>
      ) : (
        <Empty icon={GitCommitHorizontal} title={filter ? 'No loaded commits match' : 'No commits yet'}>
          {filter && q.hasNextPage ? 'Load older commits to search further back.' : null}
        </Empty>
      )}
      <div className="mt-4 flex items-center gap-3 text-sm text-muted-foreground">
        <span>{plural(all.length, 'commit')} loaded</span>
        {q.hasNextPage && (
          <Button variant="outline" size="sm" disabled={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>
            {q.isFetchingNextPage ? 'Loading…' : 'Load older commits'}
          </Button>
        )}
      </div>
    </>
  )
}

function Largest({ repoId, filter }: { repoId: number; filter: string }) {
  const q = useLargestCommits(repoId)
  return (
    <QueryView query={q} label="largest commits">
      {({ commits }) => {
        const shown = filter ? commits.filter((c) => matches(c, filter)) : commits
        return shown.length ? (
          <ol>
            {shown.map((c) => (
              <CommitRow key={c.hash} c={c} />
            ))}
          </ol>
        ) : (
          <Empty icon={GitCommitHorizontal} title="No commits match" />
        )
      }}
    </QueryView>
  )
}

function CommitsPage({ repo }: { repo: Repository }) {
  const [filter, setFilter] = useState('')
  return (
    <>
      <PageHeader
        title="Commits"
        description={`${plural(repo.commitCount, 'commit')} reachable from branches, tags and remotes.`}
        actions={<ExportLinks repoId={repo.id} kind="commits" />}
      />
      <Tabs defaultValue="newest">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b">
          <TabsList className="h-10">
            <TabsTrigger value="newest">Newest first</TabsTrigger>
            <TabsTrigger value="largest">Largest changes</TabsTrigger>
          </TabsList>
          <Input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter by message, author or hash"
            aria-label="Filter commits"
            className="mb-2 w-72"
          />
        </div>
        <TabsContent value="newest">
          <Newest repoId={repo.id} filter={filter.trim()} />
        </TabsContent>
        <TabsContent value="largest">
          <Largest repoId={repo.id} filter={filter.trim()} />
        </TabsContent>
      </Tabs>
    </>
  )
}

export default function Commits() {
  return <RequireRepo>{(repo) => <CommitsPage repo={repo} />}</RequireRepo>
}

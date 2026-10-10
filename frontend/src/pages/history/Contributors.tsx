import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import DiffStat from '@/components/DiffStat'
import ExportLinks from '@/components/ExportLinks'
import Meter from '@/components/Meter'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { QueryView } from '@/components/states'
import { useContributors, useOwnership } from '@/lib/api'
import { formatDate, formatNumber, formatPercent, formatRelative, plural } from '@/lib/format'
import type { Repository } from '@/lib/types'

function Leaderboard({ repoId }: { repoId: number }) {
  const q = useContributors(repoId)
  return (
    <Section title="Commits by author">
      <QueryView query={q} label="contributors">
        {({ contributors }) => {
          if (!contributors.length) return <p className="text-sm text-muted-foreground">No commits yet.</p>
          const total = contributors.reduce((s, c) => s + c.commits, 0)
          const max = contributors[0].commits
          return (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Author</TableHead>
                  <TableHead className="w-56">Commits</TableHead>
                  <TableHead>Lines changed</TableHead>
                  <TableHead>First commit</TableHead>
                  <TableHead>Latest commit</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {contributors.map((c) => (
                  <TableRow key={c.email || c.name}>
                    <TableCell className="max-w-72">
                      <div className="truncate font-medium">{c.name || c.email}</div>
                      <div className="truncate text-xs text-muted-foreground">{c.email}</div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-baseline justify-between gap-2">
                        <span>{formatNumber(c.commits)}</span>
                        <span className="text-xs text-muted-foreground">{formatPercent(c.commits / total)}</span>
                      </div>
                      <Meter value={c.commits} max={max} className="mt-1" />
                    </TableCell>
                    <TableCell>
                      <DiffStat insertions={c.insertions} deletions={c.deletions} />
                    </TableCell>
                    <TableCell className="text-muted-foreground">{formatDate(c.firstCommitAt)}</TableCell>
                    <TableCell className="text-muted-foreground" title={formatDate(c.lastCommitAt)}>
                      {formatRelative(c.lastCommitAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )
        }}
      </QueryView>
    </Section>
  )
}

function Ownership({ repoId }: { repoId: number }) {
  const q = useOwnership(repoId)
  return (
    <Section
      title="File ownership"
      description="Files counted for the author who made most of their commits, following renames."
    >
      <QueryView query={q} label="ownership">
        {({ byAuthor, total }) => {
          if (!byAuthor?.length) return <p className="text-sm text-muted-foreground">No tracked files have history yet.</p>
          const max = byAuthor[0].files
          return (
            <ol className="grid max-w-3xl gap-3">
              {byAuthor.map((o) => (
                <li key={o.email || o.author} className="grid grid-cols-[minmax(8rem,14rem)_1fr_auto] items-center gap-4 text-sm">
                  <span className="truncate font-medium" title={o.email}>{o.author}</span>
                  <Meter value={o.files} max={max} />
                  <span className="w-36 text-right tabular-nums text-muted-foreground">
                    {plural(o.files, 'file')}, {formatPercent(o.share)}
                  </span>
                </li>
              ))}
              <li className="text-xs text-muted-foreground">{plural(total, 'file')} with Git history.</li>
            </ol>
          )
        }}
      </QueryView>
    </Section>
  )
}

function ContributorsPage({ repo }: { repo: Repository }) {
  return (
    <>
      <PageHeader
        title="Contributors"
        description={`${plural(repo.contributorCount, 'person has', 'people have')} committed to ${repo.name}, grouped by email address.`}
        actions={<ExportLinks repoId={repo.id} kind="contributors" />}
      />
      <Leaderboard repoId={repo.id} />
      <Ownership repoId={repo.id} />
    </>
  )
}

export default function Contributors() {
  return <RequireRepo>{(repo) => <ContributorsPage repo={repo} />}</RequireRepo>
}

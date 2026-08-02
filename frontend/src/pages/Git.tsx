import { Activity, Flame, FolderGit2, GitBranch, GitCommitHorizontal, Tag, Users } from 'lucide-react'
import { useState } from 'react'
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner, Tabs, TabsContent, TabsList, TabsTrigger, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import { useCommits, useContributors, useHeatmap, useLargestCommits, useOwnership, useRepo, useBranches, useTags } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'
import RepoSelector from '../components/RepoSelector'

function HeatmapGrid({ heatmap }: { heatmap: import('../lib/types').Heatmap }) {
  const daily = heatmap.daily ?? []
  const max = Math.max(1, ...daily.map((d) => d.count))
  const weeks: { date: string; count: number }[][] = []
  let week: { date: string; count: number }[] = []
  for (const d of heatmap.daily) {
    week.push(d)
    if (week.length === 7) {
      weeks.push(week)
      week = []
    }
  }
  if (week.length) weeks.push(week)
  const level = (count: number) => {
    const r = count / max
    if (count === 0) return 'bg-muted'
    if (r > 0.66) return 'bg-chart-2'
    if (r > 0.33) return 'bg-chart-2/60'
    return 'bg-chart-2/30'
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">
          Commit activity ({heatmap.total} commits, {heatmap.start} → {heatmap.end})
        </CardTitle>
      </CardHeader>
      <CardContent className="flex gap-1 overflow-x-auto">
        {weeks.map((w, i) => (
          <div key={i} className="flex flex-col gap-1">
            {w.map((d) => (
              <div
                key={d.date}
                title={`${d.date}: ${d.count}`}
                className={`size-3 rounded-sm ${level(d.count)}`}
              />
            ))}
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

function HourlyHeatmap({ hourly }: { hourly: number[][] }) {
  const data = hourly.flatMap((row, wd) =>
    row.map((count, hr) => ({ wd, hr, count })),
  )
  const max = Math.max(1, ...data.map((d) => d.count))
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Hour of week</CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={180}>
          <BarChart data={data}>
            <XAxis dataKey="hr" hide />
            <YAxis hide />
            <Tooltip
              contentStyle={{ background: 'var(--popover)', border: '1px solid var(--border)' }}
              labelStyle={{ color: 'var(--popover-foreground)' }}
              formatter={(value) => [Number(value), 'commits']}
              labelFormatter={(_, p) => {
                const d = p?.[0]?.payload as { wd: number; hr: number }
                return d ? `weekday ${d.wd} · ${d.hr}:00` : ''
              }}
            />
            <Bar dataKey="count" radius={[2, 2, 0, 0]} isAnimationActive={false}>
              {data.map((d, i) => (
                <Cell key={i} fill="var(--chart-1)" fillOpacity={d.count / max > 0.5 ? 1 : 0.5} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}

function StreaksView({ streaks }: { streaks: import('../lib/types').StreaksResult[] }) {
  if (!streaks || !streaks.length) {
    return (
      <Card>
        <CardContent>
          <EmptyState icon={Flame} title="No streak data" />
        </CardContent>
      </Card>
    )
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Streaks</CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Author</TableHead>
              <TableHead>Total commits</TableHead>
              <TableHead>Active days</TableHead>
              <TableHead>Longest streak</TableHead>
              <TableHead>Current streak</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {streaks.map((s) => (
              <TableRow key={s.email}>
                <TableCell>{s.email}</TableCell>
                <TableCell>{s.totalCommits}</TableCell>
                <TableCell>{s.activeDays}</TableCell>
                <TableCell>{`${s.longest.start} · ${s.longest.days}d`}</TableCell>
                <TableCell>{s.current.days > 0 ? `${s.current.start} · ${s.current.days}d` : '—'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

function ContributorsChart({
  contributors,
}: {
  contributors: import('../lib/types').Contributor[]
}) {
  const data = contributors.slice(0, 10).map((c) => ({
    name: c.name || c.email,
    commits: c.commits,
  }))
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Top contributors</CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={260}>
          <BarChart data={data} layout="vertical" margin={{ left: 24 }}>
            <XAxis type="number" stroke="var(--muted-foreground)" fontSize={12} />
            <YAxis
              type="category"
              dataKey="name"
              stroke="var(--muted-foreground)"
              fontSize={11}
              width={110}
            />
            <Tooltip
              contentStyle={{ background: 'var(--popover)', border: '1px solid var(--border)' }}
              labelStyle={{ color: 'var(--popover-foreground)' }}
            />
            <Bar dataKey="commits" fill="var(--chart-1)" radius={[0, 4, 4, 0]} isAnimationActive={false} />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}

function CommitsView({ repoId }: { repoId: number }) {
  const { data, isLoading } = useCommits(repoId, 200)
  if (isLoading) return <Spinner />
  const commits = data?.commits ?? []
  if (!commits.length) return <EmptyState icon={GitCommitHorizontal} title="No commits" />
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Hash</TableHead>
            <TableHead>Author</TableHead>
            <TableHead>Date</TableHead>
            <TableHead>Message</TableHead>
            <TableHead>Files</TableHead>
            <TableHead>+</TableHead>
            <TableHead>−</TableHead>
            <TableHead>Merge</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {commits.map((c) => (
            <TableRow key={c.hash}>
              <TableCell><code className="text-xs text-primary">{c.hash.slice(0, 8)}</code></TableCell>
              <TableCell>{c.author}</TableCell>
              <TableCell>{new Date(c.date).toLocaleDateString()}</TableCell>
              <TableCell><span className="max-w-96 truncate">{c.message}</span></TableCell>
              <TableCell>{c.filesChanged}</TableCell>
              <TableCell><span className="text-chart-2">+{c.insertions}</span></TableCell>
              <TableCell><span className="text-destructive">−{c.deletions}</span></TableCell>
              <TableCell>{c.isMerge ? <span className="text-chart-3 text-xs">✓</span> : ''}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function LargestCommitsView({ repoId }: { repoId: number }) {
  const { data, isLoading } = useLargestCommits(repoId)
  if (isLoading) return <Spinner />
  const commits = data?.commits ?? []
  if (!commits.length) return <EmptyState icon={GitCommitHorizontal} title="No commits" />
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Hash</TableHead>
            <TableHead>Author</TableHead>
            <TableHead>Date</TableHead>
            <TableHead>Message</TableHead>
            <TableHead>Files</TableHead>
            <TableHead>+</TableHead>
            <TableHead>−</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {commits.map((c) => (
            <TableRow key={c.hash}>
              <TableCell><code className="text-xs text-primary">{c.hash.slice(0, 8)}</code></TableCell>
              <TableCell>{c.author}</TableCell>
              <TableCell>{new Date(c.date).toLocaleDateString()}</TableCell>
              <TableCell><span className="max-w-96 truncate">{c.message}</span></TableCell>
              <TableCell>{c.filesChanged}</TableCell>
              <TableCell><span className="text-chart-2">+{c.insertions}</span></TableCell>
              <TableCell><span className="text-destructive">−{c.deletions}</span></TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function OwnershipView({ repoId }: { repoId: number }) {
  const { data, isLoading } = useOwnership(repoId)
  if (isLoading) return <Spinner />
  const ownership = data?.byAuthor ?? []
  if (!ownership.length) {
    return (
      <Card>
        <CardContent>
          <EmptyState icon={Users} title="No ownership data" />
        </CardContent>
      </Card>
    )
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">
          File ownership ({ownership.length} authors, {data?.total} files)
        </CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Author</TableHead>
              <TableHead>Files</TableHead>
              <TableHead>Share</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {ownership.map((o) => (
              <TableRow key={o.author}>
                <TableCell>{o.author}</TableCell>
                <TableCell>{o.files}</TableCell>
                <TableCell>{`${(o.share * 100).toFixed(1)}%`}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

function BranchesTagsView({ repoId }: { repoId: number }) {
  const { data: branchesData, isLoading: branchesLoading } = useBranches(repoId)
  const { data: tagsData, isLoading: tagsLoading } = useTags(repoId)
  const branches = branchesData?.branches ?? []
  const tags = tagsData?.tags ?? []
  if (branchesLoading || tagsLoading) return <Spinner />
  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Branches</CardTitle>
        </CardHeader>
        <CardContent>
          {branches.length === 0 ? (
            <EmptyState icon={GitBranch} title="No branches" />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Commit</TableHead>
                    <TableHead>Current</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {branches.map((b) => (
                    <TableRow key={b.name}>
                      <TableCell><span className={b.isCurrent ? 'text-primary font-medium' : ''}>{b.name}</span></TableCell>
                      <TableCell><code className="text-xs text-muted-foreground">{b.commitHash.slice(0, 8)}</code></TableCell>
                      <TableCell>{b.isCurrent ? <span className="text-chart-2 text-xs">HEAD</span> : ''}</TableCell>
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
          <CardTitle className="text-sm">Tags</CardTitle>
        </CardHeader>
        <CardContent>
          {tags.length === 0 ? (
            <EmptyState icon={Tag} title="No tags" />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Commit</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tags.map((t) => (
                    <TableRow key={t.name}>
                      <TableCell>{t.name}</TableCell>
                      <TableCell><code className="text-xs text-muted-foreground">{t.commitHash.slice(0, 8)}</code></TableCell>
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

function ActivityTab({ repoId }: { repoId: number }) {
  const { data, isLoading } = useHeatmap(repoId)
  if (isLoading) return <Spinner />
  if (!data) return <EmptyState icon={Activity} title="No activity yet" />
  return (
    <div className="flex flex-col gap-4">
      <HeatmapGrid heatmap={data.heatmap} />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <HourlyHeatmap hourly={data.heatmap.hourly} />
        <StreaksView streaks={data.streaks} />
      </div>
    </div>
  )
}

function ContributorsTab({ repoId }: { repoId: number }) {
  const { data, isLoading } = useContributors(repoId)
  if (isLoading) return <Spinner />
  const contributors = data?.contributors ?? []
  if (!contributors.length) return <EmptyState icon={Users} title="No contributors" />
  return (
    <div className="flex flex-col gap-4">
      <ContributorsChart contributors={contributors} />
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Commits</TableHead>
              <TableHead>Insertions</TableHead>
              <TableHead>Deletions</TableHead>
              <TableHead>First commit</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {contributors.map((c) => (
              <TableRow key={c.email}>
                <TableCell>{c.name}</TableCell>
                <TableCell>{c.email}</TableCell>
                <TableCell>{c.commits}</TableCell>
                <TableCell><span className="text-chart-2">+{c.insertions}</span></TableCell>
                <TableCell><span className="text-destructive">−{c.deletions}</span></TableCell>
                <TableCell>{new Date(c.firstCommitAt).toLocaleDateString()}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function GitInner() {
  const { repoId } = useRepoContext()
  const [tab, setTab] = useState<'activity' | 'commits' | 'contributors' | 'largest' | 'ownership' | 'branches-tags'>('activity')
  const repo = useRepo(repoId).data
  if (repoId === 0) return <EmptyState icon={FolderGit2} title="Scan a repository first" />
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Git</h1>
        <RepoSelector />
      </div>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList className="w-full justify-start">
          <TabsTrigger value="activity">Activity</TabsTrigger>
          <TabsTrigger value="commits">Commits</TabsTrigger>
          <TabsTrigger value="largest">Largest</TabsTrigger>
          <TabsTrigger value="contributors">Contributors</TabsTrigger>
          <TabsTrigger value="ownership">Ownership</TabsTrigger>
          <TabsTrigger value="branches-tags">Branches & Tags</TabsTrigger>
        </TabsList>
        <TabsContent value="activity"><ActivityTab repoId={repoId} /></TabsContent>
        <TabsContent value="commits"><CommitsView repoId={repoId} /></TabsContent>
        <TabsContent value="largest"><LargestCommitsView repoId={repoId} /></TabsContent>
        <TabsContent value="contributors"><ContributorsTab repoId={repoId} /></TabsContent>
        <TabsContent value="ownership"><OwnershipView repoId={repoId} /></TabsContent>
        <TabsContent value="branches-tags"><BranchesTagsView repoId={repoId} /></TabsContent>
      </Tabs>
    </div>
  )
}

export default function Git() {
  return (
    <RepoProvider>
      <GitInner />
    </RepoProvider>
  )
}

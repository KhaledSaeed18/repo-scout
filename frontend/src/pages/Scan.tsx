import { ListTodo } from 'lucide-react'
import { useState } from 'react'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  Input,
  Progress,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import { api, useCreateRepo, useJobs } from '../lib/api'
import { RepoProvider } from '../lib/repoctx'
import type { Job } from '../lib/types'

const jobActions = ['pause', 'resume', 'cancel'] as const

function statusVariant(s: string): 'success' | 'warning' | 'secondary' | 'destructive' {
  switch (s) {
    case 'completed':
      return 'success'
    case 'running':
    case 'queued':
      return 'warning'
    case 'paused':
      return 'secondary'
    case 'failed':
    case 'cancelled':
    case 'interrupted':
      return 'destructive'
    default:
      return 'secondary'
  }
}

function JobRow({ job }: { job: Job }) {
  const act = async (action: (typeof jobActions)[number]) => {
    try {
      await api.jobAction(job.id, action)
    } catch {
      /* ignore */
    }
  }
  const buttons = jobActions
    .filter((a) =>
      a === 'resume'
        ? job.status === 'paused'
        : job.status === 'running' || job.status === 'queued',
    )
    .map((a) => (
      <Button key={a} variant="outline" className="text-xs" onClick={() => void act(a)}>
        {a}
      </Button>
    ))
  return (
    <TableRow key={job.id}>
      <TableCell>#{job.id}</TableCell>
      <TableCell>{job.repoId}</TableCell>
      <TableCell>{job.kind}</TableCell>
      <TableCell>
        <Badge variant={statusVariant(job.status)}>
          {job.status}
        </Badge>
      </TableCell>
      <TableCell>
        <div className="w-40">
          <Progress value={Math.round(job.progress * 100)} />
        </div>
      </TableCell>
      <TableCell className="block max-w-56 truncate text-xs text-muted-foreground">
        {job.message}
      </TableCell>
      <TableCell>
        <div className="flex gap-1">
          {buttons}
        </div>
      </TableCell>
    </TableRow>
  )
}

function ScanPage() {
  const [path, setPath] = useState('')
  const create = useCreateRepo()
  const { data, isLoading } = useJobs(1500)
  const jobs = data?.jobs ?? []

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (path.trim()) create.mutate(path.trim())
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-xl font-semibold">Scan</h1>
      <Card>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-3">
            <label className="text-sm text-muted-foreground">
              Local repository path
              <Input
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/Users/me/projects/my-repo"
                className="mt-1"
              />
            </label>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={create.isPending || !path.trim()}>
                {create.isPending && <Spinner />}
                Scan repository
              </Button>
              {create.isError && (
                <span className="text-sm text-destructive">{create.error.message}</span>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Jobs</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Spinner />
          ) : jobs.length === 0 ? (
            <EmptyState icon={ListTodo} title="No jobs yet" />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Repo</TableHead>
                    <TableHead>Kind</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Progress</TableHead>
                    <TableHead>Message</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {jobs.map((j) => JobRow({ job: j }))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function Scan() {
  return (
    <RepoProvider>
      <ScanPage />
    </RepoProvider>
  )
}

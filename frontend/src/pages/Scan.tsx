import { useState } from 'react'
import { Badge, Button, Card, Empty, Input, Progress, Spinner, Table } from '../components/ui'
import { api, useCreateRepo, useJobs } from '../lib/api'
import { RepoProvider } from '../lib/repoctx'
import type { Job } from '../lib/types'

const jobActions = ['pause', 'resume', 'cancel'] as const

function statusColor(s: string) {
  switch (s) {
    case 'completed':
      return 'bg-emerald-500/20 text-emerald-400 border-emerald-800'
    case 'running':
    case 'queued':
      return 'bg-amber-500/20 text-amber-400 border-amber-800'
    case 'paused':
      return 'bg-sky-500/20 text-sky-400 border-sky-800'
    case 'failed':
    case 'cancelled':
    case 'interrupted':
      return 'bg-rose-500/20 text-rose-400 border-rose-800'
    default:
      return ''
  }
}

function JobRow({ job }: { job: Job }) {
  const [busy, setBusy] = useState(false)
  const act = async (action: (typeof jobActions)[number]) => {
    setBusy(true)
    try {
      await api.jobAction(job.id, action)
    } finally {
      setBusy(false)
    }
  }
  const buttons = jobActions
    .filter((a) =>
      a === 'resume'
        ? job.status === 'paused'
        : job.status === 'running' || job.status === 'queued',
    )
    .map((a) => (
      <Button key={a} variant="outline" className="text-xs" disabled={busy} onClick={() => void act(a)}>
        {a}
      </Button>
    ))
  return [
    <span key="id">#{job.id}</span>,
    <span key="repo">{job.repoId}</span>,
    <span key="kind">{job.kind}</span>,
    <Badge key="status" className={statusColor(job.status)}>
      {job.status}
    </Badge>,
    <div key="progress" className="w-40">
      <Progress value={job.progress} />
    </div>,
    <span key="msg" className="block max-w-56 truncate text-xs text-slate-400">
      {job.message}
    </span>,
    <div key="actions" className="flex gap-1">
      {buttons}
    </div>,
  ]
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
        <form onSubmit={submit} className="flex flex-col gap-3">
          <label className="text-sm text-slate-300">
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
              <span className="text-sm text-rose-400">{create.error.message}</span>
            )}
          </div>
        </form>
      </Card>

      <Card>
        <h2 className="mb-3 text-sm font-medium text-slate-300">Jobs</h2>
        {isLoading ? (
          <Spinner />
        ) : jobs.length === 0 ? (
          <Empty message="No jobs yet" />
        ) : (
          <Table
            headers={['ID', 'Repo', 'Kind', 'Status', 'Progress', 'Message', 'Actions']}
            rows={jobs.map((j) => JobRow({ job: j }))}
          />
        )}
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

import { FolderGit2, History, RefreshCw, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Badge,
  Button,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  Input,
  Label,
  Progress,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import FolderPicker from '@/components/FolderPicker'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useCreateRepo, useDeleteRepo, useJobAction, useJobs } from '@/lib/api'
import { formatNumber, formatRelative } from '@/lib/format'
import { useRepoContext } from '@/lib/repo-context'
import type { Job, Repository } from '@/lib/types'

const repoStatus: Record<Repository['status'], { label: string; variant: 'success' | 'warning' | 'destructive' }> = {
  ready: { label: 'Ready', variant: 'success' },
  scanning: { label: 'Scanning', variant: 'warning' },
  failed: { label: 'Failed', variant: 'destructive' },
}

const jobStatus: Record<string, { label: string; variant: 'success' | 'warning' | 'destructive' | 'secondary' }> = {
  queued: { label: 'Queued', variant: 'secondary' },
  running: { label: 'Running', variant: 'warning' },
  paused: { label: 'Paused', variant: 'secondary' },
  cancelling: { label: 'Cancelling', variant: 'secondary' },
  completed: { label: 'Completed', variant: 'success' },
  cancelled: { label: 'Cancelled', variant: 'secondary' },
  failed: { label: 'Failed', variant: 'destructive' },
  interrupted: { label: 'Interrupted', variant: 'destructive' },
}

function AddRepository() {
  const [path, setPath] = useState('')
  const create = useCreateRepo()
  const { setRepoId } = useRepoContext()

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = path.trim()
    if (!trimmed) return
    create.mutate(trimmed, {
      onSuccess: ({ repository }) => {
        setRepoId(repository.id)
        setPath('')
      },
    })
  }

  return (
    <form onSubmit={submit} className="flex max-w-3xl flex-col gap-2">
      <Label htmlFor="repo-path">Folder of a Git repository on this machine</Label>
      <div className="flex flex-wrap gap-2">
        <Input
          id="repo-path"
          value={path}
          onChange={(e) => setPath(e.target.value)}
          placeholder="/Users/you/code/project"
          className="min-w-64 flex-1 font-mono text-sm"
          autoComplete="off"
          spellCheck={false}
        />
        <FolderPicker onSelect={setPath} />
        <Button type="submit" disabled={create.isPending || !path.trim()}>
          {create.isPending && <Spinner className="border-primary-foreground/40 border-t-primary-foreground" />}
          Scan repository
        </Button>
      </div>
      {create.isError && <p className="text-sm text-destructive">{create.error.message}</p>}
    </form>
  )
}

function DeleteRepository({ repo }: { repo: Repository }) {
  const del = useDeleteRepo()
  return (
    <Dialog>
      <DialogTrigger
        render={<Button variant="ghost" size="icon-sm" aria-label={`Remove ${repo.name}`} title="Remove" />}
      >
        <Trash2 />
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Remove {repo.name}?</DialogTitle>
          <DialogDescription>
            This deletes the scan results stored by Repo Scout. The folder on disk is not touched, and you can
            scan it again later.
          </DialogDescription>
        </DialogHeader>
        {del.isError && <p className="text-sm text-destructive">{del.error.message}</p>}
        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Keep it</DialogClose>
          <Button variant="destructive" disabled={del.isPending} onClick={() => del.mutate(repo.id)}>
            Remove scan data
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function RepositoryTable({ repos }: { repos: Repository[] }) {
  const create = useCreateRepo()
  const { setRepoId } = useRepoContext()
  const navigate = useNavigate()
  if (!repos.length) {
    return (
      <Empty icon={FolderGit2} title="Nothing scanned yet">
        Pick a folder above. Scans read the files and Git history locally; nothing leaves this machine.
      </Empty>
    )
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Repository</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Files</TableHead>
          <TableHead className="text-right">Lines of code</TableHead>
          <TableHead className="text-right">Commits</TableHead>
          <TableHead>Last scan</TableHead>
          <TableHead>
            <span className="sr-only">Actions</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {repos.map((r) => {
          const status = repoStatus[r.status]
          return (
            <TableRow key={r.id}>
              <TableCell className="max-w-80">
                <button
                  className="cursor-pointer text-left font-medium text-primary hover:underline"
                  onClick={() => {
                    setRepoId(r.id)
                    navigate('/')
                  }}
                >
                  {r.name}
                </button>
                <div className="truncate font-mono text-xs text-muted-foreground" title={r.path}>
                  {r.path}
                </div>
              </TableCell>
              <TableCell>
                <Badge variant={status.variant}>{status.label}</Badge>
              </TableCell>
              <TableCell className="text-right">{formatNumber(r.fileCount)}</TableCell>
              <TableCell className="text-right">{formatNumber(r.totalCode)}</TableCell>
              <TableCell className="text-right">{formatNumber(r.commitCount)}</TableCell>
              <TableCell className="text-muted-foreground">
                {r.lastScannedAt ? formatRelative(r.lastScannedAt) : 'Never'}
              </TableCell>
              <TableCell>
                <div className="flex justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Scan ${r.name} again`}
                    title="Scan again"
                    disabled={r.status === 'scanning' || create.isPending}
                    onClick={() => create.mutate(r.path)}
                  >
                    <RefreshCw />
                  </Button>
                  <DeleteRepository repo={r} />
                </div>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}

function JobActions({ job }: { job: Job }) {
  const action = useJobAction()
  const actions: ('pause' | 'resume' | 'cancel')[] =
    job.status === 'running' ? ['pause', 'cancel'] : job.status === 'paused' ? ['resume', 'cancel'] : job.status === 'queued' ? ['cancel'] : []
  if (!actions.length) return null
  return (
    <div className="flex justify-end gap-1">
      {actions.map((a) => (
        <Button
          key={a}
          variant="ghost"
          size="sm"
          disabled={action.isPending}
          onClick={() => action.mutate({ id: job.id, action: a })}
        >
          {a[0].toUpperCase() + a.slice(1)}
        </Button>
      ))}
    </div>
  )
}

function ScanHistory({ jobs, repos }: { jobs: Job[]; repos: Repository[] }) {
  if (!jobs.length) {
    return (
      <Empty icon={History} title="No scans yet">
        Every scan you start shows up here with its progress.
      </Empty>
    )
  }
  const nameOf = (id: number) => repos.find((r) => r.id === id)?.name ?? 'Removed repository'
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Repository</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="w-48">Progress</TableHead>
          <TableHead>Started</TableHead>
          <TableHead>
            <span className="sr-only">Actions</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {jobs.map((job) => {
          const status = jobStatus[job.status] ?? { label: job.status, variant: 'secondary' as const }
          const live = job.status === 'running' || job.status === 'paused'
          return (
            <TableRow key={job.id}>
              <TableCell className="font-medium">{nameOf(job.repoId)}</TableCell>
              <TableCell>
                <Badge variant={status.variant}>{status.label}</Badge>
                {job.error && <p className="mt-1 max-w-80 truncate text-xs text-destructive" title={job.error}>{job.error}</p>}
              </TableCell>
              <TableCell>
                {live ? (
                  <div className="flex items-center gap-2">
                    <Progress value={Math.round(job.progress * 100)} className="flex-1" aria-label="Progress" />
                    <span className="w-9 text-right text-xs text-muted-foreground">{Math.round(job.progress * 100)}%</span>
                  </div>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
                {live && job.message && <p className="mt-1 truncate text-xs text-muted-foreground">{job.message}</p>}
              </TableCell>
              <TableCell className="text-muted-foreground">{formatRelative(job.startedAt ?? job.createdAt)}</TableCell>
              <TableCell>
                <JobActions job={job} />
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}

export default function Repositories() {
  const { repos } = useRepoContext()
  const jobs = useJobs()
  return (
    <>
      <PageHeader
        title="Repositories"
        description="Scan a local Git repository to explore its history, code and structure. Scans run on this machine and results stay here."
      />
      <Section title="Add a repository">
        <AddRepository />
      </Section>
      <Section title="Scanned repositories">
        <RepositoryTable repos={repos} />
      </Section>
      <Section title="Scan history" description="The 20 most recent scans.">
        <QueryView query={jobs} label="scans">
          {(data) => <ScanHistory jobs={data.jobs.slice(0, 20)} repos={repos} />}
        </QueryView>
      </Section>
    </>
  )
}

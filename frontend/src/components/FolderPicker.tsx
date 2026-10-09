import { ArrowUp, Folder, FolderGit2, FolderX, House } from 'lucide-react'
import { useState } from 'react'
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui'
import { useBrowse } from '@/lib/api'
import { cn } from '@/lib/utils'
import { Empty, Loading } from './states'

/** Dialog for picking a local folder. Git repositories are marked. */
export default function FolderPicker({ onSelect }: { onSelect: (path: string) => void }) {
  const [open, setOpen] = useState(false)
  const [path, setPath] = useState('')
  const [highlighted, setHighlighted] = useState('')
  const { data, isPending, isError, error } = useBrowse(path, open)

  const go = (next: string) => {
    setPath(next)
    setHighlighted('')
  }
  const choose = (p: string) => {
    onSelect(p)
    setOpen(false)
  }
  const target = highlighted || data?.path || ''
  const targetIsRepo = highlighted ? !!data?.entries.find((e) => e.path === highlighted)?.isRepo : !!data?.isRepo

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) go('')
      }}
    >
      <DialogTrigger render={<Button type="button" variant="outline" />}>Browse…</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Choose a repository folder</DialogTitle>
          <DialogDescription>Folders with Git history are marked. Double-click a folder to open it.</DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-1.5">
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            aria-label="Up one folder"
            disabled={!data?.parent}
            onClick={() => data?.parent && go(data.parent)}
          >
            <ArrowUp />
          </Button>
          <Button type="button" variant="outline" size="icon-sm" aria-label="Home folder" onClick={() => go('')}>
            <House />
          </Button>
          <p className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground" title={data?.path}>
            {data?.path}
          </p>
        </div>

        <div className="h-72 overflow-y-auto rounded-md border bg-card p-1" role="listbox" aria-label="Folders">
          {isPending ? (
            <Loading label="Reading folder…" className="justify-center" />
          ) : isError ? (
            <Empty icon={FolderX} title="Can't open this folder" className="border-0">
              {error.message}
            </Empty>
          ) : data.entries.length === 0 ? (
            <Empty icon={Folder} title="No folders inside" className="border-0" />
          ) : (
            data.entries.map((entry) => {
              const Icon = entry.isRepo ? FolderGit2 : Folder
              return (
                <button
                  key={entry.path}
                  type="button"
                  role="option"
                  aria-selected={highlighted === entry.path}
                  onClick={() => setHighlighted(entry.path)}
                  onDoubleClick={() => go(entry.path)}
                  onKeyDown={(e) => e.key === 'Enter' && go(entry.path)}
                  className={cn(
                    'flex w-full cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm hover:bg-secondary',
                    highlighted === entry.path && 'bg-accent text-accent-foreground hover:bg-accent',
                  )}
                >
                  <Icon className={cn('size-4 shrink-0', entry.isRepo ? 'text-primary' : 'text-muted-foreground')} aria-hidden />
                  <span className="truncate">{entry.name}</span>
                  {entry.isRepo && (
                    <Badge variant="outline" className="ml-auto">
                      Git
                    </Badge>
                  )}
                </button>
              )
            })
          )}
        </div>

        <DialogFooter>
          <p className="mr-auto self-center truncate text-xs text-muted-foreground">
            {target && !targetIsRepo ? 'This folder has no Git history of its own.' : null}
          </p>
          <Button type="button" variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={!target} onClick={() => choose(target)}>
            Use this folder
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

import { useState } from 'react'
import { ChevronUp, FolderClosed, FolderX, House } from 'lucide-react'
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  EmptyState,
  ScrollArea,
  Spinner,
} from '@/components/ui'
import { useBrowse } from '../lib/api'

export default function FolderPicker({ onSelect }: { onSelect: (path: string) => void }) {
  const [open, setOpen] = useState(false)
  const [path, setPath] = useState('')
  const { data, isLoading, isError, error } = useBrowse(path, open)

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) setPath('')
      }}
    >
      <DialogTrigger render={<Button type="button" variant="outline" />}>
        Browse…
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Choose a folder</DialogTitle>
          <DialogDescription className="truncate font-mono">
            {data?.path ?? ' '}
          </DialogDescription>
        </DialogHeader>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            disabled={!data?.parent}
            onClick={() => data?.parent && setPath(data.parent)}
          >
            <ChevronUp />
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={() => setPath('')}>
            <House />
            Home
          </Button>
        </div>
        <ScrollArea className="h-72 rounded-md border border-border">
          {isLoading ? (
            <div className="flex justify-center p-6">
              <Spinner />
            </div>
          ) : isError ? (
            <EmptyState icon={FolderX} title="Can't open this folder" description={(error as Error).message} />
          ) : !data?.entries.length ? (
            <EmptyState icon={FolderClosed} title="No subfolders here" />
          ) : (
            <ul className="flex flex-col gap-0.5 p-1">
              {data.entries.map((entry) => (
                <li key={entry.path}>
                  <button
                    type="button"
                    onClick={() => setPath(entry.path)}
                    className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-foreground hover:bg-muted"
                  >
                    <FolderClosed className="size-4 shrink-0 text-muted-foreground" />
                    <span className="truncate">{entry.name}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </ScrollArea>
        <DialogFooter>
          <Button
            type="button"
            disabled={!data?.path}
            onClick={() => {
              if (data?.path) {
                onSelect(data.path)
                setOpen(false)
              }
            }}
          >
            Select this folder
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

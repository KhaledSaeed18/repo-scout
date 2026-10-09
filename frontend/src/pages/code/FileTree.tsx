import { ChevronRight, File, Folder, FolderOpen } from 'lucide-react'
import { useState } from 'react'
import { Spinner } from '@/components/ui'
import { useTree } from '@/lib/api'
import { formatCompact } from '@/lib/format'
import { cn } from '@/lib/utils'

interface TreeProps {
  repoId: number
  selected: string
  onSelect: (path: string) => void
}

const indent = (depth: number) => ({ paddingLeft: `${depth * 14 + 6}px` })

function Level({ repoId, folder, depth, selected, onSelect }: TreeProps & { folder: string; depth: number }) {
  const { data, isPending, isError } = useTree(repoId, folder)
  if (isPending) {
    return (
      <div className="py-1" style={indent(depth)}>
        <Spinner className="size-3" />
      </div>
    )
  }
  if (isError) return <p className="py-1 text-xs text-destructive" style={indent(depth)}>Couldn't load this folder.</p>
  return (
    <ul role="group">
      {data.folders.map((name) => {
        const path = folder ? `${folder}/${name}` : name
        return (
          <FolderNode
            key={path}
            repoId={repoId}
            path={path}
            name={name}
            depth={depth}
            selected={selected}
            onSelect={onSelect}
          />
        )
      })}
      {data.files.map((f) => (
        <li key={f.path} role="treeitem" aria-selected={f.path === selected}>
          <button
            onClick={() => onSelect(f.path)}
            className={cn(
              'flex w-full cursor-pointer items-center gap-1.5 rounded-sm py-1 pr-2 text-left text-sm hover:bg-secondary',
              f.path === selected && 'bg-accent text-accent-foreground hover:bg-accent',
            )}
            style={indent(depth + 1)}
          >
            <File className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            <span className="truncate">{f.name}</span>
            {f.linesCode > 0 && (
              <span className="ml-auto shrink-0 pl-2 text-xs tabular-nums text-muted-foreground">
                {formatCompact(f.linesCode)}
              </span>
            )}
          </button>
        </li>
      ))}
    </ul>
  )
}

function FolderNode({ repoId, path, name, depth, selected, onSelect }: TreeProps & { path: string; name: string; depth: number }) {
  const [open, setOpen] = useState(() => selected.startsWith(`${path}/`))
  const Icon = open ? FolderOpen : Folder
  return (
    <li role="treeitem" aria-expanded={open} aria-selected={false}>
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full cursor-pointer items-center gap-1.5 rounded-sm py-1 pr-2 text-left text-sm hover:bg-secondary"
        style={indent(depth)}
      >
        <ChevronRight className={cn('size-3.5 shrink-0 text-muted-foreground transition-transform', open && 'rotate-90')} aria-hidden />
        <Icon className="size-3.5 shrink-0 text-primary/80" aria-hidden />
        <span className="truncate">{name}</span>
      </button>
      {open && <Level repoId={repoId} folder={path} depth={depth + 1} selected={selected} onSelect={onSelect} />}
    </li>
  )
}

/** Lazily loaded folder tree. Folders on the path to the selection start open. */
export default function FileTree(props: TreeProps) {
  return (
    <div role="tree" aria-label="Files">
      <Level {...props} folder="" depth={0} />
    </div>
  )
}

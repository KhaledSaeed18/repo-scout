import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui'
import { useRepoContext } from '@/lib/repo-context'

export default function RepoSelector() {
  const { repos, repoId, setRepoId } = useRepoContext()
  if (!repos.length) return null
  return (
    <Select value={String(repoId)} onValueChange={(val) => setRepoId(Number(val))}>
      <SelectTrigger className="w-64">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {repos.map((r) => (
          <SelectItem key={r.id} value={String(r.id)}>
            {r.name} {r.status === 'ready' ? '' : `(${r.status})`}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui'
import { useRepoContext } from '../lib/repoctx'

export default function RepoSelector() {
  const { repos, repoId, setRepoId } = useRepoContext()
  if (!repos?.repositories.length) return null
  return (
    <Select value={String(repoId)} onValueChange={(val) => setRepoId(Number(val))}>
      <SelectTrigger className="w-64">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {repos.repositories.map((r) => (
          <SelectItem key={r.id} value={String(r.id)}>
            {r.name} {r.status === 'ready' ? '' : `(${r.status})`}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

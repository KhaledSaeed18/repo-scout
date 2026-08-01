import { Select } from '../components/ui'
import { useRepoContext } from '../lib/repoctx'

export default function RepoSelector() {
  const { repos, repoId, setRepoId } = useRepoContext()
  if (!repos?.repositories.length) return null
  return (
    <Select
      value={repoId}
      onChange={(e) => setRepoId(Number(e.target.value))}
      className="w-64"
    >
      {repos.repositories.map((r) => (
        <option key={r.id} value={r.id}>
          {r.name} {r.status === 'ready' ? '' : `(${r.status})`}
        </option>
      ))}
    </Select>
  )
}

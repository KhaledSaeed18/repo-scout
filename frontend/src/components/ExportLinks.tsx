import { Download } from 'lucide-react'
import { buttonVariants } from '@/components/ui'
import { api } from '@/lib/api'

/** Download links for one of the backend's export kinds. */
export default function ExportLinks({ repoId, kind }: { repoId: number; kind: 'files' | 'commits' | 'contributors' }) {
  return (
    <div className="flex gap-1">
      {(['csv', 'json'] as const).map((format) => (
        <a
          key={format}
          href={api.exportUrl(repoId, kind, format)}
          download
          className={buttonVariants({ variant: 'outline', size: 'sm' })}
        >
          <Download />
          {format.toUpperCase()}
        </a>
      ))}
    </div>
  )
}

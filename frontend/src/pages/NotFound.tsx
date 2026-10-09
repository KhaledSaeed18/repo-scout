import { Compass } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import { buttonVariants } from '@/components/ui'
import { Empty } from '@/components/states'

export default function NotFound() {
  const { pathname } = useLocation()
  return (
    <Empty icon={Compass} title="There is no page here" className="mt-4">
      <p className="mb-3">
        Nothing lives at <code className="font-mono">{pathname}</code>. It may be an old link.
      </p>
      <Link to="/" className={buttonVariants({ variant: 'outline' })}>
        Go to the overview
      </Link>
    </Empty>
  )
}

import { Link } from 'react-router-dom'
import { UserMenu } from '@/components/UserMenu'

export function BrandBar() {
  return (
    <div className="mb-6 flex items-center justify-between">
      <Link to="/workspaces" className="font-medium">
        beancount-gs
      </Link>
      <UserMenu align="end" />
    </div>
  )
}

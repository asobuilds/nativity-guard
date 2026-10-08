import { Link } from 'react-router-dom'
import { Card } from '@/components/ui/Card'

export function AdminSettingsPage() {
  return <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
    <header><h1 className="text-xl font-semibold text-ink">Unit settings</h1><p className="mt-1 text-sm text-ink-muted">Platform-wide settings and email templates are managed by super administrators. Unit administrators cannot change global configuration.</p></header>
    <Card className="p-5 space-y-3"><p className="text-sm text-ink-muted">To manage your unit, use the unit-specific management pages. No global settings will be modified from this page.</p><Link className="text-sm font-semibold text-signal underline" to="/admin">Return to unit administration</Link></Card>
  </div>
}

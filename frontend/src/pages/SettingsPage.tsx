import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '@/auth/AuthContext'
import { Card } from '@/components/ui/Card'
import { readPreferences, savePreferences, type Accent, type TextSize } from '@/lib/preferences'

export function SettingsPage() {
  const { role } = useAuth()
  const [preferences, setPreferences] = useState(readPreferences)

  function change(next: Partial<typeof preferences>) {
    const updated = { ...preferences, ...next }
    setPreferences(updated)
    savePreferences(updated)
  }

  return (
    <div className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header><h1 className="text-2xl font-semibold text-ink">Your settings</h1><p className="mt-2 text-sm text-ink-muted">Choose how Nativity Guard looks on this browser. These choices save automatically on this device.</p></header>
      <Card className="space-y-4 p-5">
        <h2 className="font-semibold text-ink">Appearance</h2>
        <div>
          <label htmlFor="setting-accent" className="mb-1 block text-sm text-ink">Accent colour</label>
          <select id="setting-accent" value={preferences.accent} onChange={(event) => change({ accent: event.target.value as Accent })} className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-ink sm:w-64">
            <option value="dawn">Dawn gold</option><option value="sky">Sky blue</option><option value="forest">Forest green</option>
          </select>
        </div>
        <div>
          <label htmlFor="setting-text" className="mb-1 block text-sm text-ink">Text size</label>
          <select id="setting-text" value={preferences.textSize} onChange={(event) => change({ textSize: event.target.value as TextSize })} className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-ink sm:w-64">
            <option value="standard">Standard</option><option value="large">Larger</option>
          </select>
        </div>
        <p className="text-xs text-ink-muted">Emergency red and case status colours do not change with your accent.</p>
      </Card>
      <Card className="space-y-3 p-5">
        <h2 className="font-semibold text-ink">Account and safety</h2>
        <div className="flex flex-col gap-2 text-sm text-signal">
          <Link to="/profile" className="hover:underline">Edit your profile</Link>
          <Link to="/sessions" className="hover:underline">Manage signed-in devices</Link>
          <Link to="/notifications" className="hover:underline">View notifications</Link>
          {role === 'citizen' ? <Link to="/subscriptions" className="hover:underline">Manage alert subscriptions</Link> : null}
          {role === 'unit_admin' ? <Link to="/admin/settings" className="hover:underline">Unit settings</Link> : null}
          {role === 'super_admin' ? <Link to="/super/settings" className="hover:underline">Platform settings</Link> : null}
        </div>
      </Card>
      <Card className="space-y-2 p-5"><h2 className="font-semibold text-ink">Rules and privacy</h2><p className="text-sm text-ink-muted">Read the rules for your account type and how to use reports, evidence, and community features responsibly.</p><Link to="/terms" className="inline-block text-sm text-signal hover:underline">Read user terms and role guidance</Link></Card>
    </div>
  )
}

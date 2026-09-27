import { useState, type FormEvent } from 'react'
import { useAuth } from '@/auth/AuthContext'
import { api } from '@/lib/apiClient'
import { USE_MOCKS } from '@/mocks/config'
import { ErrorState } from '@/components/ui/States'
import type { User } from '@/types/api'

export function ProfileDemoPage() {
  const { user, refreshProfile } = useAuth()
  const [form, setForm] = useState({ firstName: user?.firstName ?? '', lastName: user?.lastName ?? '', phone: user?.phone ?? '', photoUrl: user?.photoUrl ?? '' })
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  if (!USE_MOCKS) return <div className="p-6"><ErrorState title="Demo profile unavailable" description="Use the Profile page for live editing. This older demo uses browser-session data only." /></div>
  const save = async (e: FormEvent) => {
    e.preventDefault(); setError(''); setMessage('')
    try { await api.put<{ user: User }>('/demo/profile', form); await refreshProfile(); setMessage('Demo profile updated for this session.') }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not update profile') }
  }
  return <div className="mx-auto max-w-xl space-y-5 p-5 md:p-8"><p className="text-xs font-bold uppercase tracking-widest text-signal">Session demo · no live changes</p><h1 className="text-2xl font-bold text-ink">Edit profile</h1><p className="text-sm text-ink-muted">Name, contact and photo URL are stored only in this demo session. Use an HTTPS image URL for the photo.</p>
    {form.photoUrl.startsWith('https://') && <img src={form.photoUrl} alt="Profile preview" className="size-20 rounded-full object-cover" />}
    <form onSubmit={(e) => void save(e)} className="space-y-4 rounded-xl border border-border bg-surface p-5">{([['First name','firstName'],['Last name','lastName'],['Contact phone','phone'],['Photo URL (HTTPS)','photoUrl']] as const).map(([label,key]) => <label key={key} className="block text-sm text-ink-muted">{label}<input className="mt-1 w-full rounded-lg border border-border bg-base px-3 py-2 text-ink" value={form[key]} onChange={(e) => setForm((previous) => ({ ...previous, [key]: e.target.value }))} /></label>)}<button className="rounded-lg bg-signal px-4 py-2 font-semibold">Save demo profile</button></form>{message && <p role="status" className="text-sm text-signal">{message}</p>}{error && <p role="alert" className="text-sm text-emergency">{error}</p>}
  </div>
}

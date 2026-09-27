import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Shield } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Field, Input } from '@/components/ui/Field'
import { api, ApiError } from '@/lib/apiClient'

type InviteScope = 'platform' | 'unit'
type CreatedInvite = { invite: { code: string; scope: InviteScope; expiresAt: string | null; maxUses: number }; warning: string }
type ValidatedInvite = { invite: { scope: InviteScope; unitName?: string } }

function message(error: unknown) {
  return error instanceof ApiError ? error.message : 'Could not reach the invitation service. Please try again.'
}

/** A code is returned only once by the server. Keep it in component state, not browser storage. */
export function InvitationsPage() {
  const [scope, setScope] = useState<InviteScope>('platform')
  const [unitId, setUnitId] = useState('')
  const [expiresInHours, setExpiresInHours] = useState('')
  const [maxUses, setMaxUses] = useState('1')
  const [created, setCreated] = useState<CreatedInvite | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function create(event: FormEvent) {
    event.preventDefault()
    setCreated(null)
    setError(null)
    if (scope === 'unit' && !unitId.trim()) { setError('Enter the verified unit ID.'); return }
    setPending(true)
    try {
      const result = await api.post<CreatedInvite>('/invites', {
        scope,
        ...(scope === 'unit' ? { unitId: unitId.trim() } : {}),
        ...(expiresInHours ? { expiresInHours: Number(expiresInHours) } : {}),
        maxUses: Number(maxUses),
      })
      setCreated(result)
    } catch (cause) { setError(message(cause)) }
    finally { setPending(false) }
  }

  return <div className="mx-auto max-w-xl space-y-5 p-6">
    <h1 className="text-2xl font-bold text-ink">Invite someone</h1>
    <p className="text-sm text-ink-muted">Create a registration invitation. Unit invitations require admin access to a verified unit.</p>
    <Card><CardHeader title="New invitation" /><CardBody>
      <form onSubmit={create} className="space-y-4">
        <Field label="Invitation type">{({ id, ...aria }) => <select id={id} {...aria} className="w-full rounded-lg border border-border-hi bg-surface p-3 text-ink" value={scope} onChange={e => { setScope(e.target.value as InviteScope); setCreated(null) }}><option value="platform">Platform registration</option><option value="unit">Unit invitation</option></select>}</Field>
        {scope === 'unit' && <Field label="Verified unit ID" required hint="Only a unit admin or head admin can invite to their verified unit.">{props => <Input {...props} required value={unitId} onChange={e => setUnitId(e.target.value)} />}</Field>}
        <Field label="Expires after (hours)" hint="Leave blank for no expiry.">{props => <Input {...props} type="number" min="1" step="1" value={expiresInHours} onChange={e => setExpiresInHours(e.target.value)} />}</Field>
        <Field label="Maximum uses">{props => <Input {...props} type="number" min="1" step="1" required value={maxUses} onChange={e => setMaxUses(e.target.value)} />}</Field>
        {error && <p role="alert" className="text-sm text-emergency">{error}</p>}
        <Button type="submit" variant="primary" loading={pending}>Create invitation</Button>
      </form>
    </CardBody></Card>
    {created && <Card><CardHeader title="Save this code now" /><CardBody className="space-y-3">
      <p role="status" className="text-sm text-ink-muted">{created.warning} It will disappear if you leave this page.</p>
      <code className="block break-all rounded-lg bg-surface-hi p-3 text-sm text-ink">{created.invite.code}</code>
      <Button onClick={() => { void navigator.clipboard.writeText(`${window.location.origin}/invite?code=${encodeURIComponent(created.invite.code)}`).catch(() => setError('Could not copy the link. Copy the code above instead.')) }}>Copy invitation link</Button>
    </CardBody></Card>}
  </div>
}

/** Validation reveals only safe invite metadata; signup does not redeem a unit code. */
export function InviteLandingPage() {
  const [params] = useSearchParams()
  const [code, setCode] = useState(params.get('code') ?? '')
  const [result, setResult] = useState<ValidatedInvite['invite'] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function validate(event: FormEvent) {
    event.preventDefault()
    setResult(null)
    setError(null)
    if (!code.trim()) { setError('Enter an invitation code.'); return }
    setPending(true)
    try { setResult((await api.postAnonymous<ValidatedInvite>('/invites/validate', { code: code.trim() })).invite) }
    catch (cause) { setError(message(cause)) }
    finally { setPending(false) }
  }

  return <div className="flex min-h-screen items-center justify-center bg-base p-6"><div className="w-full max-w-md space-y-5">
    <Link to="/" className="flex items-center gap-2 text-ink"><Shield className="size-6 text-signal" aria-hidden /> NATIVITY GUARD</Link>
    <h1 className="text-2xl font-semibold text-ink">Your invitation</h1>
    <form onSubmit={validate} className="space-y-3"><Field label="Invitation code" required>{props => <Input {...props} required value={code} onChange={e => { setCode(e.target.value); setResult(null) }} />}</Field><Button type="submit" variant="primary" loading={pending}>Check invitation</Button></form>
    {error && <p role="alert" className="text-sm text-emergency">{error}</p>}
    {result && <Card><CardBody className="space-y-4">
      <p role="status" className="text-ink">{result.scope === 'unit' ? `Invitation to ${result.unitName ?? 'a unit'}` : 'Invitation to Nativity Guard'}</p>
      <p className="text-sm text-ink-muted">You can create a citizen account or continue as a citizen. Registration does not automatically join a unit or redeem this invitation code.</p>
      <div className="flex flex-wrap gap-4 text-sm"><Link className="text-signal underline" to="/auth/signup">Create citizen account</Link><Link className="text-signal underline" to="/auth/login">Already registered? Sign in</Link></div>
    </CardBody></Card>}
  </div></div>
}

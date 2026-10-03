import { useState } from 'react'
import { Check, Copy, Plus, Trash2 } from 'lucide-react'
import { cn } from '@/lib/cn'
import { relativeTime } from '@/lib/format'
import { useToast } from '@/components/ui/Toast'
import { ApiError } from '@/lib/apiClient'
import { useInviteActions, useUnitInvites } from '@/hooks/useUnitInvites'

interface InviteManagerProps {
  unitId: string
}

/**
 * Create, list and revoke invite codes for a unit. The plaintext code is
 * shown exactly once, right after creation, and immediately copied as a
 * full signup link — after that, only the hash exists.
 */
export function InviteManager({ unitId }: InviteManagerProps) {
  const { notify } = useToast()
  const invites = useUnitInvites(unitId)
  const actions = useInviteActions(unitId)

  const [expiresInHours, setExpiresInHours] = useState('168')
  const [maxUses, setMaxUses] = useState('5')
  const [justCreated, setJustCreated] = useState<{ code: string; link: string } | null>(null)
  const [revokingId, setRevokingId] = useState<string | null>(null)

  function reportError(cause: unknown, fallback: string) {
    if (cause instanceof ApiError) notify(cause.message, 'error')
    else notify(fallback, 'error')
  }

  function create() {
    const hours = Number(expiresInHours) || 0
    const uses = Number(maxUses) || 1
    actions.create.mutate(
      {
        ...(hours > 0 ? { expiresInHours: hours } : {}),
        maxUses: uses,
      },
      {
        onSuccess: (res) => {
          const link = `${window.location.origin}/invite?code=${encodeURIComponent(res.invite.code)}`
          setJustCreated({ code: res.invite.code, link })
          notify('Invite created', 'success')
        },
        onError: (cause) => reportError(cause, 'Could not create invite'),
      },
    )
  }

  async function copyLink(link: string) {
    try {
      await navigator.clipboard.writeText(link)
      notify('Link copied', 'success')
    } catch {
      notify('Could not copy — copy by hand.', 'info')
    }
  }

  return (
    <div className="space-y-4">
      {/* Create new */}
      <div className="rounded-panel border border-border bg-surface/60 p-4">
        <h3 className="text-sm font-semibold text-ink">Create an invite link</h3>
        <p className="mt-0.5 text-xs text-ink-muted">
          Anyone with the link can register and apply to join this unit as an officer.
        </p>

        <div className="mt-3 grid grid-cols-2 gap-3">
          <label className="text-xs text-ink-muted">
            Expires in (hours)
            <input
              type="number"
              min={1}
              step={1}
              value={expiresInHours}
              onChange={(e) => setExpiresInHours(e.target.value)}
              className="mt-1 w-full rounded-lg border border-border bg-surface-hi px-3 py-2 text-sm text-ink"
            />
          </label>
          <label className="text-xs text-ink-muted">
            Max uses
            <input
              type="number"
              min={1}
              max={500}
              step={1}
              value={maxUses}
              onChange={(e) => setMaxUses(e.target.value)}
              className="mt-1 w-full rounded-lg border border-border bg-surface-hi px-3 py-2 text-sm text-ink"
            />
          </label>
        </div>

        <button
          type="button"
          onClick={create}
          disabled={actions.create.isPending}
          className="mt-3 inline-flex items-center gap-2 rounded-lg bg-signal px-4 py-2 text-sm font-semibold text-signal-ink transition-colors hover:bg-signal/90 disabled:opacity-50"
        >
          <Plus className="size-4" aria-hidden />
          {actions.create.isPending ? 'Creating…' : 'Create invite'}
        </button>
      </div>

      {/* Just-created banner */}
      {justCreated ? (
        <div className="rounded-panel border border-ok/40 bg-ok/5 p-4">
          <div className="flex items-start gap-2">
            <Check className="mt-0.5 size-4 shrink-0 text-ok" aria-hidden />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-ink">Save this link now</p>
              <p className="mt-0.5 text-xs text-ink-muted">
                It cannot be retrieved again — only the hash is stored.
              </p>
              <code className="mt-2 block break-all rounded-md bg-surface-hi p-2 text-xs text-ink">
                {justCreated.link}
              </code>
              <button
                type="button"
                onClick={() => void copyLink(justCreated.link)}
                className="mt-2 inline-flex items-center gap-1.5 rounded-md bg-signal px-3 py-1.5 text-xs font-medium text-signal-ink"
              >
                <Copy className="size-3.5" aria-hidden />
                Copy link
              </button>
            </div>
            <button
              type="button"
              aria-label="Dismiss"
              onClick={() => setJustCreated(null)}
              className="rounded-md p-1 text-ink-faint hover:bg-surface-hi"
            >
              ✕
            </button>
          </div>
        </div>
      ) : null}

      {/* List */}
      <div className="overflow-hidden rounded-panel border border-border bg-surface/60">
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <h3 className="text-sm font-semibold text-ink">Existing invites</h3>
          <span className="rounded-full bg-surface-hi px-2 py-0.5 text-[11px] text-ink-faint tabular-nums">
            {invites.data?.length ?? 0}
          </span>
        </div>
        {invites.isLoading ? (
          <div className="space-y-2 p-3">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-12 animate-pulse rounded bg-surface-hi/60" />
            ))}
          </div>
        ) : (invites.data ?? []).length === 0 ? (
          <p className="px-4 py-6 text-center text-xs text-ink-muted">No invites yet.</p>
        ) : (
          <ul className="divide-y divide-border">
            {invites.data!.map((inv) => {
              const revoking = revokingId === inv.id
              const effective =
                inv.status === 'pending'
                  ? inv.useCount >= inv.maxUses
                    ? 'exhausted'
                    : inv.expiresAt && new Date(inv.expiresAt) < new Date()
                      ? 'expired'
                      : 'active'
                  : inv.status
              return (
                <li key={inv.id} className="flex items-center gap-3 p-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={cn(
                          'rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider',
                          effective === 'active' && 'bg-ok/15 text-ok',
                          effective === 'expired' && 'bg-warn/15 text-warn',
                          effective === 'exhausted' && 'bg-warn/15 text-warn',
                          effective === 'revoked' && 'bg-emergency/15 text-emergency',
                        )}
                      >
                        {effective}
                      </span>
                      <span className="text-[11px] tabular-nums text-ink-faint">
                        {inv.useCount}/{inv.maxUses} used
                      </span>
                    </div>
                    <p className="mt-1 text-[11px] text-ink-faint">
                      Created {relativeTime(inv.createdAt)}
                      {inv.expiresAt ? ` · expires ${new Date(inv.expiresAt).toLocaleDateString()}` : ' · no expiry'}
                    </p>
                  </div>
                  {effective === 'active' || effective === 'exhausted' || effective === 'expired' ? (
                    <button
                      type="button"
                      onClick={() => {
                        setRevokingId(inv.id)
                        actions.revoke.mutate(inv.id, {
                          onSuccess: () => {
                            notify('Invite revoked', 'success')
                            setRevokingId(null)
                          },
                          onError: (cause) => {
                            reportError(cause, 'Could not revoke')
                            setRevokingId(null)
                          },
                        })
                      }}
                      disabled={revoking}
                      aria-label="Revoke invite"
                      className="grid size-8 place-items-center rounded-md text-emergency transition-colors hover:bg-emergency/10 disabled:opacity-40"
                    >
                      <Trash2 className="size-4" aria-hidden />
                    </button>
                  ) : null}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

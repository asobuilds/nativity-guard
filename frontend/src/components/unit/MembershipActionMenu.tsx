import { useEffect, useRef, useState } from 'react'
import { Check, Crown, MoreVertical, Shield, UserX } from 'lucide-react'
import { cn } from '@/lib/cn'
import type { RosterMember } from '@/hooks/useUnitRoster'

interface MembershipActionMenuProps {
  member: RosterMember
  currentUserId: string | undefined
  onApprove: () => void
  onReject: (reason: string) => void
  onPromote: () => void
  onRevoke: (reason: string) => void
  busy: boolean
}

type Mode = 'menu' | 'revoke' | 'reject'

/**
 * Compact dropdown for admin actions on one roster row. Confirm dialogs
 * for destructive actions ask for a reason, which the backend stores.
 * Cannot act on yourself or on the head admin.
 */
export function MembershipActionMenu({
  member,
  currentUserId,
  onApprove,
  onReject,
  onPromote,
  onRevoke,
  busy,
}: MembershipActionMenuProps) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<Mode>('menu')
  const [reason, setReason] = useState('')
  const wrapRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function onDoc(e: MouseEvent) {
      if (!open) return
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false)
        setMode('menu')
        setReason('')
      }
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  const isSelf = currentUserId === member.userId
  const isHead = member.isHeadAdmin
  const isPending = member.status === 'pending'
  const isActive = member.status === 'active'
  const canPromote = isActive && member.role === 'officer' && !isSelf && !isHead
  const canRevoke = isActive && !isSelf && !isHead

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        aria-label="Member actions"
        disabled={busy}
        onClick={() => setOpen((v) => !v)}
        className="grid size-8 place-items-center rounded-md text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink disabled:opacity-40"
      >
        <MoreVertical className="size-4" aria-hidden />
      </button>

      {open ? (
        <div className="absolute right-0 top-9 z-20 w-56 overflow-hidden rounded-lg border border-border bg-base shadow-lg">
          {mode === 'menu' ? (
            <div className="flex flex-col">
              {isPending ? (
                <>
                  <MenuItem
                    icon={<Check className="size-3.5 text-ok" aria-hidden />}
                    label="Approve application"
                    onClick={() => {
                      onApprove()
                      setOpen(false)
                    }}
                  />
                  <MenuItem
                    icon={<UserX className="size-3.5 text-emergency" aria-hidden />}
                    label="Reject application"
                    onClick={() => setMode('reject')}
                  />
                </>
              ) : null}

              {canPromote ? (
                <MenuItem
                  icon={<Shield className="size-3.5 text-signal" aria-hidden />}
                  label="Promote to admin"
                  onClick={() => {
                    onPromote()
                    setOpen(false)
                  }}
                />
              ) : null}

              {canRevoke ? (
                <MenuItem
                  icon={<UserX className="size-3.5 text-emergency" aria-hidden />}
                  label="Revoke membership"
                  onClick={() => setMode('revoke')}
                />
              ) : null}

              {isSelf ? (
                <p className="px-3 py-2 text-[11px] text-ink-faint">
                  You cannot act on your own membership.
                </p>
              ) : null}
              {isHead && !isSelf ? (
                <p className="px-3 py-2 text-[11px] text-ink-faint">
                  <Crown className="mr-1 inline size-3 text-warn" aria-hidden />
                  Head admin removal requires a revocation cycle.
                </p>
              ) : null}
            </div>
          ) : null}

          {mode === 'reject' ? (
            <div className="p-3">
              <p className="text-xs font-medium text-ink">Reject this application?</p>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Optional reason"
                rows={2}
                maxLength={500}
                className="mt-2 w-full rounded-md border border-border bg-surface-hi px-2 py-1 text-xs text-ink"
              />
              <div className="mt-2 flex justify-end gap-1.5">
                <button
                  type="button"
                  onClick={() => {
                    setMode('menu')
                    setReason('')
                  }}
                  className="rounded-md px-2 py-1 text-[11px] text-ink-muted hover:bg-surface-hi"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => {
                    onReject(reason.trim())
                    setOpen(false)
                    setMode('menu')
                    setReason('')
                  }}
                  className="rounded-md bg-emergency px-2 py-1 text-[11px] font-medium text-white disabled:opacity-50"
                >
                  Confirm
                </button>
              </div>
            </div>
          ) : null}

          {mode === 'revoke' ? (
            <div className="p-3">
              <p className="text-xs font-medium text-ink">Revoke membership?</p>
              <p className="mt-1 text-[11px] text-ink-faint">
                The member is removed from the unit. A reason is required.
              </p>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Reason"
                rows={3}
                maxLength={500}
                required
                className="mt-2 w-full rounded-md border border-border bg-surface-hi px-2 py-1 text-xs text-ink"
              />
              <div className="mt-2 flex justify-end gap-1.5">
                <button
                  type="button"
                  onClick={() => {
                    setMode('menu')
                    setReason('')
                  }}
                  className="rounded-md px-2 py-1 text-[11px] text-ink-muted hover:bg-surface-hi"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  disabled={busy || !reason.trim()}
                  onClick={() => {
                    onRevoke(reason.trim())
                    setOpen(false)
                    setMode('menu')
                    setReason('')
                  }}
                  className="rounded-md bg-emergency px-2 py-1 text-[11px] font-medium text-white disabled:opacity-50"
                >
                  Confirm
                </button>
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

function MenuItem({
  icon,
  label,
  onClick,
}: {
  icon: React.ReactNode
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 px-3 py-2 text-left text-xs text-ink transition-colors',
        'hover:bg-surface-hi',
      )}
    >
      {icon}
      {label}
    </button>
  )
}

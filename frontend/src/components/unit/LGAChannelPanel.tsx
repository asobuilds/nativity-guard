import { useState } from 'react'
import { AlertTriangle, Building2, Megaphone, Radio, Send } from 'lucide-react'
import { cn } from '@/lib/cn'
import { relativeTime } from '@/lib/format'
import { useToast } from '@/components/ui/Toast'
import { ApiError } from '@/lib/apiClient'
import {
  useLGAChannel,
  useLGAChannelActions,
  type LGAMessage,
} from '@/hooks/useLGAChannel'

const CATEGORY_META: Record<
  LGAMessage['category'],
  { label: string; tone: string; icon: React.ReactNode }
> = {
  alert: {
    label: 'Alert',
    tone: 'bg-emergency/15 text-emergency ring-emergency/30',
    icon: <AlertTriangle className="size-3.5" />,
  },
  intel: {
    label: 'Intel',
    tone: 'bg-signal/15 text-signal ring-signal/30',
    icon: <Radio className="size-3.5" />,
  },
  coordination: {
    label: 'Coordination',
    tone: 'bg-ok/15 text-ok ring-ok/30',
    icon: <Megaphone className="size-3.5" />,
  },
  request: {
    label: 'Request',
    tone: 'bg-warn/15 text-warn ring-warn/30',
    icon: <Send className="size-3.5" />,
  },
  general: {
    label: 'General',
    tone: 'bg-surface-hi text-ink-muted ring-border',
    icon: <Megaphone className="size-3.5" />,
  },
}

export function LGAChannelPanel() {
  const { notify } = useToast()
  const channel = useLGAChannel()
  const post = useLGAChannelActions()

  const [category, setCategory] = useState<LGAMessage['category']>('general')
  const [priority, setPriority] = useState<LGAMessage['priority']>('normal')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [expiresInHours, setExpiresInHours] = useState('336')

  function submit() {
    if (!title.trim() || !body.trim()) return
    const hours = Number(expiresInHours) || 0
    post.mutate(
      {
        category,
        priority,
        title: title.trim(),
        body: body.trim(),
        ...(hours > 0 ? { expiresInHours: hours } : {}),
      },
      {
        onSuccess: () => {
          notify('Posted to the channel', 'success')
          setTitle('')
          setBody('')
          setPriority('normal')
          setCategory('general')
        },
        onError: (cause) => {
          notify(
            cause instanceof ApiError ? cause.message : 'Could not post',
            'error',
          )
        },
      },
    )
  }

  if (channel.isLoading) {
    return <div className="h-40 animate-pulse rounded-panel bg-surface-hi/60" />
  }

  if (channel.isError || !channel.data) {
    return (
      <div className="rounded-lg border border-warn/30 bg-warn/5 p-4 text-sm text-warn">
        The channel is not available to your account right now. Verified unit members of
        the same LGA can read and post here.
      </div>
    )
  }

  const { channel: meta, messages } = channel.data

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-2 rounded-panel border border-border bg-surface/60 p-4">
        <Building2 className="size-5 text-signal" aria-hidden />
        <div>
          <p className="text-sm font-semibold text-ink">
            {meta.lga} · {meta.state}
          </p>
          <p className="text-xs text-ink-muted">
            Private channel for verified units in this LGA. Visible only to their members.
          </p>
        </div>
      </div>

      {/* Compose */}
      <div className="rounded-panel border border-border bg-surface/60 p-4">
        <h3 className="text-sm font-semibold text-ink">Post a notice</h3>

        <div className="mt-3 flex flex-wrap gap-1.5">
          {(Object.keys(CATEGORY_META) as LGAMessage['category'][]).map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => setCategory(c)}
              className={cn(
                'rounded-full px-3 py-1 text-xs font-medium ring-1 transition-colors',
                category === c
                  ? CATEGORY_META[c].tone
                  : 'bg-surface-hi text-ink-muted ring-border hover:text-ink',
              )}
            >
              {CATEGORY_META[c].label}
            </button>
          ))}
        </div>

        <div className="mt-3 grid grid-cols-2 gap-2">
          <button
            type="button"
            onClick={() => setPriority('normal')}
            className={cn(
              'rounded-lg border p-2 text-xs font-medium transition-colors',
              priority === 'normal'
                ? 'border-signal bg-signal/10 text-ink'
                : 'border-border text-ink-muted hover:bg-surface-hi',
            )}
          >
            Normal
          </button>
          <button
            type="button"
            onClick={() => setPriority('urgent')}
            className={cn(
              'rounded-lg border p-2 text-xs font-medium transition-colors',
              priority === 'urgent'
                ? 'border-emergency bg-emergency/10 text-emergency'
                : 'border-border text-ink-muted hover:bg-surface-hi',
            )}
          >
            Urgent
          </button>
        </div>

        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          maxLength={200}
          placeholder="Short title — e.g. 'Joint patrol on Bode Thomas tonight'"
          className="mt-3 w-full rounded-lg border border-border bg-surface-hi px-3 py-2 text-sm text-ink"
        />
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={4}
          maxLength={4000}
          placeholder="Full notice. Who, what, when, where, and what you need from other units."
          className="mt-2 w-full rounded-lg border border-border bg-surface-hi px-3 py-2 text-sm text-ink"
        />
        <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
          <label className="flex items-center gap-2 text-xs text-ink-muted">
            Expires in (hours)
            <input
              type="number"
              min={1}
              step={1}
              value={expiresInHours}
              onChange={(e) => setExpiresInHours(e.target.value)}
              className="w-20 rounded-md border border-border bg-surface-hi px-2 py-1 text-xs text-ink"
            />
          </label>
          <button
            type="button"
            onClick={submit}
            disabled={post.isPending || !title.trim() || !body.trim()}
            className="inline-flex items-center gap-1.5 rounded-lg bg-signal px-4 py-2 text-sm font-semibold text-signal-ink transition-colors hover:bg-signal/90 disabled:opacity-50"
          >
            <Send className="size-4" aria-hidden />
            {post.isPending ? 'Posting…' : 'Post'}
          </button>
        </div>
      </div>

      {/* Feed */}
      {messages.length === 0 ? (
        <p className="rounded-lg border border-dashed border-border px-4 py-10 text-center text-sm text-ink-muted">
          No notices yet. Anything posted here is visible to every verified unit in{' '}
          {meta.lga}.
        </p>
      ) : (
        <ul className="space-y-3">
          {messages.map((m) => {
            const catMeta = CATEGORY_META[m.category] ?? CATEGORY_META.general
            return (
              <li
                key={m.id}
                className={cn(
                  'rounded-panel border bg-surface/60 p-4',
                  m.priority === 'urgent'
                    ? 'border-emergency/40'
                    : 'border-border',
                )}
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span
                    className={cn(
                      'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider ring-1',
                      catMeta.tone,
                    )}
                  >
                    {catMeta.icon}
                    {catMeta.label}
                  </span>
                  {m.priority === 'urgent' ? (
                    <span className="rounded-full bg-emergency/15 px-2 py-0.5 text-[10px] font-semibold text-emergency">
                      Urgent
                    </span>
                  ) : null}
                  <span className="ml-auto text-[11px] text-ink-faint">
                    {relativeTime(m.createdAt)}
                  </span>
                </div>

                <p className="mt-2 text-sm font-semibold text-ink">{m.title}</p>
                <p className="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-ink-muted">
                  {m.body}
                </p>
                <p className="mt-3 text-[11px] text-ink-faint">
                  From {m.authorName} · {m.authorUnit}
                  {m.expiresAt ? ` · expires ${new Date(m.expiresAt).toLocaleDateString()}` : ''}
                </p>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

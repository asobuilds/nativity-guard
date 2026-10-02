import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Check, MapPin } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { useToast } from '@/components/ui/Toast'
import { api, ApiError } from '@/lib/apiClient'
import { usePreferences } from '@/hooks/usePreferences'
import { cn } from '@/lib/cn'
import type { Theme, TextSize } from '@/lib/preferences'

const THEMES: { value: Theme; label: string; preview: [string, string, string] }[] = [
  { value: 'forest', label: 'Forest', preview: ['#10221d', '#f4cb78', '#f8f5e9'] },
  { value: 'midnight', label: 'Midnight', preview: ['#0a0a0a', '#f4cb78', '#f5f5f5'] },
  { value: 'daylight', label: 'Daylight', preview: ['#f7f7f5', '#b8851a', '#1a1a1a'] },
  { value: 'ocean', label: 'Ocean', preview: ['#0a1a30', '#8bd3f7', '#eaf2fb'] },
  { value: 'stone', label: 'Stone', preview: ['#232019', '#e8b478', '#f4ede0'] },
]

const TEXT_SIZES: { value: TextSize; label: string; px: string }[] = [
  { value: 'standard', label: 'Standard', px: '100%' },
  { value: 'large', label: 'Large', px: '112%' },
  { value: 'xlarge', label: 'Extra large', px: '125%' },
  { value: 'xxlarge', label: 'Huge', px: '140%' },
]

export function SettingsPage() {
  const { role } = useAuth()
  const { notify } = useToast()
  const queryClient = useQueryClient()
  const { preferences, update, saving } = usePreferences()

  const locationSharing = useQuery({
    queryKey: ['location-sharing'],
    queryFn: () => api.get<{ enabled: boolean }>('/location/sharing'),
  })

  const toggleLocation = useMutation({
    mutationFn: (enabled: boolean) =>
      api.put<{ enabled: boolean }>('/location/sharing', { enabled }),
    onSuccess: () => {
      notify('Location sharing updated', 'success')
      void queryClient.invalidateQueries({ queryKey: ['location-sharing'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not update', 'error')
    },
  })

  return (
    <div className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <h1 className="text-2xl font-semibold text-ink">Your settings</h1>
        <p className="mt-2 text-sm text-ink-muted">
          These choices save to your account and follow you to any device.
        </p>
      </header>

      <Card className="space-y-4 p-5">
        <CardHeader title="Colour theme" />
        <CardBody className="pt-0">
          <p className="text-sm text-ink-muted">
            Pick the look that suits your screen and your eyes. Emergency red
            stays the same in every theme.
          </p>
          <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-5">
            {THEMES.map((t) => {
              const selected = preferences.theme === t.value
              return (
                <button
                  key={t.value}
                  type="button"
                  onClick={() => update({ theme: t.value })}
                  aria-pressed={selected}
                  className={cn(
                    'group flex flex-col items-center gap-2 rounded-lg border p-3 text-center transition-colors',
                    selected
                      ? 'border-signal bg-signal/10'
                      : 'border-border bg-surface-hi/30 hover:bg-surface-hi/60',
                  )}
                >
                  <span className="flex overflow-hidden rounded-md">
                    {t.preview.map((color, i) => (
                      <span
                        key={i}
                        className="block size-7"
                        style={{ backgroundColor: color }}
                      />
                    ))}
                  </span>
                  <span className="flex items-center gap-1 text-xs font-medium text-ink">
                    {selected ? <Check className="size-3 text-signal" /> : null}
                    {t.label}
                  </span>
                </button>
              )
            })}
          </div>
        </CardBody>
      </Card>

      <Card className="space-y-4 p-5">
        <CardHeader title="Text size" />
        <CardBody className="pt-0">
          <p className="text-sm text-ink-muted">
            Make everything bigger if small text is hard to read.
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            {TEXT_SIZES.map((s) => {
              const selected = preferences.textSize === s.value
              return (
                <button
                  key={s.value}
                  type="button"
                  onClick={() => update({ textSize: s.value })}
                  aria-pressed={selected}
                  className={cn(
                    'flex items-center gap-2 rounded-lg border px-4 py-2 text-sm transition-colors',
                    selected
                      ? 'border-signal bg-signal/10 text-ink'
                      : 'border-border bg-surface-hi/30 text-ink-muted hover:bg-surface-hi/60',
                  )}
                >
                  {selected ? <Check className="size-3.5 text-signal" /> : null}
                  <span>{s.label}</span>
                  <span className="text-xs text-ink-faint">({s.px})</span>
                </button>
              )
            })}
          </div>
        </CardBody>
      </Card>

      <Card className="space-y-4 p-5">
        <CardHeader title="Location sharing" />
        <CardBody className="pt-0">
          <p className="text-sm text-ink-muted">
            When on, your current location is sent to the platform so units near
            you can respond faster and alerts within your radius reach you.
          </p>
          <div className="mt-4 flex items-center gap-3">
            <button
              type="button"
              role="switch"
              aria-checked={locationSharing.data?.enabled === true}
              onClick={() =>
                toggleLocation.mutate(!(locationSharing.data?.enabled === true))
              }
              disabled={locationSharing.isLoading || toggleLocation.isPending}
              className={cn(
                'relative inline-flex h-7 w-12 shrink-0 items-center rounded-full border transition-colors',
                locationSharing.data?.enabled
                  ? 'border-signal bg-signal'
                  : 'border-border bg-surface-hi',
                (locationSharing.isLoading || toggleLocation.isPending) && 'opacity-60',
              )}
            >
              <span
                className={cn(
                  'block size-5 rounded-full bg-white shadow transition-transform',
                  locationSharing.data?.enabled ? 'translate-x-6' : 'translate-x-0.5',
                )}
              />
            </button>
            <div className="text-sm text-ink">
              {locationSharing.data?.enabled ? (
                <span className="flex items-center gap-1.5">
                  <MapPin className="size-3.5 text-signal" aria-hidden />
                  Sharing your location
                </span>
              ) : (
                'Not sharing your location'
              )}
            </div>
          </div>
        </CardBody>
      </Card>

      <Card className="space-y-4 p-5">
        <CardHeader title="Notifications" />
        <CardBody className="space-y-3 pt-0">
          <p className="text-sm text-ink-muted">
            Choose which notifications reach you inside the app.
          </p>
          <Toggle
            label="Emergency alerts"
            description="Alerts from your subscribed categories, filtered by your radius."
            checked={preferences.notifyAlerts}
            onChange={(v) => update({ notifyAlerts: v })}
            disabled={saving}
          />
          <Toggle
            label="Case updates"
            description="When a case you reported changes status or gets a reply."
            checked={preferences.notifyCaseUpdates}
            onChange={(v) => update({ notifyCaseUpdates: v })}
            disabled={saving}
          />
          <Toggle
            label="Community replies"
            description="When someone replies to a post you wrote."
            checked={preferences.notifyCommunityReplies}
            onChange={(v) => update({ notifyCommunityReplies: v })}
            disabled={saving}
          />
        </CardBody>
      </Card>

      <Card className="space-y-3 p-5">
        <CardHeader title="Account and safety" />
        <CardBody className="flex flex-col gap-2 pt-0 text-sm">
          <Link to="/profile" className="text-signal hover:underline">Edit your profile</Link>
          <Link to="/verify-identity" className="text-signal hover:underline">Verify your identity</Link>
          <Link to="/sessions" className="text-signal hover:underline">Manage signed-in devices</Link>
          <Link to="/notifications" className="text-signal hover:underline">View notifications</Link>
          {role === 'citizen' ? (
            <Link to="/subscriptions" className="text-signal hover:underline">Manage alert subscriptions</Link>
          ) : null}
          {role === 'unit_admin' ? (
            <Link to="/admin/settings" className="text-signal hover:underline">Unit settings</Link>
          ) : null}
          {role === 'super_admin' ? (
            <Link to="/super/settings" className="text-signal hover:underline">Platform settings</Link>
          ) : null}
        </CardBody>
      </Card>

      <Card className="space-y-2 p-5">
        <CardHeader title="Rules and privacy" />
        <CardBody className="pt-0">
          <p className="text-sm text-ink-muted">
            Read the rules for your account type and how to use reports, evidence,
            and community features responsibly.
          </p>
          <Link to="/terms" className="mt-2 inline-block text-sm text-signal hover:underline">
            Read user terms and role guidance
          </Link>
        </CardBody>
      </Card>
    </div>
  )
}

function Toggle({
  label,
  description,
  checked,
  onChange,
  disabled,
}: {
  label: string
  description?: string
  checked: boolean
  onChange: (next: boolean) => void
  disabled?: boolean
}) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-lg border border-border bg-surface-hi/30 p-3">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-ink">{label}</p>
        {description ? <p className="mt-0.5 text-xs text-ink-muted">{description}</p> : null}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        disabled={disabled}
        className={cn(
          'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors',
          checked ? 'border-signal bg-signal' : 'border-border bg-surface-hi',
          disabled && 'opacity-60',
        )}
      >
        <span
          className={cn(
            'block size-4 rounded-full bg-white shadow transition-transform',
            checked ? 'translate-x-5' : 'translate-x-0.5',
          )}
        />
      </button>
    </div>
  )
}
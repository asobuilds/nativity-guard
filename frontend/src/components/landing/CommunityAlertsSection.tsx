import { useMemo, useState } from 'react'
import {
  Bell,
  Flame,
  Info,
  MapPin,
  Megaphone,
  Radio,
  TriangleAlert,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { relativeTime } from '@/lib/format'
import type { CommunityAlert } from '@/types/community'
import type { ActionContext } from './types'

export interface CommunityAlertsSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

// No public alerts endpoint is wired into this feature yet.
// Do not publish fictional warnings as live community safety notices.
const PUBLIC_ALERTS: CommunityAlert[] = []

export function CommunityAlertsSection({ onActionTrigger }: CommunityAlertsSectionProps) {
  const [severityFilter, setSeverityFilter] = useState<'all' | 'urgent' | 'caution' | 'info'>('all')

  const alerts = useMemo(() => {
    if (severityFilter === 'all') return PUBLIC_ALERTS
    return PUBLIC_ALERTS.filter((a) => a.severity === severityFilter)
  }, [severityFilter])

  function handleSubscribe() {
    onActionTrigger({
      type: 'subscribe_alerts',
      title: 'Subscribe to Local Emergency Broadcasts',
      description:
        'Explore available notification settings. Receiving alerts depends on the service and your location.',
      badgeText: 'Availability varies',
      returnUrl: '/subscriptions',
    })
  }

  function handleCreateAlert() {
    onActionTrigger({
      type: 'community_post',
      title: 'Broadcast Community Safety Advisory',
      description:
        'Community security alerts must be verified by local unit command to prevent alarmism.',
      badgeText: 'Sign-in required',
      returnUrl: '/alerts',
    })
  }

  return (
    <section id="community-alerts" className="border-t border-border py-14">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
            <Radio className="size-3.5 text-signal" aria-hidden />
            Community Alerts
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Public Community Advisories
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            This panel does not currently provide a verified live alert feed.
            For urgent assistance in Nigeria, call 112.
          </p>
        </div>

        <div className="flex items-center gap-2 pt-2 md:pt-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={handleCreateAlert}
            icon={<Megaphone className="size-3.5 text-signal" />}
          >
            Post Advisory
          </Button>
          <Button
            variant="primary"
            size="sm"
            onClick={handleSubscribe}
            icon={<Bell className="size-3.5" />}
          >
            Subscribe to SMS Alerts
          </Button>
        </div>
      </div>

      {/* Severity Filter Pills */}
      <div className="mt-6 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setSeverityFilter('all')}
          className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
            severityFilter === 'all'
              ? 'bg-signal text-signal-ink font-semibold'
              : 'bg-surface-hi text-ink-muted hover:text-ink'
          }`}
        >
          All Notices ({PUBLIC_ALERTS.length})
        </button>

        <button
          type="button"
          onClick={() => setSeverityFilter('urgent')}
          className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
            severityFilter === 'urgent'
              ? 'bg-emergency text-white font-semibold'
              : 'bg-surface-hi text-emergency hover:bg-emergency/20'
          }`}
        >
          <Flame className="size-3" />
          Urgent Notices
        </button>

        <button
          type="button"
          onClick={() => setSeverityFilter('caution')}
          className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
            severityFilter === 'caution'
              ? 'bg-warn text-signal-ink font-semibold'
              : 'bg-surface-hi text-warn hover:bg-warn/20'
          }`}
        >
          <TriangleAlert className="size-3" />
          Cautionary
        </button>

        <button
          type="button"
          onClick={() => setSeverityFilter('info')}
          className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
            severityFilter === 'info'
              ? 'bg-surface-hi text-ink font-semibold ring-1 ring-border-hi'
              : 'bg-surface-hi text-ink-muted hover:text-ink'
          }`}
        >
          <Info className="size-3 text-signal" />
          General Announcements
        </button>
      </div>

      {PUBLIC_ALERTS.length === 0 ? (
        <p role="status" className="mt-6 rounded-panel border border-border p-5 text-sm text-ink-muted">
          There are no verified public advisories available in this panel.
          Demonstration notices are intentionally not shown as real alerts.
        </p>
      ) : null}

      {/* Alert Feed Cards */}
      <div className="mt-6 grid gap-4 sm:grid-cols-2">
        {alerts.map((alert) => {
          const isUrgent = alert.severity === 'urgent'
          const isCaution = alert.severity === 'caution'
          return (
            <Card
              key={alert.id}
              className={`p-5 flex flex-col justify-between border ${
                isUrgent
                  ? 'border-emergency/30 bg-emergency/5'
                  : isCaution
                  ? 'border-warn/30 bg-warn/5'
                  : 'border-border bg-surface'
              }`}
            >
              <div>
                <div className="flex items-center justify-between gap-2">
                  <span
                    className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide ${
                      isUrgent
                        ? 'bg-emergency/20 text-emergency ring-1 ring-emergency/40'
                        : isCaution
                        ? 'bg-warn/20 text-warn ring-1 ring-warn/40'
                        : 'bg-surface-hi text-ink-muted ring-1 ring-border-hi'
                    }`}
                  >
                    {isUrgent ? (
                      <Flame className="size-3" />
                    ) : isCaution ? (
                      <TriangleAlert className="size-3" />
                    ) : (
                      <Info className="size-3" />
                    )}
                    {alert.severity}
                  </span>

                  <span className="text-[11px] text-ink-faint">
                    {relativeTime(alert.createdAt)}
                  </span>
                </div>

                <h3 className="mt-3 text-sm font-semibold text-ink leading-snug">
                  {alert.title}
                </h3>
                <p className="mt-1.5 text-xs leading-relaxed text-ink-muted">
                  {alert.description}
                </p>
              </div>

              <div className="mt-4 border-t border-border pt-3 flex items-center justify-between text-xs text-ink-faint">
                <span className="flex items-center gap-1">
                  <MapPin className="size-3 text-signal" aria-hidden />
                  {alert.location}
                </span>

                <button
                  type="button"
                  onClick={handleSubscribe}
                  className="text-signal hover:underline text-[11px] font-medium"
                >
                  Receive phone alerts →
                </button>
              </div>
            </Card>
          )
        })}
      </div>
    </section>
  )
}

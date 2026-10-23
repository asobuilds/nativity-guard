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
import type { CommunityAlert } from '@/types/community'
import type { ActionContext } from './types'

export interface CommunityAlertsSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

/** Illustrations, never presented as real incidents or active warnings.
 * Replace with audited backend data before claiming live alert coverage. */
const SAMPLE_COMMUNITY_ALERTS: CommunityAlert[] = [
  {
    id: 'demo-weather',
    title: 'Example: Severe weather advisory',
    description: 'A verified warning would explain the affected area, source, and recommended precautions.',
    severity: 'urgent',
    location: 'Example location only',
    createdAt: '2026-01-01T00:00:00Z',
    confirmations: 0,
  },
  {
    id: 'demo-road',
    title: 'Example: Local road disruption',
    description: 'A verified community notice could outline an area closure and where to find official updates.',
    severity: 'caution',
    location: 'Example location only',
    createdAt: '2026-01-01T00:00:00Z',
    confirmations: 0,
  },
  {
    id: 'demo-meeting',
    title: 'Example: Community safety meeting',
    description: 'An approved announcement could state the meeting location, organizer, and how to participate.',
    severity: 'info',
    location: 'Example location only',
    createdAt: '2026-01-01T00:00:00Z',
    confirmations: 0,
  },
]
export function CommunityAlertsSection({ onActionTrigger }: CommunityAlertsSectionProps) {
  const [severityFilter, setSeverityFilter] = useState<'all' | 'urgent' | 'caution' | 'info'>('all')

  const alerts = useMemo(() => {
    if (severityFilter === 'all') return SAMPLE_COMMUNITY_ALERTS
    return SAMPLE_COMMUNITY_ALERTS.filter((a) => a.severity === severityFilter)
  }, [severityFilter])

  function handleSubscribe() {
    onActionTrigger({
      type: 'subscribe_alerts',
      title: 'Subscribe to Local Emergency Broadcasts',
      description:
        'Sign in to manage alert notifications, where available in your community.',
      badgeText: 'Notification preferences',
      returnUrl: '/subscriptions',
    })
  }

  function handleCreateAlert() {
    onActionTrigger({
      type: 'community_post',
      title: 'Broadcast Community Safety Advisory',
      description:
        'Community security alerts must be verified by local unit command to prevent alarmism.',
      badgeText: 'Unit review required',
      returnUrl: '/alerts',
    })
  }

  return (
    <section id="community-alerts" className="border-t border-border py-14">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
            <Radio className="size-3.5 text-signal" aria-hidden />
            Alert Preview — Examples Only
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            How Community Advisories May Appear
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            These sample cards demonstrate alert categories. They are not live notices, verified warnings, or reports of real events. Never rely on these examples for safety decisions.
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
          All Examples ({SAMPLE_COMMUNITY_ALERTS.length})
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
          Urgent Examples
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

                  <span className="text-[11px] text-ink-faint">Demonstration only</span>
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
                  Notification settings →
                </button>
              </div>
            </Card>
          )
        })}
      </div>
    </section>
  )
}

import { AlertTriangle, RotateCcw } from 'lucide-react'
import { usePublicServiceAvailability } from '@/hooks/usePublicServiceAvailability'

/**
 * This warning appears only after a failed backend health check, not while
 * status is unknown or the request is still loading. No location, login,
 * incident data or personal details are sent by this health probe.
 */
export function LandingServiceNotice() {
  const { data: availability, refetch, isFetching } = usePublicServiceAvailability()
  if (availability !== 'unavailable') return null

  return (
    <aside className="ng-ref-service-notice" role="alert" aria-label="Temporary service interruption">
      <AlertTriangle size={20} className="ng-ref-service-notice-icon" aria-hidden />
      <div className="ng-ref-service-notice-copy">
        <strong>Live services may be unavailable</strong>
        <p>
          Nativity Guard cannot currently confirm that its backend is reachable.
          Sign-in, incident reports, case tracking and in-app SOS may not work.
          Do not depend on in-app SOS for urgent assistance in Nigeria; call{' '}
          <a href="tel:112">112</a>.
        </p>
      </div>
      <button
        type="button"
        className="ng-ref-service-retry"
        onClick={() => { void refetch() }}
        disabled={isFetching}
        aria-label="Check service availability again"
      >
        <RotateCcw size={15} aria-hidden />
        {isFetching ? 'Checking…' : 'Check again'}
      </button>
    </aside>
  )
}

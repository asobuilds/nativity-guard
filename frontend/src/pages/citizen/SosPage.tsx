import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, CheckCircle2, LocateFixed, ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { cn } from '@/lib/cn'
import { Card } from '@/components/ui/Card'
import { Modal } from '@/components/ui/Modal'
import { Input, Select, Textarea } from '@/components/ui/Field'
import { ErrorState, Skeleton } from '@/components/ui/States'
import { useMySos, useSendSos } from '@/hooks/useSos'
import { useUnits } from '@/hooks/useUnits'
import { useLocation, ipFallback } from '@/hooks/useLocation'
import { useReverseGeocode, formatAddress } from '@/hooks/useReverseGeocode'
import { MapView } from '@/components/map/MapView'
import { ApiError } from '@/lib/apiClient'
import { formatDateTime } from '@/lib/format'

const statusLabel = {
  pending: 'Pending',
  dispatched: 'Dispatched',
  resolved: 'Resolved',
  escalated: 'Escalated',
}

/** An explicit confirmation is required before any emergency request leaves the device. */
export function SosPage() {
  const [confirming, setConfirming] = useState(false)
  const confirmRef = useRef<HTMLDivElement>(null)
  const [locating, setLocating] = useState(false)
  const [locationError, setLocationError] = useState('')
  const [lat, setLat] = useState('')
  const [lng, setLng] = useState('')
  const [contacts, setContacts] = useState('')
  const [medical, setMedical] = useState('')
  const [priority, setPriority] = useState<'high' | 'critical'>('high')
  const [unitId, setUnitId] = useState('')
  const [hideLocation, setHideLocation] = useState(false)
  const [receipt, setReceipt] = useState<{ trackingId: string; status: string } | null>(null)
  const alerts = useMySos()
  const units = useUnits()
  const send = useSendSos()

  const { latitude: userLat, longitude: userLng, permission } = useLocation()
  const { data: geo, isLoading: geoLoading } = useReverseGeocode(userLat, userLng)
  const [ipLocation, setIpLocation] = useState<{ latitude: number; longitude: number } | null>(null)
  const [showEnableModal, setShowEnableModal] = useState(false)
  const ipAttemptedRef = useRef(false)

  // Auto-fill SOS coordinates from the device fix the moment they arrive.
  useEffect(() => {
    if (userLat == null || userLng == null) return
    setLat((prev) => (prev.trim() === '' ? String(userLat) : prev))
    setLng((prev) => (prev.trim() === '' ? String(userLng) : prev))
  }, [userLat, userLng])

  // When the browser has remembered a prior "denied", GPS is unavailable.
  // As a last resort, try an IP-based fix once and pre-fill the fields —
  // but never block the send button on it.
  useEffect(() => {
    if (permission === 'denied' && !ipAttemptedRef.current) {
      ipAttemptedRef.current = true
      void ipFallback().then((coords) => {
        if (coords) {
          setIpLocation(coords)
          setLat((prev) => (prev.trim() === '' ? String(coords.latitude) : prev))
          setLng((prev) => (prev.trim() === '' ? String(coords.longitude) : prev))
        }
      })
    }
  }, [permission])

  const latitude = Number(lat)
  const longitude = Number(lng)
  const hasLocation = lat.trim() !== '' && lng.trim() !== '' && Number.isFinite(latitude) &&
    Number.isFinite(longitude) && Math.abs(latitude) <= 90 && Math.abs(longitude) <= 180 &&
    !(latitude === 0 && longitude === 0)

  useEffect(() => {
    if (confirming) confirmRef.current?.focus()
  }, [confirming])

  function locate() {
    if (!navigator.geolocation) {
      setLocationError('Location is unavailable on this device. Enter coordinates below.')
      return
    }
    setLocating(true)
    setLocationError('')
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setLat(String(coords.latitude))
        setLng(String(coords.longitude))
        setLocating(false)
      },
      () => {
        setLocationError('Location could not be obtained. Enter coordinates below.')
        setLocating(false)
      },
      { enableHighAccuracy: true, timeout: 10_000 },
    )
  }

  async function confirmSend() {
    if (!hasLocation || send.isPending) return
    try {
      const resp = await send.mutateAsync({
        latitude,
        longitude,
        priority,
        ...(unitId ? { unitId } : {}),
        ...(contacts.trim() ? { emergencyContacts: contacts.split(",").map(function(s){ return s.trim(); }).filter(Boolean) } : {}),
        ...(medical.trim() ? { medicalInfo: medical.trim() } : {}),
        // SOS keeps the precise coordinates internally for responders; this only
        // strips the reporter's identity from public feeds.
        ...(hideLocation ? { hideLocation: true } : {}),
      })
      setReceipt({ trackingId: resp.sos.id, status: resp.sos.status })
      setConfirming(false)
      setContacts('')
      setMedical('')
    } catch (err) {
      console.error('[SOS] send failed:', err)
      // Keep the confirmation and every field available for an explicit retry.
    }
  }

  function onDialogKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === 'Escape' && !send.isPending) setConfirming(false)
    if (event.key !== 'Tab') return
    const buttons = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button:not(:disabled)'))
    const first = buttons[0]
    const last = buttons[buttons.length - 1]
    if (!first || !last) return
    if (event.shiftKey && (document.activeElement === first || document.activeElement === event.currentTarget)) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }

  return (
    <div className="mx-auto w-full max-w-3xl space-y-5 p-4 pb-28 sm:p-6">
      {/* Hero band — the emergency action lives here and nowhere else. */}
      <div className="relative overflow-hidden rounded-panel glass-panel p-5 sm:p-6">
        <div
          aria-hidden
          className="pointer-events-none absolute -right-16 -top-16 size-56 rounded-full"
          style={{ background: 'radial-gradient(circle, rgba(239,68,68,0.25) 0%, transparent 70%)' }}
        />
        <div className="relative flex items-start gap-4">
          <span className="grid size-12 shrink-0 place-items-center rounded-full border border-emergency/40 bg-emergency/15 text-emergency">
            <ShieldAlert className="size-6" aria-hidden />
          </span>
          <div className="min-w-0">
            <h1 className="text-xl font-semibold text-ink sm:text-2xl">Emergency SOS</h1>
            <p className="mt-1.5 max-w-prose text-sm text-ink-muted">
              Prepare your location, then confirm before sending. Nothing is sent when you open this page.
            </p>
          </div>
        </div>
      </div>

      

      {receipt ? (
        <div className="relative overflow-hidden rounded-panel glass-panel p-5">
          <span role="status" className="sr-only">Emergency request received</span>
          <div
            aria-hidden
            className="pointer-events-none absolute -left-12 -top-12 size-48 rounded-full"
            style={{ background: 'radial-gradient(circle, rgba(34,197,94,0.22) 0%, transparent 70%)' }}
          />
          <div className="relative flex items-start gap-3">
            <span className="grid size-10 shrink-0 place-items-center rounded-full border border-ok/40 bg-ok/15 text-ok">
              <CheckCircle2 className="size-5" aria-hidden />
            </span>
            <div className="min-w-0 flex-1">
              <p className="font-semibold text-ink">Emergency request received</p>
              <p className="mt-1 text-xs text-ink-muted">
                A receipt does not mean a unit has been dispatched — check the status below.
              </p>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <span className="font-mono text-sm font-semibold tracking-wider text-ink tabular-nums">
                  {receipt.trackingId}
                </span>
                <Link
                  to={`/sos/${receipt.trackingId}`}
                  className="text-xs text-signal underline-offset-2 hover:underline"
                >
                  View status →
                </Link>
              </div>
            </div>
          </div>
        </div>
      ) : null}

      <Card className="space-y-5 p-5">
        <div>
          <p className="text-sm font-medium text-ink">Where should help go?</p>
          <Button className="mt-3" loading={locating} icon={<LocateFixed className="size-4" />} onClick={locate}>
            Use my location
          </Button>
           {locationError ? <p className="mt-2 text-sm text-warn" role="alert">{locationError}</p> : null}
           {permission === 'denied' && (
             <p className="mt-2 text-xs text-warn">
               Using approximate location. Precise GPS is off.{' '}
               <button type="button" className="underline" onClick={() => setShowEnableModal(true)}>
                 Enable location
               </button>
             </p>
           )}
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="text-xs text-ink-muted">Latitude
              <Input type="number" step="any" value={lat} onChange={(e) => setLat(e.target.value)} placeholder="6.5244" />
            </label>
            <label className="text-xs text-ink-muted">Longitude
              <Input type="number" step="any" value={lng} onChange={(e) => setLng(e.target.value)} placeholder="3.3792" />
            </label>
          </div>
          <MapView
            mode="pick"
            height="35vh"
            className="mt-3"
            label="Choose the SOS location"
            pickLocation={hasLocation ? [latitude, longitude] : null}
            userLocation={userLat != null && userLng != null ? { latitude: userLat, longitude: userLng } : (ipLocation ? { latitude: ipLocation.latitude, longitude: ipLocation.longitude } : null)}
            onPickLocation={(nextLat, nextLng) => { setLat(String(nextLat)); setLng(String(nextLng)) }}
          />
          {!hasLocation ? <p className="mt-2 text-xs text-ink-muted">Valid coordinates are required; 0,0 is not a usable location.</p> : null}
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-xs text-ink-muted">Urgency
            <Select value={priority} onChange={(e) => setPriority(e.target.value as 'high' | 'critical')}>
              <option value="high">High</option><option value="critical">Critical</option>
            </Select>
          </label>
          <label className="text-xs text-ink-muted">Preferred responding unit (optional)
            <Select value={unitId} onChange={(e) => setUnitId(e.target.value)}>
              <option value="">Let the service choose</option>
              {(units.data ?? []).map((unit) => <option key={unit.id} value={unit.id}>{unit.name}</option>)}
            </Select>
          </label>
        </div>
        <p className="text-xs text-ink-muted">A unit preference does not confirm dispatch.</p>
        <label className="block text-xs text-ink-muted">Emergency contact details (optional)
          <Input value={contacts} maxLength={250} onChange={(e) => setContacts(e.target.value)} placeholder="Name and phone number" />
        </label>
        <label className="block text-xs text-ink-muted">Medical information to share with responders (optional)
          <Textarea value={medical} maxLength={500} onChange={(e) => setMedical(e.target.value)} placeholder="Only what responders need to know" />
        </label>

        {/* hideLocation: SOS keeps precise coords internally for responders; this
            only hides the reporter's identity from public feeds (POST /sos). */}
        <label
          htmlFor="sos-hide-location"
          className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${hideLocation ? 'border-warn/40 bg-warn/10' : 'border-border-hi bg-surface-hi'}`}
        >
          <input
            id="sos-hide-location"
            type="checkbox"
            checked={hideLocation}
            onChange={(event) => setHideLocation(event.target.checked)}
            className="mt-0.5 size-4 shrink-0 accent-warn"
          />
          <span>
            <span className="block text-sm font-medium text-ink">
              Hide my identity from the public feed (responders still see you)
            </span>
            <span className="mt-0.5 block text-xs text-ink-muted">
              Your location is still sent to responding units; only public displays mask who reported it.
            </span>
          </span>
        </label>

        <p className="text-xs text-ink-muted">The optional details are sent only when you confirm; they are cleared from this form after success.</p>
        <Button variant="danger" size="lg" disabled={!hasLocation} onClick={() => setConfirming(true)} icon={<AlertTriangle className="size-5" />}>
          Prepare SOS
        </Button>

        {(userLat != null && userLng != null) && (
          <p className="text-sm text-ink-muted" aria-live="polite">
            {geoLoading ? (
              'Locating address…'
            ) : geo ? (
              <span>Your location: {formatAddress(geo)}</span>
            ) : (
              'Using GPS coordinates'
            )}
          </p>
        )}
      </Card>

      <section aria-label="Your SOS history">
        <h2 className="mb-3 text-lg font-semibold text-ink">Your SOS history</h2>
        {alerts.isLoading ? <Skeleton className="h-24 w-full" /> : alerts.isError ? (
          <Card><ErrorState title="Could not load SOS history" description="Reconnect and try again." onRetry={() => void alerts.refetch()} /></Card>
        ) : !alerts.data?.length ? (
          <Card className="p-5 text-sm text-ink-muted">No SOS requests on record.</Card>
        ) : (
          <ul className="space-y-2">
            {alerts.data.map((alert) => (
              <li key={alert.id}>
                <Link
                  to={`/sos/${alert.id}`}
                  className="block rounded-panel focus-visible:outline-2 focus-visible:outline-signal"
                >
                  <Card className="glass-panel--lift flex flex-wrap items-center justify-between gap-3 p-4">
                    <div className="min-w-0">
                      <p className="font-mono text-xs font-medium text-ink tabular-nums">
                        {alert.id.slice(0, 8)}…
                      </p>
                      <p className="mt-1 text-xs text-ink-muted">
                        {formatDateTime(alert.createdAt)}
                      </p>
                    </div>
                    <span
                      className={cn(
                        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium',
                        alert.status === 'resolved' && 'bg-ok/15 text-ok',
                        alert.status === 'escalated' && 'bg-warn/15 text-warn',
                        alert.status === 'dispatched' && 'bg-signal/15 text-signal',
                        alert.status === 'pending' && 'bg-surface-hi text-ink-muted',
                      )}
                    >
                      <span
                        aria-hidden
                        className={cn(
                          'size-1.5 rounded-full',
                          alert.status === 'resolved' && 'bg-ok',
                          alert.status === 'escalated' && 'bg-warn',
                          alert.status === 'dispatched' && 'bg-signal',
                          alert.status === 'pending' && 'bg-ink-faint',
                        )}
                      />
                      {statusLabel[alert.status] ?? alert.status}
                    </span>
                  </Card>
                </Link>
              </li>
            ))}
          </ul>
        )}
        <p className="mt-3 text-xs text-ink-muted">Statuses refresh while this page is open. <Link to="/" className="text-signal">Back to reports</Link></p>
      </section>

      {confirming ? (
        <div className="fixed inset-0 z-50 grid place-items-center bg-void/90 p-4" role="presentation">
          <div ref={confirmRef} tabIndex={-1} onKeyDown={onDialogKeyDown} role="alertdialog" aria-modal="true" aria-labelledby="sos-confirm-title" aria-describedby="sos-confirm-description" className="w-full max-w-md rounded-panel border border-emergency bg-surface p-5 shadow-panel">
            <h2 id="sos-confirm-title" className="text-lg font-semibold text-ink">Send emergency SOS?</h2>
            <p id="sos-confirm-description" className="mt-2 text-sm text-ink-muted">Your location ({latitude.toFixed(5)}, {longitude.toFixed(5)}) and any details above will be sent. Confirm only if you need emergency help.</p>
            {send.isError ? <p className="mt-3 text-sm text-warn" role="alert">{ApiError.isNetwork(send.error) ? 'Delivery could not be confirmed. Check your SOS history before retrying to avoid sending twice.' : 'Request was not confirmed. Check SOS history before retrying.'}</p> : null}
            <div className="mt-5 flex flex-wrap gap-2">
              <Button variant="danger" loading={send.isPending} onClick={() => void confirmSend()}>Confirm and send SOS</Button>
              <Button disabled={send.isPending} onClick={() => setConfirming(false)}>Cancel</Button>
            </div>
          </div>
        </div>
      ) : null}

      <Modal
        open={showEnableModal}
        onClose={() => setShowEnableModal(false)}
        title="Enable precise location"
        description="Follow the steps below to turn location access back on for this site."
      >
        <ul className="list-decimal space-y-1.5 text-sm text-ink-muted">
          <li>Chrome / Edge / Brave: tap the lock icon in the address bar → Site settings → Location → Allow.</li>
          <li>Firefox: tap the "i" (info) icon in the address bar → Permissions → Location → Allow.</li>
          <li>Safari (iOS/macOS): open the browser Settings app → Safari → Websites → Location → find this site → Allow.</li>
        </ul>
        <p className="mt-3 text-xs text-ink-muted">After enabling, refresh the page or tap "Use my location" to re-check.</p>
      </Modal>
    </div>
  )
}

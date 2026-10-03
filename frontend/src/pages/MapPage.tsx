import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Layers, MapPin, Navigation, Search, X } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Modal } from '@/components/ui/Modal'
import { PriorityChip, StatusChip } from '@/components/ui/Chips'
import { Input } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { MapView } from '@/components/map/MapView'
import { POIControlPanel } from '@/components/map/POIControlPanel'
import { useCases } from '@/hooks/useCases'
import { useUnits } from '@/hooks/useUnits'
import { useLocation, ipFallback } from '@/hooks/useLocation'
import { useReverseGeocode, formatAddress } from '@/hooks/useReverseGeocode'
import { useMapPOIs, type POICategory } from '@/hooks/useMapPOIs'
import { useDirections } from '@/hooks/useDirections'
import { useAuth } from '@/auth/AuthContext'
import { CASE_STATUS_ORDER, statusMeta } from '@/lib/status'
import { relativeTime } from '@/lib/format'
import type { Case } from '@/types/api'

const MAP_HEIGHT = '78vh'

export function MapPage() {
  const { role } = useAuth()
  const casesQuery = useCases()
  const unitsQuery = useUnits()
  const { latitude, longitude, permission, loading: locationLoading, request } = useLocation()
  const [ipLocation, setIpLocation] = useState<{ latitude: number; longitude: number } | null>(null)
  const [showEnableModal, setShowEnableModal] = useState(false)
  const ipAttemptedRef = useRef(false)

  const resolvedLat = latitude ?? ipLocation?.latitude ?? null
  const resolvedLng = longitude ?? ipLocation?.longitude ?? null
  const { data: geo, isLoading: geoLoading } = useReverseGeocode(resolvedLat, resolvedLng)

  const [selected, setSelected] = useState<string | null>(null)
  const [hidden, setHidden] = useState<Set<string>>(new Set())
  const [query, setQuery] = useState('')
  const [showCoverage, setShowCoverage] = useState(true)
  const [showHotspots, setShowHotspots] = useState(false)

  const [poiCategories, setPoiCategories] = useState<Set<POICategory>>(
    () => new Set<POICategory>(['hospital', 'police', 'bank']),
  )
  const [hiddenPOICategories, setHiddenPOICategories] = useState<Set<POICategory>>(new Set())
  const [poiPanelOpen, setPoiPanelOpen] = useState(true)

  const [directionsTo, setDirectionsTo] = useState<{
    lat: number
    lng: number
    label: string
  } | null>(null)

  const pois = useMapPOIs(resolvedLat, resolvedLng, 3000, [...poiCategories])

  useEffect(() => {
    if (permission === 'denied' && !ipAttemptedRef.current) {
      ipAttemptedRef.current = true
      void ipFallback().then((coords) => {
        if (coords) setIpLocation(coords)
      })
    }
  }, [permission])

  const userLocationForMap =
    latitude != null && longitude != null ? { latitude, longitude } : ipLocation

  useEffect(() => {
    if (permission === 'prompt') {
      const id = setTimeout(() => request(), 500)
      return () => clearTimeout(id)
    }
  }, [permission, request])

  const allCases = useMemo(() => casesQuery.data ?? [], [casesQuery.data])

  const visibleCases = useMemo(() => {
    const q = query.trim().toLowerCase()
    return allCases.filter((c) => {
      if (hidden.has(c.status)) return false
      if (!q) return true
      return [c.title, c.trackingId, c.location]
        .filter(Boolean)
        .some((field) => field!.toLowerCase().includes(q))
    })
  }, [allCases, hidden, query])

  const selectedCase = allCases.find((c) => c.id === selected) ?? null
  const caseDetailPath = (caseItem: Case) =>
    role === 'officer' ? `/officer/cases/${caseItem.id}` : undefined

  const directionsFrom =
    resolvedLat != null && resolvedLng != null ? { lat: resolvedLat, lng: resolvedLng } : null

  const directionsRoute = useDirections(directionsFrom, directionsTo)
  const isDirectionsActive = directionsTo != null

  function toggleStatus(status: string) {
    setHidden((prev) => {
      const next = new Set(prev)
      if (next.has(status)) next.delete(status)
      else next.add(status)
      return next
    })
  }

  function togglePOICategory(category: POICategory) {
    const isOn = poiCategories.has(category)
    if (isOn) {
      const next = new Set(poiCategories)
      next.delete(category)
      setPoiCategories(next)
      setHiddenPOICategories((prev) => new Set(prev).add(category))
    } else {
      const next = new Set(poiCategories)
      next.add(category)
      setPoiCategories(next)
      setHiddenPOICategories((prev) => {
        const nn = new Set(prev)
        nn.delete(category)
        return nn
      })
    }
  }

  function clearAllPOIs() {
    setPoiCategories(new Set())
    setHiddenPOICategories(new Set())
  }

  function startDirections(lat: number, lng: number, label: string) {
    setDirectionsTo({ lat, lng, label })
  }

  function stopDirections() {
    setDirectionsTo(null)
  }

  const loading = casesQuery.isLoading || unitsQuery.isLoading
  const errored = casesQuery.isError && unitsQuery.isError

  const mapMode: 'view' | 'route' = isDirectionsActive ? 'route' : 'view'
  const routeForMap = directionsRoute.data ?? null

  return (
    <div className="mx-auto w-full max-w-7xl p-4 sm:p-6">
      <header className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-ink">Operations map</h1>
          <p className="mt-1 text-sm text-ink-muted">
            {visibleCases.length} case{visibleCases.length === 1 ? '' : 's'} ·{' '}
            {unitsQuery.data?.length ?? 0} units
            {pois.data ? ` · ${pois.data.count} places` : ''}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            variant="secondary"
            icon={<MapPin className="size-4" aria-hidden />}
            disabled={locationLoading}
            onClick={request}
          >
            Locate me
          </Button>
          <Button
            size="sm"
            variant={showCoverage ? 'primary' : 'secondary'}
            icon={<Layers className="size-4" aria-hidden />}
            aria-pressed={showCoverage}
            onClick={() => setShowCoverage((v) => !v)}
          >
            Unit coverage
          </Button>
          <Button
            size="sm"
            variant={showHotspots ? 'primary' : 'secondary'}
            aria-pressed={showHotspots}
            onClick={() => setShowHotspots((v) => !v)}
          >
            Activity areas
          </Button>
        </div>
      </header>

      {permission === 'denied' && (
        <div className="mb-3 rounded-lg border border-warn/30 bg-warn/10 px-4 py-3 text-sm text-warn">
          <p className="font-medium">
            Using approximate location. Tap the target button to enable precise GPS.
          </p>
          <Button size="sm" variant="ghost" className="mt-2" onClick={() => setShowEnableModal(true)}>
            Enable precise location
          </Button>
        </div>
      )}

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
        <p className="mt-3 text-xs text-ink-muted">After enabling, refresh the page or tap "Locate me" to re-check.</p>
      </Modal>

      <div className="grid gap-4 lg:grid-cols-[1fr_22rem]">
        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative min-w-56 flex-1">
              <Search
                className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint"
                aria-hidden
              />
              <Input
                type="search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Find a case on the map"
                aria-label="Search cases on the map"
                className="pl-9"
              />
            </div>

            <div className="flex flex-wrap gap-1.5">
              {CASE_STATUS_ORDER.map((status) => {
                const active = !hidden.has(status)
                const meta = statusMeta(status)
                return (
                  <button
                    key={status}
                    type="button"
                    aria-pressed={active}
                    onClick={() => toggleStatus(status)}
                    className={cn(
                      'inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs ring-1 transition-opacity',
                      meta.bg,
                      meta.text,
                      meta.ring,
                      !active && 'opacity-40',
                    )}
                  >
                    <span className={cn('size-2 rounded-full', meta.dot)} aria-hidden />
                    {meta.label}
                  </button>
                )
              })}
            </div>
          </div>

          {errored ? (
            <Card>
              <ErrorState
                title="Could not load the map data"
                description="Cases and units failed to load. Check your connection."
                onRetry={() => {
                  void casesQuery.refetch()
                  void unitsQuery.refetch()
                }}
              />
            </Card>
          ) : loading ? (
            <div style={{ height: MAP_HEIGHT }} className="w-full">
              <Skeleton className="h-full w-full rounded-panel" />
            </div>
          ) : (
            <>
              <div className="relative">
                <MapView
                  mode={mapMode}
                  route={routeForMap}
                  cases={isDirectionsActive ? [] : visibleCases}
                  units={isDirectionsActive ? [] : unitsQuery.data ?? []}
                  pois={isDirectionsActive ? [] : pois.data?.items ?? []}
                  hiddenPOICategories={hiddenPOICategories}
                  showUnitCoverage={showCoverage && !isDirectionsActive}
                  showHotspots={showHotspots && !isDirectionsActive}
                  userLocation={userLocationForMap}
                  allowLocate
                  height={MAP_HEIGHT}
                  selectedCaseId={selected}
                  onSelectCase={(caseItem) => setSelected(caseItem.id)}
                />

                {!isDirectionsActive ? (
                  <div className="absolute bottom-3 left-3 z-20">
                    <POIControlPanel
                      active={poiCategories}
                      onToggle={togglePOICategory}
                      onClear={clearAllPOIs}
                      count={pois.data?.count ?? 0}
                      open={poiPanelOpen}
                      onToggleOpen={() => setPoiPanelOpen((v) => !v)}
                    />
                  </div>
                ) : null}
              </div>

              {isDirectionsActive ? (
                <Card className="border-signal/40">
                  <div className="flex flex-col gap-3 p-5">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="flex items-center gap-2 text-base font-semibold text-ink">
                          <Navigation className="size-5 text-signal" aria-hidden />
                          Directions
                        </p>
                        <p className="mt-0.5 text-sm text-ink-muted">
                          To: {directionsTo.label}
                        </p>
                        {directionsRoute.data ? (
                          <p className="mt-1 text-sm font-medium text-signal">
                            {directionsRoute.data.summary}
                          </p>
                        ) : directionsRoute.isLoading ? (
                          <p className="mt-1 text-sm text-ink-faint">Calculating route…</p>
                        ) : directionsRoute.isError ? (
                          <p className="mt-1 text-sm text-warn">
                            Could not calculate the route. Try again.
                          </p>
                        ) : null}
                      </div>
                      <Button
                        size="sm"
                        variant="ghost"
                        icon={<X className="size-4" aria-hidden />}
                        onClick={stopDirections}
                      >
                        Close
                      </Button>
                    </div>

                    {directionsRoute.data && directionsRoute.data.steps.length > 0 ? (
                      <ol className="flex flex-col gap-2 rounded-lg border border-border bg-surface-hi/30 p-3">
                        {directionsRoute.data.steps.slice(0, 10).map((step, i) => (
                          <li key={i} className="flex items-start gap-2 text-sm">
                            <span className="tabular-nums text-ink-faint">{i + 1}.</span>
                            <span className="flex-1 text-ink-muted">
                              {step.instruction}
                              <span className="ml-1 text-ink-faint">
                                ({Math.round(step.distanceMeters)} m)
                              </span>
                            </span>
                          </li>
                        ))}
                        {directionsRoute.data.steps.length > 10 ? (
                          <li className="text-xs text-ink-faint">
                            …and {directionsRoute.data.steps.length - 10} more steps
                          </li>
                        ) : null}
                      </ol>
                    ) : null}
                  </div>
                </Card>
              ) : resolvedLat != null &&
                resolvedLng != null &&
                (permission !== 'denied' || ipLocation != null) ? (
                <p className="mt-2 text-sm text-ink-muted" aria-live="polite">
                  {geoLoading ? (
                    <span className="inline-flex items-center gap-1.5">
                      <span
                        className="size-3.5 border-2 border-current border-t-transparent rounded-full animate-spin"
                        aria-hidden
                      />
                      Locating address…
                    </span>
                  ) : (
                    <span>Near {formatAddress(geo)}</span>
                  )}
                </p>
              ) : null}
            </>
          )}
        </div>

        <div className="flex flex-col gap-3">
          {selectedCase ? (
            <Card className="border-signal/40">
              <div className="flex flex-col gap-2 p-5">
                <div className="flex items-center justify-between gap-2">
                  <StatusChip status={selectedCase.status} />
                  <PriorityChip level={selectedCase.priorityLevel} />
                </div>
                <p className="text-base font-semibold text-ink">{selectedCase.title}</p>
                <p className="flex items-center gap-1.5 text-sm text-ink-muted">
                  <MapPin className="size-4" aria-hidden />
                  {selectedCase.location || 'Location recorded'}
                </p>
                <p className="text-xs text-ink-faint">
                  {selectedCase.trackingId} · reported {relativeTime(selectedCase.createdAt)}
                </p>

                {directionsFrom ? (
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={<Navigation className="size-4" aria-hidden />}
                    onClick={() =>
                      startDirections(
                        selectedCase.latitude,
                        selectedCase.longitude,
                        selectedCase.title,
                      )
                    }
                  >
                    Get directions
                  </Button>
                ) : null}

                {caseDetailPath(selectedCase) ? (
                  <Link
                    to={caseDetailPath(selectedCase)!}
                    className="text-sm text-signal underline-offset-2 hover:underline"
                  >
                    Open case workspace
                  </Link>
                ) : null}

                <Button size="sm" variant="ghost" onClick={() => setSelected(null)}>
                  Clear selection
                </Button>
              </div>
            </Card>
          ) : null}

          <Card className="overflow-hidden">
            <header className="border-b border-border px-4 py-3">
              <h2 className="text-sm font-semibold text-ink">On the map</h2>
            </header>
            {visibleCases.length === 0 ? (
              <EmptyState
                title="Nothing plotted"
                description="No cases match the current filters."
              />
            ) : (
              <ul className="max-h-[55vh] divide-y divide-border overflow-y-auto">
                {visibleCases.map((caseItem) => (
                  <li key={caseItem.id}>
                    <button
                      type="button"
                      onClick={() => setSelected(caseItem.id)}
                      className={cn(
                        'flex w-full flex-col gap-1 px-4 py-3 text-left transition-colors hover:bg-surface-hi',
                        selected === caseItem.id && 'bg-signal/5',
                      )}
                    >
                      <span className="flex items-center gap-2">
                        <span
                          className={cn(
                            'size-2.5 shrink-0 rounded-full',
                            statusMeta(caseItem.status).dot,
                          )}
                          aria-hidden
                        />
                        <span className="truncate text-sm font-medium text-ink">
                          {caseItem.title}
                        </span>
                      </span>
                      <span className="text-xs text-ink-faint">
                        {statusMeta(caseItem.status).label} · {relativeTime(caseItem.createdAt)}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
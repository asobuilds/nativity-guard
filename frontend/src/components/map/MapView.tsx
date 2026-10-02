import { useEffect, useMemo, useState } from 'react'
import { divIcon, type LatLngBoundsExpression, type LatLngTuple } from 'leaflet'
import {
  Circle,
  CircleMarker,
  MapContainer,
  Marker,
  Popup,
  TileLayer,
  useMap,
  useMapEvents,
} from 'react-leaflet'
import { LocateFixed, MapPin, TriangleAlert } from 'lucide-react'
import { cn } from '@/lib/cn'
import { formatCoord } from '@/lib/format'
import { activityGroups, groupCases } from '@/lib/mapGroups'
import { priorityMeta, statusMeta } from '@/lib/status'
import type { Case, SecurityUnit } from '@/types/api'
import { Button } from '@/components/ui/Button'

/**
 * The one map in the product.
 *
 * Two modes:
 *  - `view` — plot cases and unit coverage, click a case to select it.
 *  - `pick` — let the user drop/move a pin to choose coordinates.
 *
 * Tile failures degrade to a list of everything that would have been plotted
 * (with coordinates), never to a blank grey box.
 */

/** Leaflet needs literal colours — the Tailwind tokens are not available inside SVG. */
export const STATUS_HEX: Record<string, string> = {
  pending: '#8b95a7',
  assigned: '#4f8cff',
  dispatched: '#f0a63c',
  on_scene: '#c084fc',
  investigating: '#2dd4bf',
  pending_admin_review: '#818cf8',
  admin_changes_requested: '#e879f9',
  closed: '#3fbf7f',
}

const DEFAULT_CENTER: LatLngTuple = [6.5244, 3.3792] // Lagos — last-resort fallback only
const DEFAULT_ZOOM = 12

export interface MapViewProps {
  mode?: 'view' | 'pick'
  cases?: Case[]
  units?: SecurityUnit[]
  center?: LatLngTuple
  zoom?: number
  /** Any CSS length. Defaults to a responsive 60vh. */
  height?: string | number
  showUnitCoverage?: boolean
  showHotspots?: boolean
  selectedCaseId?: string | null
  onSelectCase?: (caseItem: Case) => void
  /** Current pick location in `pick` mode. */
  pickLocation?: LatLngTuple | null
  onPickLocation?: (lat: number, lng: number) => void
  /** Ask the browser for the user's position and recentre. */
  allowLocate?: boolean
  /** The reporter's live position, when already known. Recentres the map
   *  automatically in both modes — priority over the generic fallback. */
  userLocation?: { latitude: number; longitude: number } | null
  className?: string
  label?: string
}

/** Keep the viewport fitted to whatever is plotted, without fighting user panning. */
function FitToContent({ points, enabled }: { points: LatLngTuple[]; enabled: boolean }) {
  const map = useMap()
  const key = points.map((p) => p.join(',')).join('|')

  useEffect(() => {
    if (!enabled || points.length === 0) return
    if (points.length === 1) {
      map.setView(points[0], Math.max(map.getZoom(), 14))
      return
    }
    const bounds = points as LatLngBoundsExpression
    map.fitBounds(bounds, { padding: [40, 40], maxZoom: 15 })
    // `key` is the content identity; `points` is a fresh array each render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, enabled, map])

  return null
}

/** Click-to-pick behaviour, isolated so it only mounts in `pick` mode. */
function PickHandler({ onPick }: { onPick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(event) {
      onPick(event.latlng.lat, event.latlng.lng)
    },
  })
  return null
}

/**
 * Recenter the map when `target` changes.
 *
 * Dependencies are the primitive lat/lng, not the tuple itself. A tuple is
 * a fresh array on every render, so keying on the reference would fire the
 * effect on every render — fighting the user's pan and jittering the map.
 */
function Recenter({ target, zoom }: { target: LatLngTuple | null; zoom?: number }) {
  const map = useMap()
  const lat = target?.[0]
  const lng = target?.[1]
  useEffect(() => {
    if (lat == null || lng == null) return
    map.setView([lat, lng], zoom ?? Math.max(map.getZoom(), 15))
  }, [lat, lng, zoom, map])
  return null
}

const pickIcon = divIcon({
  className: '',
  html: '<div style="width:22px;height:22px;border-radius:9999px;background:#3fbf7f;border:3px solid #0b0e14;box-shadow:0 0 0 2px #3fbf7f"></div>',
  iconSize: [22, 22],
  iconAnchor: [11, 11],
})

function CaseMarkers({
  cases,
  selectedCaseId,
  onSelectCase,
  showHotspots,
}: {
  cases: Case[]
  selectedCaseId?: string | null
  onSelectCase?: (item: Case) => void
  showHotspots: boolean
}) {
  const map = useMapEvents({ zoomend: () => setZoom(map.getZoom()) })
  const [zoom, setZoom] = useState(map.getZoom())
  const project = (lat: number, lng: number) => map.project([lat, lng], zoom)
  const groups = groupCases(cases, project, 52)
  const concentrations = showHotspots && zoom <= 12
    ? activityGroups(groupCases(cases, project, 110))
    : []

  return (
    <>
      {concentrations.map((group) => (
        <Circle
          key={`activity-${group.cases.map((item) => item.id).sort().join('-')}`}
          center={[group.latitude, group.longitude]}
          radius={Math.max(350, 1800 - zoom * 90)}
          pathOptions={{ color: '#f4cb78', weight: 1, fillColor: '#f4cb78', fillOpacity: 0.16 }}
        >
          <Popup>{group.cases.length} reports in this area. This shows activity, not a prediction of danger.</Popup>
        </Circle>
      ))}
      {groups.map((group) => {
        if (group.cases.length > 1 && zoom < 17) {
          return (
            <CircleMarker
              key={`cluster-${group.cases.map((item) => item.id).sort().join('-')}`}
              center={[group.latitude, group.longitude]}
              radius={Math.min(24, 10 + Math.log2(group.cases.length) * 4)}
              pathOptions={{ color: '#f8f5e9', weight: 2, fillColor: '#254137', fillOpacity: 1 }}
              eventHandlers={{ click: () => map.flyTo([group.latitude, group.longitude], Math.min(17, zoom + 2)) }}
            >
              <Popup>{group.cases.length} reports nearby. Select the marker to zoom in.</Popup>
            </CircleMarker>
          )
        }
        return group.cases.map((caseItem) => {
          const hex = STATUS_HEX[caseItem.status] ?? STATUS_HEX.pending
          const selected = caseItem.id === selectedCaseId
          return (
            <CircleMarker
              key={caseItem.id}
              center={[caseItem.latitude, caseItem.longitude]}
              radius={selected ? 10 : 7}
              pathOptions={{
                color: selected ? '#ffffff' : '#091613',
                weight: selected ? 3 : 1.5,
                fillColor: hex,
                fillOpacity: 1,
              }}
              eventHandlers={{ click: () => onSelectCase?.(caseItem) }}
            >
              <Popup>
                <PopupBody
                  title={caseItem.title}
                  subtitle={`${statusMeta(caseItem.status).label} · ${caseItem.trackingId}`}
                  rows={[
                    ['Priority', priorityMeta(caseItem.priorityLevel).label],
                    ['Location', caseItem.location || formatCoord(caseItem.latitude, caseItem.longitude)],
                  ]}
                />
              </Popup>
            </CircleMarker>
          )
        })
      })}
    </>
  )
}

export function MapView({
  mode = 'view',
  cases = [],
  units = [],
  center,
  zoom = DEFAULT_ZOOM,
  height = '60vh',
  showUnitCoverage = true,
  showHotspots = false,
  selectedCaseId,
  onSelectCase,
  pickLocation = null,
  onPickLocation,
  allowLocate = false,
  userLocation = null,
  className,
  label = 'Map of cases and security units',
}: MapViewProps) {
  const [tileFailed, setTileFailed] = useState(false)
  const [locating, setLocating] = useState(false)
  const [locateError, setLocateError] = useState<string | null>(null)
  const [userCenter, setUserCenter] = useState<LatLngTuple | null>(null)

  const casePoints = useMemo<LatLngTuple[]>(
    () =>
      cases
        .filter((c) => Number.isFinite(c.latitude) && Number.isFinite(c.longitude))
        .map((c) => [c.latitude, c.longitude] as LatLngTuple),
    [cases],
  )

  const unitPoints = useMemo<LatLngTuple[]>(
    () =>
      units
        .filter((u) => Number.isFinite(u.latitude) && Number.isFinite(u.longitude))
        .map((u) => [u.latitude, u.longitude] as LatLngTuple),
    [units],
  )

  const allPoints = useMemo(
    () => [...casePoints, ...unitPoints],
    [casePoints, unitPoints],
  )

  const userLocationCenter: LatLngTuple | null = userLocation
    ? [userLocation.latitude, userLocation.longitude]
    : null

  /** The user's own position for display: the reported fix, or the locate button's. */
  const ownPosition: LatLngTuple | null = userLocationCenter ?? userCenter

  /**
   * Initial map position. Priority, highest first:
   *   1. Explicit `center` prop
   *   2. In pick mode, a saved pin (user already chose that point)
   *   3. The user's live location
   *   4. A placed pin (view mode)
   *   5. First plotted case / unit
   *   6. Lagos — last resort only
   */
  const initialCenter: LatLngTuple =
    center ??
    (mode === 'pick' ? (pickLocation as LatLngTuple | null) : null) ??
    userLocationCenter ??
    (pickLocation as LatLngTuple | null) ??
    casePoints[0] ??
    unitPoints[0] ??
    DEFAULT_CENTER

  function locate() {
    if (!navigator.geolocation) {
      setLocateError('This device does not report a location.')
      return
    }
    setLocating(true)
    setLocateError(null)
    navigator.geolocation.getCurrentPosition(
      (position) => {
        const next: LatLngTuple = [position.coords.latitude, position.coords.longitude]
        setUserCenter(next)
        if (mode === 'pick') onPickLocation?.(next[0], next[1])
        setLocating(false)
      },
      () => {
        setLocateError('Location permission denied — pick the point on the map instead.')
        setLocating(false)
      },
      { enableHighAccuracy: true, timeout: 10_000 },
    )
  }

  const plotted = mode === 'pick' ? [] : allPoints

  if (tileFailed) {
    return (
      <MapFallback
        cases={cases}
        units={units}
        height={height}
        className={className}
        onSelectCase={onSelectCase}
      />
    )
  }

  return (
    <div
      role="region"
      aria-label={label}
      className={cn('relative overflow-hidden rounded-panel border border-border bg-surface', className)}
    >
      <MapContainer
        center={initialCenter}
        zoom={zoom}
        style={{ height, width: '100%', background: '#091613' }}
        scrollWheelZoom
        className="z-0"
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
          eventHandlers={{ tileerror: () => setTileFailed(true) }}
        />

        {/* User's own position — always visible when available */}
        {ownPosition ? (
          <>
            {/* Outer pulse ring */}
            <CircleMarker
              center={ownPosition}
              radius={18}
              pathOptions={{
                color: '#3b82f6',
                weight: 1,
                opacity: 0.4,
                fillColor: '#3b82f6',
                fillOpacity: 0.15,
              }}
            />
            {/* Inner solid dot */}
            <CircleMarker
              center={ownPosition}
              radius={7}
              pathOptions={{
                color: '#ffffff',
                weight: 2,
                fillColor: '#3b82f6',
                fillOpacity: 1,
              }}
            />
          </>
        ) : null}

        {mode === 'view' ? (
          <>
            {userLocationCenter ? <Recenter target={userLocationCenter} /> : null}
            <FitToContent points={plotted} enabled={plotted.length > 0 && !ownPosition} />

            {showUnitCoverage
              ? units
                  .filter((u) => Number.isFinite(u.latitude) && Number.isFinite(u.longitude))
                  .map((unit) => (
                    <Circle
                      key={`coverage-${unit.id}`}
                      center={[unit.latitude, unit.longitude]}
                      radius={(unit.operationalRadius || 2) * 1000}
                      pathOptions={{
                        color: '#4f8cff',
                        weight: 1,
                        opacity: 0.5,
                        fillColor: '#4f8cff',
                        fillOpacity: 0.07,
                      }}
                    >
                      <Popup>
                        <PopupBody
                          title={unit.name}
                          subtitle={`${unit.type || 'Unit'} · ${unit.operationalRadius} km coverage`}
                          rows={[
                            ['Location', [unit.city, unit.lga, unit.state].filter(Boolean).join(', ') || '—'],
                            ['Contact', unit.contactPhone || '—'],
                          ]}
                        />
                      </Popup>
                    </Circle>
                  ))
              : null}

            {units
              .filter((u) => Number.isFinite(u.latitude) && Number.isFinite(u.longitude))
              .map((unit) => (
                <CircleMarker
                  key={`unit-${unit.id}`}
                  center={[unit.latitude, unit.longitude]}
                  radius={5}
                  pathOptions={{ color: '#0b0e14', weight: 1.5, fillColor: '#4f8cff', fillOpacity: 1 }}
                >
                  <Popup>
                    <PopupBody
                      title={unit.name}
                      subtitle={`${unit.type || 'Unit'}${unit.isVerified ? ' · Verified' : ''}`}
                      rows={[['Coordinates', formatCoord(unit.latitude, unit.longitude)]]}
                    />
                  </Popup>
                </CircleMarker>
              ))}

            <CaseMarkers cases={cases} selectedCaseId={selectedCaseId} onSelectCase={onSelectCase} showHotspots={showHotspots} />
          </>
        ) : (
          <>
            {pickLocation ? (
              <Marker
                position={pickLocation}
                icon={pickIcon}
                draggable
                eventHandlers={{
                  dragend(event) {
                    const { lat, lng } = event.target.getLatLng()
                    onPickLocation?.(lat, lng)
                  },
                }}
              />
            ) : null}
            <PickHandler onPick={(lat, lng) => onPickLocation?.(lat, lng)} />

            {/* Auto-recenter to the user's live location ONLY while no pin
                has been placed yet. Once they place a pin, panning is
                theirs to control and we stop moving the map for them. */}
            {!pickLocation && userLocationCenter ? (
              <Recenter target={userLocationCenter} />
            ) : null}

            <Recenter target={userCenter} />
          </>
        )}
      </MapContainer>

      {/* Controls + legend */}
      <div className="pointer-events-none absolute inset-x-0 top-0 flex items-start justify-between gap-2 p-2">
        <div className="pointer-events-auto flex flex-wrap gap-1.5 rounded-lg border border-border bg-base/85 px-2 py-1.5 backdrop-blur">
          {mode === 'pick' ? (
            <span className="flex items-center gap-1 text-[11px] text-ink-muted">
              <MapPin className="size-3.5 text-signal" aria-hidden />
              Tap the map to place the pin, or drag it to adjust.
            </span>
          ) : (
            Object.entries(STATUS_HEX).map(([status, hex]) => (
              <span key={status} className="flex items-center gap-1 text-[11px] text-ink-muted">
                <span className="size-2 rounded-full" style={{ background: hex }} aria-hidden />
                {statusMeta(status).label}
              </span>
            ))
          )}
        </div>

        {allowLocate ? (
          <Button
            size="sm"
            variant="secondary"
            className="pointer-events-auto"
            loading={locating}
            icon={<LocateFixed className="size-4" aria-hidden />}
            onClick={locate}
          >
            Use my location
          </Button>
        ) : null}
      </div>

      {locateError ? (
        <p className="absolute inset-x-2 bottom-2 rounded-lg border border-warn/30 bg-warn/10 px-2 py-1 text-[11px] text-warn">
          {locateError}
        </p>
      ) : null}
    </div>
  )
}

function PopupBody({
  title,
  subtitle,
  rows,
}: {
  title: string
  subtitle: string
  rows: [string, string][]
}) {
  return (
    <div className="min-w-44">
      <p className="text-sm font-semibold text-ink">{title}</p>
      <p className="text-[11px] text-ink-muted">{subtitle}</p>
      <dl className="mt-2 space-y-0.5">
        {rows.map(([key, value]) => (
          <div key={key} className="flex gap-2 text-[11px]">
            <dt className="text-ink-faint">{key}</dt>
            <dd className="text-ink">{value}</dd>
          </div>
        ))}
      </dl>
    </div>
  )
}

/** Degraded view used when map tiles cannot load. */
function MapFallback({
  cases,
  units,
  height,
  className,
  onSelectCase,
}: {
  cases: Case[]
  units: SecurityUnit[]
  height: string | number
  className?: string
  onSelectCase?: (caseItem: Case) => void
}) {
  const rows = [
    ...cases.map((c) => ({
      id: c.id,
      title: c.title,
      meta: `${statusMeta(c.status).label} · ${c.trackingId}`,
      coords: formatCoord(c.latitude, c.longitude),
      caseItem: c,
    })),
    ...units.map((u) => ({
      id: u.id,
      title: u.name,
      meta: `${u.type || 'Unit'} · ${u.operationalRadius} km`,
      coords: formatCoord(u.latitude, u.longitude),
      caseItem: null,
    })),
  ]

  return (
    <div
      style={{ minHeight: height }}
      className={cn('overflow-hidden rounded-panel border border-border bg-surface', className)}
    >
      <div className="flex items-center gap-2 border-b border-border bg-warn/10 px-3 py-2 text-xs text-warn">
        <TriangleAlert className="size-4 shrink-0" aria-hidden />
        <span>Map tiles are unavailable. Showing the plotted items as a list.</span>
      </div>
      {rows.length === 0 ? (
        <p className="px-3 py-10 text-center text-xs text-ink-muted">Nothing to plot yet.</p>
      ) : (
        <ul className="max-h-96 divide-y divide-border overflow-y-auto">
          {rows.map((row) => (
            <li key={row.id}>
              <button
                type="button"
                disabled={!row.caseItem || !onSelectCase}
                onClick={() => row.caseItem && onSelectCase?.(row.caseItem)}
                className="flex w-full items-start justify-between gap-3 px-3 py-2.5 text-left transition-colors hover:bg-surface-hi disabled:cursor-default"
              >
                <span className="min-w-0">
                  <span className="block truncate text-xs font-medium text-ink">{row.title}</span>
                  <span className="block text-[11px] text-ink-muted">{row.meta}</span>
                </span>
                <span className="shrink-0 text-[11px] tabular-nums text-ink-faint">{row.coords}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { LatLngTuple } from 'leaflet'
import {
  Compass,
  CreditCard,
  Eye,
  Filter,
  Layers,
  MapPin,
  Plus,
  Shield,
  ShieldCheck,
  Users,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Field'
import { MapView } from '@/components/map/MapView'
import { useGeoStates, useGeoLgas } from '@/hooks/useGeo'
import { usePublicUnits } from '@/hooks/usePublicUnits'
import { usePublicCases } from '@/hooks/usePublicCases'
import { useLocation, ipFallback } from '@/hooks/useLocation'
import { useReverseGeocode, formatAddress } from '@/hooks/useReverseGeocode'
import { locateFailureKind, locateFailureMessage } from '@/lib/locationNotice'
import { formatCoord } from '@/lib/format'
import type { Case, SecurityUnit } from '@/types/api'
import type { ActionContext } from './types'

export interface PublicSafetyMapSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

export function PublicSafetyMapSection({ onActionTrigger }: PublicSafetyMapSectionProps) {
  const { data: states = [] } = useGeoStates()
  const { data: units = [], isError: unitsError } = usePublicUnits()
  const { data: publicCases = [], isError: casesError } = usePublicCases()
  // `auto: false` — a visitor is never asked for GPS just by opening the
  // landing page. The position is requested when "Locate My Area" is pressed.
  const {
    latitude: userLat,
    longitude: userLng,
    permission,
    error: locationError,
    errorCode,
    request: requestLocation,
    loading: locating,
  } = useLocation({ auto: false })

  const [selectedState, setSelectedState] = useState<string>('all')
  const { data: lgas = [] } = useGeoLgas(selectedState === 'all' ? null : selectedState)
  const [selectedLga, setSelectedLga] = useState<string>('all')
  const [selectedUnitId, setSelectedUnitId] = useState<string | null>(null)
  const [showCoverage, setShowCoverage] = useState(true)
  const [showCases, setShowCases] = useState(true)
  const [pickedCoord, setPickedCoord] = useState<LatLngTuple | null>(null)
  /** The point the visitor asked the map to pinpoint. */
  const [focusPoint, setFocusPoint] = useState<LatLngTuple | null>(null)
  /** True when the point came from the network, not from GPS. */
  const [approximatePoint, setApproximatePoint] = useState(false)
  const mapRef = useRef<HTMLDivElement>(null)
  /** Set while a locate request is in flight, so the fix that lands is focused. */
  const locatePendingRef = useRef(false)

  // Reverse-geocode user location or picked location
  const activeLat = pickedCoord ? pickedCoord[0] : userLat
  const activeLng = pickedCoord ? pickedCoord[1] : userLng
  const { data: activeGeo } = useReverseGeocode(activeLat ?? null, activeLng ?? null)
  const locationLabel = formatAddress(activeGeo)

  // Filter units by state & LGA
  const filteredUnits = useMemo(() => {
    return units.filter((u) => {
      if (selectedState !== 'all' && u.state?.toLowerCase() !== selectedState.toLowerCase()) {
        return false
      }
      if (selectedLga !== 'all' && u.lga?.toLowerCase() !== selectedLga.toLowerCase()) {
        return false
      }
      return true
    })
  }, [units, selectedState, selectedLga])

  // Map public cases to Case type for MapView
  const mappedCases: Case[] = useMemo(() => {
    if (!showCases) return []
    return publicCases.map((pc) => ({
      id: pc.id,
      unitId: pc.unitId,
      reportedBy: 'anonymous',
      title: pc.title,
      description: pc.description,
      location: pc.location,
      latitude: pc.latitude,
      longitude: pc.longitude,
      status: pc.status,
      priorityLevel: pc.priorityLevel,
      trackingId: pc.trackingId,
      gisLatitude: pc.gisLatitude ?? pc.latitude,
      gisLongitude: pc.gisLongitude ?? pc.longitude,
      isPublic: true,
      createdAt: pc.createdAt,
      updatedAt: pc.updatedAt,
    }))
  }, [publicCases, showCases])

  const selectedUnit = useMemo(() => {
    return units.find((u) => u.id === selectedUnitId) ?? null
  }, [units, selectedUnitId])

  // Center coordinate determination
  const mapCenter: LatLngTuple = useMemo(() => {
    if (selectedUnit && selectedUnit.latitude && selectedUnit.longitude) {
      return [selectedUnit.latitude, selectedUnit.longitude]
    }
    if (userLat && userLng) {
      return [userLat, userLng]
    }
    if (filteredUnits.length > 0 && filteredUnits[0].latitude && filteredUnits[0].longitude) {
      return [filteredUnits[0].latitude, filteredUnits[0].longitude]
    }
    return [9.082, 8.6753] // Nigeria geographical center
  }, [selectedUnit, userLat, userLng, filteredUnits])

  /** Put the visitor's own position at the centre of the map. */
  const focusOnUser = useCallback((lat: number, lng: number, approximate = false) => {
    setApproximatePoint(approximate)
    setFocusPoint([lat, lng])
    // On a phone the map sits below the fold, so bring it into view.
    if (typeof window !== 'undefined' && window.innerWidth < 1024) {
      mapRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }, [])

  function handleLocateMe() {
    locatePendingRef.current = true
    if (userLat != null && userLng != null) {
      // A fix is already known — show it straight away, then refresh it.
      focusOnUser(userLat, userLng)
      requestLocation()
      return
    }
    requestLocation()
  }

  // A fix that arrives after the visitor pressed the button pins the map.
  useEffect(() => {
    if (!locatePendingRef.current) return
    if (userLat == null || userLng == null) return
    locatePendingRef.current = false
    focusOnUser(userLat, userLng)
  }, [userLat, userLng, focusOnUser])

  /**
   * Last resort when the browser will not hand over a GPS fix: fall back to
   * the area the network says we are in. It is an approximation, and it is
   * labelled as one — but it beats leaving the visitor with a dead map.
   */
  const showApproximateArea = useCallback(async () => {
    const approximate = await ipFallback()
    if (!approximate) return
    focusOnUser(approximate.latitude, approximate.longitude, true)
  }, [focusOnUser])

  // A failed request falls back to the approximate area.
  useEffect(() => {
    if (!locatePendingRef.current) return
    if (!locationError) return
    locatePendingRef.current = false
    void showApproximateArea()
  }, [locationError, showApproximateArea])

  const locateNotice = useMemo(() => {
    const kind = locateFailureKind({
      errorCode,
      permission,
      secureContext: typeof window === 'undefined' ? true : window.isSecureContext,
      supported: typeof navigator === 'undefined' ? true : 'geolocation' in navigator,
    })
    if (!kind) return null
    const reason = locateFailureMessage(kind)
    return approximatePoint
      ? `${reason} Showing an approximate area from your network in the meantime.`
      : reason
  }, [errorCode, permission, approximatePoint])

  const locatedLabel = useMemo(() => {
    if (!focusPoint) return null
    const place = locationLabel || formatCoord(focusPoint[0], focusPoint[1])
    return approximatePoint ? `approximate area — ${place}` : place
  }, [focusPoint, locationLabel, approximatePoint])

  function handleReportAtLocation() {
    // Never replace a missing visitor location with Nigeria's map-center coordinates.
    const lat = pickedCoord?.[0] ?? userLat ?? focusPoint?.[0] ?? null
    const lng = pickedCoord?.[1] ?? userLng ?? focusPoint?.[1] ?? null
    const hasLocation = lat !== null && lng !== null
    onActionTrigger({
      type: 'report_incident',
      title: 'File Incident Report',
      description: hasLocation
        ? 'Sign in to begin a report. Review the selected location before submitting.'
        : 'Sign in to begin a report. Choose the incident location on the reporting form.',
      badgeText: hasLocation
        ? approximatePoint ? 'Approximate area selected — confirm before reporting' : 'Map location selected'
        : 'Incident location not yet selected',
      returnUrl: '/report',
      payload: hasLocation ? { lat, lng, locationLabel } : {},
    })
  }

  function handleJoinUnit(unit: SecurityUnit) {
    onActionTrigger({
      type: 'join_unit',
      title: `Apply to Join ${unit.name}`,
      description: `Official unit membership requires a verified citizen account. Background verification applies.`,
      badgeText: `Unit: ${unit.name} (${unit.lga}, ${unit.state})`,
      returnUrl: `/units/${unit.id}`,
      payload: { unitId: unit.id, unitName: unit.name },
    })
  }

  return (
    <section id="safety-map" className="border-t border-border py-14">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
            <Compass className="size-3.5 text-signal" aria-hidden />
            Interactive Coverage Explorer
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Live Community Safety Map
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            Explore published security unit and incident records where available.
            Displayed records do not imply nationwide coverage.
          </p>
        </div>

        <div className="pt-2 md:pt-0">
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={handleLocateMe}
              loading={locating}
              icon={<MapPin className="size-3.5 text-signal" />}
            >
              Locate My Area
            </Button>
            <Button
              variant="primary"
              size="sm"
              onClick={handleReportAtLocation}
              icon={<Plus className="size-3.5" />}
            >
              Report Incident Here
            </Button>
          </div>

          {locatedLabel ? (
            <p className="mt-2 flex items-center gap-1.5 text-xs text-ink-muted">
              <MapPin className="size-3.5 text-signal" aria-hidden />
              Showing your area: {locatedLabel}
            </p>
          ) : null}

          {locateNotice ? (
            <p className="mt-2 max-w-sm text-xs text-warn">{locateNotice}</p>
          ) : null}
        </div>
      </div>

      {(unitsError || casesError) ? (
        <p role="alert" className="mt-4 rounded-panel border border-warn/30 p-3 text-xs text-warn">
          Some public map data could not be loaded. No sample incidents or units are substituted.
        </p>
      ) : null}

      {/* Filter and Control Bar */}
      <Card className="mt-6 p-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-1.5 text-xs font-medium text-ink-muted">
            <Filter className="size-3.5 text-signal" aria-hidden />
            <span>Filter Jurisdiction:</span>
          </div>

          <div className="min-w-44 flex-1 sm:flex-initial">
            <Select
              value={selectedState}
              onChange={(e) => {
                setSelectedState(e.target.value)
                setSelectedLga('all')
              }}
              className="text-xs py-1.5"
            >
              <option value="all">All States ({units.length} units)</option>
              {states.map((s) => (
                <option key={s.name} value={s.name}>
                  {s.name} ({s.zone})
                </option>
              ))}
            </Select>
          </div>

          {selectedState !== 'all' ? (
            <div className="min-w-44 flex-1 sm:flex-initial">
              <Select
                value={selectedLga}
                onChange={(e) => setSelectedLga(e.target.value)}
                className="text-xs py-1.5"
              >
                <option value="all">All LGAs in {selectedState}</option>
                {lgas.map((lga) => (
                  <option key={lga} value={lga}>
                    {lga}
                  </option>
                ))}
              </Select>
            </div>
          ) : null}

          <div className="flex items-center gap-4 ml-auto pt-1 sm:pt-0">
            <label className="flex items-center gap-2 text-xs text-ink-muted cursor-pointer select-none">
              <input
                type="checkbox"
                checked={showCoverage}
                onChange={(e) => setShowCoverage(e.target.checked)}
                className="rounded border-border bg-surface text-signal focus:ring-signal"
              />
              <span className="flex items-center gap-1">
                <Layers className="size-3 text-signal" aria-hidden />
                Patrol Radii
              </span>
            </label>

            <label className="flex items-center gap-2 text-xs text-ink-muted cursor-pointer select-none">
              <input
                type="checkbox"
                checked={showCases}
                onChange={(e) => setShowCases(e.target.checked)}
                className="rounded border-border bg-surface text-signal focus:ring-signal"
              />
              <span className="flex items-center gap-1">
                <Eye className="size-3 text-status-investigating" aria-hidden />
                Public Incidents
              </span>
            </label>
          </div>
        </div>
      </Card>

      {/* Map Canvas and Selected Unit Drawer */}
      <div className="mt-4 grid gap-4 lg:grid-cols-3">
        <div ref={mapRef} className="overflow-hidden rounded-panel border border-border bg-surface lg:col-span-2">
          <MapView
            height="460px"
            center={mapCenter}
            zoom={selectedState === 'all' ? 6 : 11}
            units={filteredUnits}
            cases={mappedCases}
            showUnitCoverage={showCoverage}
            allowLocate
            userLocation={
              focusPoint ? { latitude: focusPoint[0], longitude: focusPoint[1] } : null
            }
            approximateUserLocation={approximatePoint}
            focusLocation={focusPoint}
            focusZoom={15}
            autoRecenter={false}
            onPickLocation={(lat, lng) => setPickedCoord([lat, lng])}
          />
        </div>

        {/* Selected Unit Details & Community Enlistment */}
        <div className="flex flex-col gap-3">
          {selectedUnit ? (
            <Card className="flex flex-col justify-between p-5 h-full">
              <div>
                <div className="flex items-start justify-between gap-2">
                  <span className="inline-flex items-center gap-1 rounded-full bg-ok/10 px-2.5 py-0.5 text-[11px] font-medium text-ok ring-1 ring-ok/30">
                    <ShieldCheck className="size-3.5" aria-hidden />
                    {selectedUnit.isVerified ? 'Platform-verified unit' : 'Verification pending'}
                  </span>
                  <span className="text-[11px] text-ink-faint tabular">
                    Radius: {selectedUnit.operationalRadius} km
                  </span>
                </div>

                <h3 className="mt-3 text-lg font-semibold text-ink leading-snug">
                  {selectedUnit.name}
                </h3>
                <p className="mt-1 text-xs text-ink-muted">
                  {selectedUnit.type} · {selectedUnit.lga}, {selectedUnit.state}
                </p>

                <div className="mt-4 space-y-2 border-t border-border pt-3 text-xs text-ink-muted">
                  <div className="flex items-center justify-between">
                    <span className="flex items-center gap-1.5">
                      <Users className="size-3.5 text-signal" aria-hidden />
                      Active Roster:
                    </span>
                    <span className="font-medium text-ink tabular">
                      {selectedUnit.totalMembers ?? 'Unknown'} members
                    </span>
                  </div>

                  {selectedUnit.contactPerson ? (
                    <div className="flex items-center justify-between">
                      <span>Command Contact:</span>
                      <span className="font-medium text-ink">{selectedUnit.contactPerson}</span>
                    </div>
                  ) : null}

                  {selectedUnit.registrationNumber ? (
                    <div className="flex items-center justify-between">
                      <span>Registry ID:</span>
                      <span className="font-mono text-[10px] text-ink-muted">{selectedUnit.registrationNumber}</span>
                    </div>
                  ) : null}
                </div>
              </div>

              <div className="mt-5 space-y-2 pt-3 border-t border-border">
                <Button
                  variant="primary"
                  size="sm"
                  block
                  onClick={() => handleJoinUnit(selectedUnit)}
                  icon={<Shield className="size-3.5" />}
                >
                  Apply to Join this Unit
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  block
                  disabled={!selectedUnit.isVerified}
                  onClick={() =>
                    onActionTrigger({
                      type: 'join_unit',
                      title: `Support ${selectedUnit.name}`,
                      description: `Donation details, if available, should be verified directly with ${selectedUnit.name} before payment.`,
                      badgeText: `Published donation details`,
                      returnUrl: `/units/${selectedUnit.id}`,
                    })
                  }
                  icon={<CreditCard className="size-3.5" />}
                >
                  {selectedUnit.isVerified ? 'View donation information' : 'Donation unavailable'}
                </Button>
              </div>
            </Card>
          ) : (
            <Card className="flex flex-col items-center justify-center p-6 text-center h-full">
              <span className="grid size-12 place-items-center rounded-full bg-surface-hi text-signal">
                <MapPin className="size-6" aria-hidden />
              </span>
              <h3 className="mt-3 text-sm font-semibold text-ink">Explore Security Units</h3>
              <p className="mt-1.5 text-xs leading-relaxed text-ink-muted">
                Select any unit marker or use the filters above to view its operational radius,
                command roster, and direct verified donation details.
              </p>
              <div className="mt-4 flex flex-wrap justify-center gap-1.5">
                {filteredUnits.slice(0, 3).map((u) => (
                  <button
                    key={u.id}
                    type="button"
                    onClick={() => setSelectedUnitId(u.id)}
                    className="rounded-full bg-surface-hi px-2.5 py-1 text-[11px] text-ink-muted hover:text-ink hover:bg-surface-hi/80 transition-colors"
                  >
                    {u.name.split(' ')[0]}...
                  </button>
                ))}
              </div>
            </Card>
          )}
        </div>
      </div>
    </section>
  )
}


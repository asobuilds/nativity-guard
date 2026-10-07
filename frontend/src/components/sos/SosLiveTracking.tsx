import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { MapPin, Radio } from 'lucide-react'
import { api } from '@/lib/apiClient'
import { freshResponderLocations, isSosActive, type ResponderLocation, type SosDetail, type SosResponder } from '@/lib/sosTracking'
import { createSosLocationSession } from '@/lib/sosLocationSession'
import { MapView } from '@/components/map/MapView'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Field'

interface LocationResponse { responders: ResponderLocation[]; serverTime: string }
interface Candidate { userId: string; firstName: string; lastName: string }

function OfficerAssignment({ sos, responder }: { sos: SosDetail; responder: SosResponder }) {
  const client = useQueryClient()
  const base = `/sos/${sos.id}/responders/${responder.unitId}`
  const candidates = useQuery({
    queryKey: ['sos', 'officers', sos.id, responder.unitId],
    queryFn: () => api.get<{ officers: Candidate[] }>(`${base}/officers`),
    enabled: isSosActive(sos),
    retry: false,
  })
  const [selected, setSelected] = useState(responder.assignedUserId ?? '')
  useEffect(() => { setSelected(responder.assignedUserId ?? '') }, [responder.assignedUserId])
  const assign = useMutation({
    mutationFn: () => api.put(`${base}/officer`, { userId: selected || null }),
    onSuccess: () => { void client.invalidateQueries({ queryKey: ['sos', 'detail', sos.id] }) },
  })
  return (
    <div className="mt-3 space-y-2">
      <label className="block text-xs text-ink-muted" htmlFor={`officer-${responder.id}`}>Officer travelling to this SOS</label>
      <div className="flex flex-wrap gap-2">
        <Select id={`officer-${responder.id}`} value={selected} onChange={(e) => setSelected(e.target.value)} disabled={candidates.isLoading || candidates.isError || !isSosActive(sos)}>
          <option value="">Unassigned</option>
          {candidates.data?.officers.map((o) => <option key={o.userId} value={o.userId}>{o.firstName} {o.lastName}</option>)}
        </Select>
        <Button size="sm" disabled={!isSosActive(sos) || candidates.isError || selected === (responder.assignedUserId ?? '')} loading={assign.isPending} onClick={() => assign.mutate()}>Save assignment</Button>
      </div>
      {candidates.isError ? <p role="alert" className="text-xs text-warn">Could not load the officer list.</p> : null}
      {candidates.data?.officers.length === 0 ? <p className="text-xs text-ink-muted">This unit needs an active officer membership linked to a user account.</p> : null}
      {assign.isError ? <p role="alert" className="text-xs text-warn">{assign.error.message}</p> : null}
    </div>
  )
}

function OfficerSharing({ sos, responder }: { sos: SosDetail; responder: SosResponder }) {
  const [state, setState] = useState<'starting' | 'sharing' | 'stopped'>('stopped')
  const [message, setMessage] = useState('Start sharing when you are responding. You can stop at any time.')
  const session = useRef<ReturnType<typeof createSosLocationSession> | null>(null)
  const mounted = useRef(true)
  const active = isSosActive(sos)
  const base = `/sos/${sos.id}/responders/${responder.unitId}`
  const preference = useQuery({ queryKey: ['location-sharing'], queryFn: () => api.get<{ enabled: boolean }>('/location/sharing'), refetchOnWindowFocus: true })
  useEffect(() => {
    mounted.current = true
    const stop = () => { void session.current?.stop() }
    const visibility = () => { if (document.hidden) stop() }
    document.addEventListener('visibilitychange', visibility)
    window.addEventListener('pagehide', stop)
    return () => {
      mounted.current = false
      stop()
      document.removeEventListener('visibilitychange', visibility)
      window.removeEventListener('pagehide', stop)
    }
  }, [])
  useEffect(() => { if (!active || preference.data?.enabled === false) void session.current?.stop() }, [active, preference.data?.enabled])
  function start() {
    if (!navigator.geolocation || !window.isSecureContext) {
      setMessage('Location sharing needs HTTPS (or localhost) and a browser with GPS support.')
      return
    }
    session.current = createSosLocationSession(navigator.geolocation, {
      start: async () => (await api.post<{ sessionId: string }>(`${base}/tracking`, {})).sessionId,
      upload: (sessionId, coords) => api.put(`${base}/location`, { sessionId, latitude: coords.latitude, longitude: coords.longitude, accuracy: coords.accuracy }),
      stop: (sessionId) => api.delete(`${base}/tracking?sessionId=${encodeURIComponent(sessionId)}`),
    }, (next, text) => { if (mounted.current) { setState(next); setMessage(text) } })
  }
  return (
    <div className="mt-3 space-y-2 rounded-lg bg-signal/10 p-3">
      <p className="text-sm text-ink">You are assigned to this response.</p>
      <p role="status" className="text-xs text-ink-muted">{message}</p>
      {preference.data?.enabled === false ? <Link className="text-sm text-signal underline" to="/settings">Enable location sharing in Settings</Link> : null}
      {preference.isError ? <p role="alert" className="text-xs text-warn">Could not check your sharing preference. Refresh to try again.</p> : null}
      {state === 'stopped' ? <Button size="sm" icon={<Radio className="size-4" />} disabled={!active || !preference.data?.enabled} onClick={start}>Start sharing for this SOS</Button> : <Button size="sm" variant="danger" onClick={() => void session.current?.stop()}>Stop sharing</Button>}
      <p className="text-xs text-ink-faint">The reporter and authorized response staff can see your shared position. Sharing stops when you hide or leave this page.</p>
    </div>
  )
}

export function SosLiveTracking({ sos }: { sos: SosDetail }) {
  const active = isSosActive(sos)
  const [now, setNow] = useState(Date.now())
  useEffect(() => { const timer = window.setInterval(() => setNow(Date.now()), 5_000); return () => window.clearInterval(timer) }, [])
  const locations = useQuery({
    queryKey: ['sos', 'detail', sos.id, 'locations'],
    queryFn: async () => ({ ...(await api.get<LocationResponse>(`/sos/${sos.id}/responders/locations`)), receivedAt: Date.now() }),
    enabled: active,
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
    retry: false,
  })
  // Use server time plus elapsed local time, avoiding device clock skew.
  const serverNow = locations.data ? Date.parse(locations.data.serverTime) + Math.max(0, now - locations.data.receivedAt) : now
  const fresh = active && !locations.isError ? freshResponderLocations(locations.data?.responders ?? [], serverNow) : []
  const validSOS = Number.isFinite(sos.latitude) && Math.abs(sos.latitude) <= 90 && Number.isFinite(sos.longitude) && Math.abs(sos.longitude) <= 180
  const markers = [
    ...(validSOS ? [{ id: 'sos', latitude: sos.latitude, longitude: sos.longitude, label: 'SOS location', detail: 'Reported emergency location', color: '#f97316' }] : []),
    ...fresh.map((r) => ({ id: r.id, latitude: r.latitude, longitude: r.longitude, label: sos.responders.find((unit) => unit.unitId === r.unitId)?.unit?.name ?? 'Responding unit', detail: `Shared location · accuracy ${Math.round(r.accuracy)} m`, color: '#22c55e' })),
  ]
  return (
    <Card className="space-y-4 p-5">
      <h2 className="flex items-center gap-2 text-sm font-semibold text-ink"><MapPin className="size-4 text-signal" />Responder locations</h2>
      <MapView mode="view" cases={[]} units={[]} markers={markers} height="36vh" showHotspots={false} showUnitCoverage={false} allowLocate={false} label="SOS and shared responder locations" />
      <p role="status" className="text-xs text-ink-muted">{!active ? 'This SOS is closed. Live location sharing has ended.' : locations.isError ? 'Locations could not be refreshed. Previous positions are hidden.' : locations.isLoading ? 'Checking shared locations…' : fresh.length === 0 ? 'No recent shared location. A unit accepting an SOS does not confirm that an officer is travelling.' : 'Orange: SOS location. Green: shared responder position. Updates refresh every 10 seconds; positions expire after two minutes.'}</p>
      {locations.isError && active ? <Button size="sm" variant="ghost" onClick={() => void locations.refetch()}>Retry locations</Button> : null}
      {fresh.length > 0 ? <ul className="space-y-1 text-xs text-ink-muted">{fresh.map((r) => <li key={r.id}>{sos.responders.find((unit) => unit.unitId === r.unitId)?.unit?.name ?? 'Responding unit'} · {r.latitude.toFixed(5)}, {r.longitude.toFixed(5)} · accuracy {Math.round(r.accuracy)} m</li>)}</ul> : null}
      {sos.responders.map((r) => <div key={r.id} className="rounded-lg border border-border p-3">
        <p className="text-sm text-ink">{r.unit?.name ?? 'Unit'} · {r.assigned ? 'Officer assigned' : 'Awaiting officer assignment'}</p>
        {r.canManage && active ? <OfficerAssignment sos={sos} responder={r} /> : null}
        {r.isMyAssignment && active ? <OfficerSharing sos={sos} responder={r} /> : null}
      </div>)}
    </Card>
  )
}

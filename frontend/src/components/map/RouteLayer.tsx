import { useEffect } from 'react'
import { Polyline, useMap } from 'react-leaflet'
import type { LatLngBoundsExpression } from 'leaflet'
import type { Route } from '@/hooks/useDirections'
import { decodePolyline } from '@/lib/polyline'

interface RouteLayerProps {
  route: Route
}

/**
 * Renders a route as a two-layer polyline: a wide transparent shadow
 * underneath, and the signal-coloured route on top with rounded joins. The
 * map auto-fits to the route bounds once, on first paint of a new route.
 */
export function RouteLayer({ route }: RouteLayerProps) {
  const map = useMap()
  const positions = decodePolyline(route.geometry)

  useEffect(() => {
    if (positions.length < 2) return
    const bounds = positions as LatLngBoundsExpression
    map.fitBounds(bounds, { padding: [40, 40], maxZoom: 16 })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [route.geometry, map])

  if (positions.length < 2) return null

  return (
    <>
      {/* Shadow line */}
      <Polyline
        positions={positions}
        pathOptions={{
          color: '#000000',
          weight: 9,
          opacity: 0.25,
          lineJoin: 'round',
          lineCap: 'round',
        }}
      />
      {/* Main line */}
      <Polyline
        positions={positions}
        pathOptions={{
          color: '#f4cb78',
          weight: 6,
          opacity: 1,
          lineJoin: 'round',
          lineCap: 'round',
        }}
      />
    </>
  )
}
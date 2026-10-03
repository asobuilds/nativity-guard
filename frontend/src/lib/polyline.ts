import type { LatLngTuple } from 'leaflet'

/**
 * Decodes an encoded polyline string (Google's algorithm, which OSRM uses)
 * into an array of [lat, lng] tuples suitable for Leaflet.
 *
 * Reference: https://developers.google.com/maps/documentation/utilities/polylinealgorithm
 */
export function decodePolyline(encoded: string): LatLngTuple[] {
  if (!encoded) return []

  const points: LatLngTuple[] = []
  let index = 0
  let lat = 0
  let lng = 0

  while (index < encoded.length) {
    let result = 1
    let shift = 0
    let b: number

    do {
      b = encoded.charCodeAt(index++) - 63 - 1
      result += b << shift
      shift += 5
    } while (b >= 0x1f)
    lat += result & 1 ? ~(result >> 1) : result >> 1

    result = 1
    shift = 0

    do {
      b = encoded.charCodeAt(index++) - 63 - 1
      result += b << shift
      shift += 5
    } while (b >= 0x1f)
    lng += result & 1 ? ~(result >> 1) : result >> 1

    points.push([lat * 1e-5, lng * 1e-5])
  }

  return points
}
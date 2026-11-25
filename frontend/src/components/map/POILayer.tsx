import { CircleMarker, Popup } from 'react-leaflet'
import type { POI, POICategory } from '@/hooks/useMapPOIs'
import { POI_META } from '@/hooks/useMapPOIs'
import type { UnitLandmark } from '@/hooks/useUnitLandmarks'

interface POILayerProps {
  items: POI[]
  hidden?: Set<POICategory>
  customLandmarks?: UnitLandmark[]
}

function getLandmarkStyle(type: string) {
  switch (type.toLowerCase()) {
    case 'safe house':
      return { color: '#ffffff', fillColor: '#10b981', radius: 10, label: '🛡️ Safe House' }
    case 'known hotspot':
      return { color: '#ef4444', fillColor: '#ef4444', radius: 10, label: '⚠️ Known Hotspot' }
    case 'checkpoint':
      return { color: '#3b82f6', fillColor: '#3b82f6', radius: 9, label: '🛂 Checkpoint' }
    default:
      return { color: '#f59e0b', fillColor: '#f59e0b', radius: 9, label: type || 'Custom Landmark' }
  }
}

export function POILayer({ items, hidden, customLandmarks = [] }: POILayerProps) {
  const visible =
    hidden && hidden.size > 0 ? items.filter((p) => !hidden.has(p.category)) : items

  return (
    <>
      {visible.map((poi) => {
        const meta = POI_META[poi.category]
        if (!meta) return null
        return (
          <CircleMarker
            key={poi.id}
            center={[poi.latitude, poi.longitude]}
            radius={8}
            pathOptions={{
              color: '#ffffff',
              weight: 2,
              fillColor: meta.color,
              fillOpacity: 1,
            }}
          >
            <Popup>
              <div className="min-w-44">
                <p className="text-sm font-semibold text-ink">{poi.name}</p>
                <p className="text-[11px] text-ink-muted">{meta.label}</p>
                {poi.address ? (
                  <p className="mt-1.5 text-[11px] text-ink-muted">{poi.address}</p>
                ) : null}
                {poi.phone ? (
                  <p className="mt-1 text-[11px]">
                    <a href={`tel:${poi.phone}`} className="text-signal hover:underline">
                      {poi.phone}
                    </a>
                  </p>
                ) : null}
              </div>
            </Popup>
          </CircleMarker>
        )
      })}

      {customLandmarks.map((landmark) => {
        const style = getLandmarkStyle(landmark.type)
        return (
          <CircleMarker
            key={`custom-landmark-${landmark.id}`}
            center={[landmark.latitude, landmark.longitude]}
            radius={style.radius}
            pathOptions={{
              color: style.color,
              weight: 3,
              fillColor: style.fillColor,
              fillOpacity: 1,
            }}
          >
            <Popup>
              <div className="min-w-44">
                <p className="text-sm font-semibold text-ink">{landmark.name}</p>
                <p className="text-[11px] font-medium text-signal mt-0.5">{style.label}</p>
                <p className="mt-1 text-[11px] text-ink-faint">Custom Unit Landmark</p>
              </div>
            </Popup>
          </CircleMarker>
        )
      })}
    </>
  )
}

import { CircleMarker, Popup } from 'react-leaflet'
import type { POI, POICategory } from '@/hooks/useMapPOIs'
import { POI_META } from '@/hooks/useMapPOIs'

interface POILayerProps {
  items: POI[]
  hidden?: Set<POICategory>
}

/**
 * Points of interest rendered as coloured circles with a white border.
 * Each category has a distinct colour from POI_META. A Popup shows the name,
 * category, address and phone.
 */
export function POILayer({ items, hidden }: POILayerProps) {
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
    </>
  )
}

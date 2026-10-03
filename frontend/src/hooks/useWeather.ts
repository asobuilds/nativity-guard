import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface WeatherCurrent {
  temperature: number
  apparentTemperature: number
  weatherCode: number
  weatherLabel: string
  icon: 'sun' | 'partly-cloudy' | 'cloud' | 'fog' | 'drizzle' | 'rain' | 'snow' | 'thunderstorm'
  windKph: number
  humidity: number
  isDay: boolean
  observedAt: string
}

export interface WeatherHourly {
  time: string
  temperature: number
  weatherCode: number
  weatherLabel: string
  icon: WeatherCurrent['icon']
  precipitationProbability: number
}

export interface WeatherDaily {
  date: string
  tempMax: number
  tempMin: number
  weatherCode: number
  weatherLabel: string
  icon: WeatherCurrent['icon']
  precipitationProbability: number
}

export interface WeatherBundle {
  location: string
  timezone: string
  current: WeatherCurrent
  hourly: WeatherHourly[]
  daily: WeatherDaily[]
  advisory?: string
  alert?: string
  approximate: boolean
}

/**
 * Weather for a coordinate pair. The backend caches 15 minutes per rounded
 * location, so repeat calls within a session are cheap.
 *
 * Pass `enabled: false` when the caller has no coordinates yet — this avoids
 * a useless request and a wasted LLM advisory call.
 */
export function useWeather(
  latitude: number | null | undefined,
  longitude: number | null | undefined,
  label: string,
  approximate: boolean,
  enabled = true,
) {
  const hasCoords =
    typeof latitude === 'number' &&
    typeof longitude === 'number' &&
    latitude !== 0 &&
    longitude !== 0

  return useQuery({
    queryKey: ['weather', latitude?.toFixed(3), longitude?.toFixed(3), label],
    queryFn: () => {
      const qs = new URLSearchParams({
        lat: String(latitude),
        lng: String(longitude),
        label: label || 'Your area',
        approximate: approximate ? 'true' : 'false',
      })
      return api.get<WeatherBundle>(`/weather/current?${qs.toString()}`)
    },
    enabled: enabled && hasCoords,
    staleTime: 10 * 60_000, // 10 minutes — backend caches 15
    refetchOnWindowFocus: false,
    retry: 1,
  })
}
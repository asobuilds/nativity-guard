import type { SecurityUnit } from '@/types/api'

/** Live platform-unit count. An unavailable API is never misrepresented as zero. */
export function publishedUnitCount(
  units: readonly Pick<SecurityUnit, 'status' | 'isVerified'>[] | undefined,
  isPending: boolean,
  isError: boolean,
): { value: string; label: string } {
  if (isError) return { value: '—', label: 'Unit count unavailable' }
  if (isPending || !units) return { value: '—', label: 'Loading unit count' }

  return {
    value: String(units.filter(unit => unit.status === 'active' && unit.isVerified === true).length),
    label: 'Active platform-verified units',
  }
}

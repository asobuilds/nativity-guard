import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { MapPin, Search, Shield } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Input, Select } from '@/components/ui/Field'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { useUnits, useNearbyUnits } from '@/hooks/useUnits'
import { useLocation } from '@/hooks/useLocation'
import type { SecurityUnit, UnitWithDistance } from '@/types/api'

type SortKey = 'name' | 'distance' | 'members'

function isWithDistance(u: SecurityUnit | UnitWithDistance): u is UnitWithDistance {
  return typeof (u as UnitWithDistance).distance === 'number'
}

export function UnitsPage() {
  const { latitude, longitude } = useLocation()
  const all = useUnits()
  const nearby = useNearbyUnits(latitude ?? undefined, longitude ?? undefined, 500)

  const [search, setSearch] = useState('')
  const [stateFilter, setStateFilter] = useState('all')
  const [typeFilter, setTypeFilter] = useState('all')
  const [sort, setSort] = useState<SortKey>(latitude != null ? 'distance' : 'name')

  const units = useMemo(() => {
    const source: (SecurityUnit | UnitWithDistance)[] =
      (nearby.data && nearby.data.length > 0) ? nearby.data : (all.data ?? [])
    const q = search.trim().toLowerCase()
    const filtered = source.filter((u) => {
      if (stateFilter !== 'all' && u.state !== stateFilter) return false
      if (typeFilter !== 'all' && u.type !== typeFilter) return false
      if (!q) return true
      return [u.name, u.state, u.lga, u.city].filter(Boolean).join(' ').toLowerCase().includes(q)
    })
    const sorted = [...filtered]
    if (sort === 'distance') {
      sorted.sort((a, b) => {
        const da = isWithDistance(a) ? a.distance : Number.POSITIVE_INFINITY
        const db = isWithDistance(b) ? b.distance : Number.POSITIVE_INFINITY
        return da - db
      })
    } else if (sort === 'members') {
      sorted.sort((a, b) => (b.totalMembers ?? 0) - (a.totalMembers ?? 0))
    } else {
      sorted.sort((a, b) => a.name.localeCompare(b.name))
    }
    return sorted
  }, [all.data, nearby.data, search, stateFilter, typeFilter, sort])

  const states = useMemo(() => {
    const set = new Set<string>()
    for (const u of (all.data ?? [])) if (u.state) set.add(u.state)
    return Array.from(set).sort()
  }, [all.data])

  const types = useMemo(() => {
    const set = new Set<string>()
    for (const u of (all.data ?? [])) if (u.type) set.add(u.type)
    return Array.from(set).sort()
  }, [all.data])

  const loading = all.isLoading && nearby.isLoading
  const errored = all.isError && nearby.isError

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-ink">Security units</h1>
        <p className="mt-1 text-sm text-ink-muted">{units.length} units · sorted by {sort}</p>
      </header>

      <Card>
        <div className="flex flex-wrap gap-2">
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
            <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search name, state, LGA" className="pl-9" />
          </div>
          <Select value={stateFilter} onChange={(e) => setStateFilter(e.target.value)}>
            <option value="all">All states</option>
            {states.map((s) => <option key={s} value={s}>{s}</option>)}
          </Select>
          <Select value={typeFilter} onChange={(e) => setTypeFilter(e.target.value)}>
            <option value="all">All types</option>
            {types.map((t) => <option key={t} value={t}>{t}</option>)}
          </Select>
          <Select value={sort} onChange={(e) => setSort(e.target.value as SortKey)}>
            <option value="distance">Nearest</option>
            <option value="name">Name</option>
            <option value="members">Members</option>
          </Select>
        </div>
      </Card>

      {loading ? (
        <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2, 3, 4, 5].map((i) => <Skeleton key={i} className="h-32 w-full" />)}
        </div>
      ) : errored ? (
        <Card><ErrorState title="Could not load units" description="Try again." onRetry={() => { void all.refetch(); void nearby.refetch() }} /></Card>
      ) : units.length === 0 ? (
        <Card><EmptyState title="No units found" description="Adjust the filters above or check back later." /></Card>
      ) : (
        <ul className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
          {units.map((u) => {
            const distance = isWithDistance(u) ? u.distance : null
            return (
              <li key={u.id}>
                <Link to={`/units/${u.id}`} className="block h-full rounded-panel focus-visible:outline-2 focus-visible:outline-signal">
                  <Card className="flex h-full flex-col gap-2 p-4 transition-colors hover:border-signal/60">
                    <div className="flex items-start justify-between gap-2">
                      <h2 className="text-sm font-semibold text-ink">{u.name}</h2>
                      {u.verificationStatus === 'verified' ? <Shield className="size-4 shrink-0 text-signal" aria-hidden /> : null}
                    </div>
                    <p className="text-xs text-ink-muted">{u.type}</p>
                    {[u.city, u.lga, u.state].filter(Boolean).length > 0 ? (
                      <p className="flex items-center gap-1 text-xs text-ink-muted">
                        <MapPin className="size-3" aria-hidden /> {[u.city, u.lga, u.state].filter(Boolean).join(', ')}
                      </p>
                    ) : null}
                    <div className="mt-auto flex items-center justify-between pt-2 text-xs">
                      <span className="text-ink-faint">{u.totalMembers ?? 0} members</span>
                      {distance != null ? <span className="text-signal tabular-nums">{distance.toFixed(1)} km</span> : null}
                    </div>
                  </Card>
                </Link>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type {
  NearbyUnitsResponse,
  UnitResponse,
  UnitWithDistance,
  UnitsResponse,
} from '@/types/api'

export const unitKeys = {
  all: ['units'] as const,
  nearby: (lat: number, lng: number, radius: number) =>
    ['units', 'nearby', lat, lng, radius] as const,
}

/** All active security units (public endpoint, no auth required). */
export function useUnits() {
  return useQuery({
    queryKey: unitKeys.all,
    queryFn: () => api.get<UnitsResponse>('/units'),
    select: (data) => data.units,
    staleTime: 5 * 60_000,
  })
}

/**
 * Units near a coordinate, enriched with `distance` and `isInRange`.
 * Disabled until a coordinate is supplied.
 *
 * The `select` is annotated `UnitWithDistance[]`, not `SecurityUnit[]`: the old
 * annotation widened the elements and threw away exactly the two fields this
 * endpoint exists to add. It still type-checked, which is why it survived —
 * `SecurityUnit` is the base, so every use compiled while `distance` was
 * unreachable. Narrow it to the base type only if a caller genuinely wants less.
 */
export function useNearbyUnits(
  lat: number | undefined,
  lng: number | undefined,
  radiusKm = 50,
) {
  return useQuery({
    queryKey: unitKeys.nearby(lat ?? 0, lng ?? 0, radiusKm),
    queryFn: () =>
      api.get<NearbyUnitsResponse>(`/units/nearby?lat=${lat}&lng=${lng}&radius=${radiusKm}`),
    enabled: typeof lat === 'number' && typeof lng === 'number',
    select: (data): UnitWithDistance[] => data.units,
  })
}

/** `POST /units` — the registry create path, in one place so a test can pin it. */
export const CREATE_UNIT_PATH = '/units'

/**
 * Body of `POST /units`, field for field from the `CreateUnit` input struct in
 * `backend/handlers/unit_handler.go:248-287`.
 *
 * Two things about that struct are worth stating, because they decide what a
 * form may ask for:
 *
 *  - **`commanderNin` is lower-camel with a lower-case `n`.** The Go field is
 *    `CommanderNIN`, the JSON tag is `"commanderNin"`, and the rest of the
 *    contract spells acronyms out (`registrationNumber`, `operationalRadius`).
 *    Sending `commanderNIN` binds to nothing and the value is silently lost.
 *  - **`permittedTools` and `shiftPattern` are plain strings, not nested
 *    objects.** `permittedTools` is a JSON array *encoded into a string*
 *    (`models/Unit.go:52`), so the caller serialises and the backend stores it
 *    opaquely — never send an array, which Go's decoder will reject outright.
 *
 * `hasUniform`, `weaponsRegistered` and `formationDate` are pointers on the
 * backend, so they are omitted rather than sent `false`/`""` unless the officer
 * actually answered them.
 */
export interface CreateUnitInput {
  name: string
  type: string
  state?: string
  lga?: string
  ward?: string
  city?: string
  latitude?: number
  longitude?: number
  operationalRadius?: number
  coverageArea?: string
  contactPerson?: string
  contactPhone?: string
  contactEmail?: string
  registrationNumber?: string
  /** `YYYY-MM-DD`. Omitted when unset; the backend parses it, not the browser. */
  formationDate?: string
  totalMembers?: number
  // Section B — commander
  commanderName?: string
  commanderNin?: string
  commanderPhoneAlt?: string
  commanderOccupation?: string
  commanderPriorExperience?: string
  // Section C — operational & equipment profile
  hasUniform?: boolean
  uniformDescription?: string
  shiftPattern?: string
  /** A JSON array **as a string**, e.g. `["batons","radios"]`. */
  permittedTools?: string
  weaponsRegistered?: boolean
  // Section D — traditional & local endorsement
  kindredHeadName?: string
  kindredHeadPhone?: string
  wardHeadName?: string
  wardHeadPhone?: string
}

/** The 201 from `POST /units`. The message is optional: the handler's shape varies. */
export interface CreateUnitResponse extends UnitResponse {
  message?: string
}

/**
 * The request itself, separated from the hook.
 *
 * `renderHook` is unavailable — `vite.config.ts` pins `environment: 'node'` and
 * the repo has no DOM test environment — so the mutation's `mutationFn` is the
 * seam a test can invoke directly, exactly as `useGeo.ts` exports its query
 * options for the same reason.
 */
export function createUnitRequest(input: CreateUnitInput): Promise<CreateUnitResponse> {
  return api.post<CreateUnitResponse>(CREATE_UNIT_PATH, input)
}

/**
 * Register a unit (`POST /units`).
 *
 * Not an optimistic update: the backend decides `status`, `isVerified` and
 * `verificationStatus`, and a row that is later refused by a 400 must not appear
 * in the registry in the meantime. The list is invalidated on success instead.
 *
 * Errors are *not* swallowed. A 400 carries the backend's own reason
 * (unknown state, malformed `formationDate`, refused `shiftPattern`) and the form
 * shows it, and a 403 means the session is not an admin — the two need different
 * copy, so the `ApiError` is left intact for the caller to read.
 */
export function useCreateUnit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateUnitInput) => createUnitRequest(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: unitKeys.all }),
  })
}

/** `POST /units/:id/logo` - multipart, field name "file". */
export function useUploadUnitLogo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { unitId: string; file: File }) =>
      api.upload<{ message: string; brandLogoUrl: string; hash: string; size: number }>(
        `/units/${input.unitId}/logo`,
        input.file,
      ),
    onSuccess: (_d, v) => {
      void qc.invalidateQueries({ queryKey: unitKeys.all })
      void qc.invalidateQueries({ queryKey: ['unit', v.unitId] })
    },
  })
}

/** `POST /units/:id/cover` - multipart, field name "file". */
export function useUploadUnitCover() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { unitId: string; file: File }) =>
      api.upload<{ message: string; brandCoverUrl: string; hash: string; size: number }>(
        `/units/${input.unitId}/cover`,
        input.file,
      ),
    onSuccess: (_d, v) => {
      void qc.invalidateQueries({ queryKey: unitKeys.all })
      void qc.invalidateQueries({ queryKey: ['unit', v.unitId] })
    },
  })
}

export function useDeleteUnitLogo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (unitId: string) => api.delete<{ message: string }>(`/units/${unitId}/logo`),
    onSuccess: (_d, unitId) => {
      void qc.invalidateQueries({ queryKey: unitKeys.all })
      void qc.invalidateQueries({ queryKey: ['unit', unitId] })
    },
  })
}

export function useDeleteUnitCover() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (unitId: string) => api.delete<{ message: string }>(`/units/${unitId}/cover`),
    onSuccess: (_d, unitId) => {
      void qc.invalidateQueries({ queryKey: unitKeys.all })
      void qc.invalidateQueries({ queryKey: ['unit', unitId] })
    },
  })
}
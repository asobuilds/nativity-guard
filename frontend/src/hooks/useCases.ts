import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, api } from '@/lib/apiClient'
import { postQueued } from '@/lib/apiClient'
import type {
  Case,
  CaseDetailResponse,
  CasesResponse,
  CreateCaseInput,
  CreateCaseResponse,
} from '@/types/api'

export const caseKeys = {
  all: ['cases'] as const,
  list: () => [...caseKeys.all, 'list'] as const,
  detail: (id: string) => [...caseKeys.all, 'detail', id] as const,
}

/** All cases visible to the current user (the backend scopes by role). */
export function useCases() {
  return useQuery({
    queryKey: caseKeys.list(),
    queryFn: () => api.get<CasesResponse>('/cases'),
    select: (data) => data.cases,
  })
}

/** A single case with its timeline and feedback. */
export function useCaseDetail(id: string | undefined) {
  return useQuery({
    queryKey: caseKeys.detail(id ?? ''),
    queryFn: () => api.get<CaseDetailResponse>(`/cases/${id}`),
    enabled: Boolean(id),
  })
}

type CaseActionResponse = { message: string; case: Case }

/**
 * File a new report (`POST /cases`).
 *
 * Deliberately *not* an optimistic update: the tracking id, the priority band and
 * the case's own id are all decided by the server, so there is nothing honest to
 * show before the response arrives. The receipt waits for the real one.
 *
 * On success the list is refetched — but note the flow does not depend on it:
 * the receipt renders from the mutation's own response, so a slow refetch cannot
 * make a successfully filed report look like it failed.
 */
export function useCreateCase() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (input: CreateCaseInput) => {
      // Offline-safe: if the network is down, queue the report and
      // return a synthetic "queued" case so the UI can show it saved.
      const result = await postQueued<CreateCaseResponse>('/cases', input)
      if (result && typeof result === 'object' && 'queued' in result) {
        return { queued: true, trackingId: 'QUEUED-' + Date.now().toString(36).toUpperCase() } as unknown as CreateCaseResponse
      }
      return result
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: caseKeys.list() }),
  })
}

async function runAction(
  id: string,
  action: 'dispatch' | 'arrive' | 'assign' | 'submit-review',
  body?: unknown,
): Promise<CaseActionResponse> {
  return api.post<CaseActionResponse>(`/cases/${id}/${action}`, body)
}

/**
 * Case lifecycle mutations. Each invalidates the case detail + list so the
 * stepper, timeline and queue reflect the new state.
 *
 * There is no `close`: closure became an approval on the backend, so the officer's
 * path to it is `submitForReview` (see `useReviewDecision` for the administrator's
 * half). The old mutation posted to `POST /cases/:id/close`, which is no longer
 * registered in `backend/routes/routes.go` — it could only ever 404.
 */
export function useCaseActions(id: string | undefined) {
  const queryClient = useQueryClient()

  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: caseKeys.detail(id ?? '') }),
      queryClient.invalidateQueries({ queryKey: caseKeys.list() }),
    ])
  }

  const dispatch = useMutation({
    mutationFn: () => runAction(id as string, 'dispatch'),
    onSuccess: invalidate,
  })
  const arrive = useMutation({
    mutationFn: () => runAction(id as string, 'arrive'),
    onSuccess: invalidate,
  })
  const submitForReview = useMutation({
    mutationFn: (finalReport: string) =>
      runAction(id as string, 'submit-review', { finalReport }),
    onSuccess: invalidate,
    // A 409 here means the case moved while this screen was open — the backend
    // refuses a submission from any state but `investigating` /
    // `admin_changes_requested`. Refetch so the screen stops showing a stale state.
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409) void invalidate()
    },
  })
  const assign = useMutation({
    mutationFn: (vars: { officerId: string; role?: string }) =>
      runAction(id as string, 'assign', vars),
    onSuccess: invalidate,
  })

  return { dispatch, arrive, submitForReview, assign }
}

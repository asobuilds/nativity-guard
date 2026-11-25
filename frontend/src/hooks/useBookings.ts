import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type {
  BookingListResponse,
  BookingRequestCreateInput,
  BookingResponse,
} from '@/types/api'

export const bookingKeys = {
  all: ['bookings'] as const,
  list: () => ['bookings', 'list'] as const,
  detail: (id: string) => ['bookings', 'detail', id] as const,
}

export function useMyBookings() {
  return useQuery({
    queryKey: bookingKeys.list(),
    queryFn: () => api.get<BookingListResponse>('/bookings'),
    select: (data) => data.bookings,
  })
}

export function useCreateBooking() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: BookingRequestCreateInput) =>
      api.post<BookingResponse>('/bookings', input),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: bookingKeys.list() }),
  })
}

export function useAcceptBooking() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      api.post<BookingResponse>(`/bookings/${id}/accept`, {}),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: bookingKeys.list() }),
  })
}

export function useDeclineBooking() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      api.post<BookingResponse>(`/bookings/${id}/decline`, {}),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: bookingKeys.list() }),
  })
}

export function useMarkBookingPaid() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      api.post<BookingResponse>(`/bookings/${id}/mark-paid`, {}),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: bookingKeys.list() }),
  })
}

import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import {
  CheckCircle,
  Clock,
  FileText,
  HelpCircle,
  Shield,
  XCircle,
} from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { BackLink } from '@/components/ui/BackLink'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import {
  useAcceptBooking,
  useDeclineBooking,
  useMarkBookingPaid,
  useMyBookings,
} from '@/hooks/useBookings'
import { api, ApiError } from '@/lib/apiClient'
import type { BookingRequest } from '@/types/api'

function statusTone(status: string): 'ok' | 'warn' | 'neutral' | 'emergency' {
  if (status === 'paid' || status === 'completed') return 'ok'
  if (status === 'accepted') return 'warn'
  if (status === 'declined' || status === 'cancelled') return 'emergency'
  return 'neutral'
}

function statusIcon(status: string) {
  switch (status) {
    case 'accepted':
      return <Clock className="size-4 text-warn" aria-hidden />
    case 'paid':
    case 'completed':
      return <CheckCircle className="size-4 text-ok" aria-hidden />
    case 'declined':
    case 'cancelled':
      return <XCircle className="size-4 text-emergency" aria-hidden />
    default:
      return <HelpCircle className="size-4 text-ink-faint" aria-hidden />
  }
}

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

function currency(n: number): string {
  return '₦' + (n ?? 0).toLocaleString(undefined, { maximumFractionDigits: 2 })
}

export function BookingListPage() {
  const navigate = useNavigate()
  const { notify } = useToast()
  const { user, role } = useAuth()
  const bookings = useMyBookings()

  const unitId = user?.unitId
  void unitId

  const incoming = useQuery({
    queryKey: ['my-unit-incoming-bookings'],
    queryFn: () => api.get<{ bookings: BookingRequest[] }>('/bookings'),
    enabled: role === 'unit_admin' || role === 'super_admin',
    staleTime: 30_000,
  })

  const accept = useAcceptBooking()
  const decline = useDeclineBooking()
  const markPaid = useMarkBookingPaid()

  const myBookings = useMemo(() => {
    if (role === 'unit_admin' || role === 'super_admin') {
      return incoming.data?.bookings ?? []
    }
    return bookings.data ?? []
  }, [role, incoming.data, bookings.data])

  const isUnitAdmin = role === 'unit_admin' || role === 'super_admin'

  function handleAccept(booking: BookingRequest) {
    accept.mutate(booking.id, {
      onSuccess: () => notify('Booking accepted. The client will receive the quotation.', undefined),
      onError: (err: unknown) => {
        const message = err instanceof Error ? err.message : 'Could not accept booking.'
        notify(message, 'error')
      },
    })
  }

  function handleDecline(booking: BookingRequest) {
    decline.mutate(booking.id, {
      onSuccess: () => notify('Booking declined.', undefined),
      onError: (err: unknown) => {
        const message = err instanceof Error ? err.message : 'Could not decline booking.'
        notify(message, 'error')
      },
    })
  }

  function handleMarkPaid(booking: BookingRequest) {
    markPaid.mutate(booking.id, {
      onSuccess: () => notify('Booking marked as paid.', undefined),
      onError: (err: unknown) => {
        const message = err instanceof Error ? err.message : 'Could not mark booking as paid.'
        notify(message, 'error')
      },
    })
  }

  const loading = bookings.isLoading || (isUnitAdmin && incoming.isLoading)
  const hasError = bookings.isError || (isUnitAdmin && incoming.isError)

  if (loading) {
    return (
      <main className="mx-auto w-full max-w-4xl space-y-4 p-4 sm:p-6">
        <Skeleton className="h-8 w-56" />
        {[0, 1, 2].map((i) => <Skeleton key={i} className="h-40 rounded-panel" />)}
      </main>
    )
  }

  if (hasError) {
    return (
      <main className="mx-auto w-full max-w-3xl p-4 sm:p-6">
        <BackLink to="/" label="Back" className="mb-4" />
        <Card>
          <ErrorState
            title="Could not load bookings"
            description="Try again to see your booking requests."
            offline={ApiError.isNetwork(bookings.error)}
            onRetry={() => void bookings.refetch()}
          />
        </Card>
      </main>
    )
  }

  if ((myBookings ?? []).length === 0) {
    return (
      <main className="mx-auto w-full max-w-3xl p-4 sm:p-6">
        <BackLink to="/" label="Back" className="mb-4" />
        <Card>
          <EmptyState
            icon={<FileText className="size-6" />}
            title="No bookings yet"
            description={
              isUnitAdmin
                ? 'Incoming booking requests from units will appear here.'
                : 'Request security for an event to get started.'
            }
            action={
              !isUnitAdmin ? (
                <Button onClick={() => void navigate('/bookings/new')}>Request security</Button>
              ) : undefined
            }
          />
        </Card>
      </main>
    )
  }

  return (
    <main className="mx-auto w-full max-w-4xl space-y-6 p-4 sm:p-6">
      <header className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            <Shield className="size-6 text-signal" aria-hidden />
            {isUnitAdmin ? 'Incoming bookings' : 'Your bookings'}
          </h1>
          <p className="mt-1 text-sm text-ink-muted">
            {isUnitAdmin
              ? 'Review and manage booking requests from citizens.'
              : 'View the status of your security unit booking requests.'}
          </p>
        </div>
        {!isUnitAdmin ? (
          <Button variant="primary" onClick={() => void navigate('/bookings/new')}>
            Request security
          </Button>
        ) : null}
      </header>

      <div className="grid gap-4">
        {myBookings.map((booking) => (
          <Card key={booking.id} as="article">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={statusTone(booking.status)}>
                    {statusIcon(booking.status)}
                    {booking.status}
                  </Badge>
                  <span className="text-xs text-ink-muted">{booking.eventType}</span>
                </div>
                <p className="mt-2 text-lg font-semibold text-ink">
                  {booking.eventLocation}
                </p>
                <div className="mt-1 flex flex-wrap items-center gap-3 text-sm text-ink-muted">
                  <span className="flex items-center gap-1">
                    <Clock className="size-3" aria-hidden />
                    {formatDate(booking.eventDate)}
                  </span>
                  <span>{booking.durationHours}h × {booking.guardsNeeded} guard(s)</span>
                </div>
                {booking.notes ? (
                  <p className="mt-2 text-sm text-ink-muted">{booking.notes}</p>
                ) : null}
              </div>
              <div className="shrink-0 text-right">
                <p className="text-sm font-semibold text-ink">
                  {currency(booking.quotationAmount)}
                </p>
                <p className="text-xs text-ink-muted">+ {currency(booking.platformFee)} fee</p>
                <p className="mt-1 text-sm font-semibold text-signal">
                  {currency(booking.quotationAmount + booking.platformFee)}
                </p>
              </div>
            </div>

            {booking.status === 'pending' && isUnitAdmin ? (
              <div className="mt-4 flex items-center gap-2 border-t border-border pt-3">
                <Button
                  size="sm"
                  variant="primary"
                  loading={accept.isPending}
                  onClick={() => handleAccept(booking)}
                >
                  Accept
                </Button>
                <Button
                  size="sm"
                  variant="secondary"
                  loading={decline.isPending}
                  onClick={() => handleDecline(booking)}
                >
                  Decline
                </Button>
              </div>
            ) : null}

            {booking.status === 'accepted' && isUnitAdmin ? (
              <div className="mt-4 border-t border-border pt-3">
                <Button
                  size="sm"
                  variant="primary"
                  loading={markPaid.isPending}
                  onClick={() => handleMarkPaid(booking)}
                >
                  Mark as paid
                </Button>
              </div>
            ) : null}
          </Card>
        ))}
      </div>
    </main>
  )
}

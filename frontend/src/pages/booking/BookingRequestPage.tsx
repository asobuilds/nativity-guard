import { useMemo, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Clock, ShieldCheck, Users } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { BackLink } from '@/components/ui/BackLink'
import { Button } from '@/components/ui/Button'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { useCreateBooking } from '@/hooks/useBookings'
import { useUnits } from '@/hooks/useUnits'
import { ApiError } from '@/lib/apiClient'

const DEFAULT_BASE_RATE = 25000

function currency(n: number): string {
  return '₦' + (n ?? 0).toLocaleString(undefined, { maximumFractionDigits: 2 })
}

export function BookingRequestPage() {
  const { notify } = useToast()
  const navigate = useNavigate()
  const { role } = useAuth()
  const units = useUnits()
  const create = useCreateBooking()

  const [unitId, setUnitId] = useState('')
  const [eventType, setEventType] = useState('')
  const [eventDate, setEventDate] = useState('')
  const [eventLocation, setEventLocation] = useState('')
  const [durationHours, setDurationHours] = useState('1')
  const [guardsNeeded, setGuardsNeeded] = useState('1')
  const [notes, setNotes] = useState('')

  const hours = parseFloat(durationHours) || 0
  const guards = parseInt(guardsNeeded, 10) || 0
  const baseRate = DEFAULT_BASE_RATE
  const quotationAmount = hours * guards * baseRate
  const platformFee = quotationAmount * 0.05
  const totalAmount = quotationAmount + platformFee

  const filteredUnits = useMemo(() => {
    if (role === 'unit_admin' && units.data?.[0]?.id) {
      return [units.data[0]]
    }
    return units.data ?? []
  }, [role, units.data])

  function submit(event: FormEvent) {
    event.preventDefault()
    if (!unitId || !eventType || !eventDate || !eventLocation || hours <= 0 || guards <= 0) return
    create.mutate(
      {
        unitId,
        eventType,
        eventDate,
        eventLocation,
        durationHours: hours,
        guardsNeeded: guards,
        notes: notes.trim() || undefined,
      },
      {
        onSuccess: () => {
          notify('Booking request submitted. The unit will review and respond.', undefined)
          void navigate('/bookings')
        },
        onError: (err: unknown) => {
          const message = err instanceof Error ? err.message : 'Could not submit booking request.'
          notify(message, 'error')
        },
      },
    )
  }

  if (units.isLoading) {
    return (
      <main className="mx-auto w-full max-w-2xl space-y-4 p-4 sm:p-6">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-96 rounded-panel" />
      </main>
    )
  }

  return (
    <main className="mx-auto w-full max-w-2xl space-y-6 p-4 sm:p-6">
      <BackLink to="/bookings" label="Back to bookings" />

      <header>
        <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
          <ShieldCheck className="size-6 text-signal" aria-hidden />
          Request security for an event
        </h1>
        <p className="mt-1 text-sm text-ink-muted">
          Submit a booking request to a security unit. Payment is by bank transfer — the unit
          confirms receipt once the funds arrive.
        </p>
      </header>

      {units.isError ? (
        <Card>
          <ErrorState
            title="Could not load units"
            description="Try again to select a security unit."
            offline={ApiError.isNetwork(units.error)}
            onRetry={() => void units.refetch()}
          />
        </Card>
      ) : (
        <div className="grid gap-6">
          <form onSubmit={submit} className="space-y-6">
            <Card>
              <CardHeader title="Event details" />
              <CardBody className="space-y-4">
                <Field label="Security unit" htmlFor="booking-unit" required>
                  {(props) => (
                    <Select
                      {...props}
                      value={unitId}
                      onChange={(e) => setUnitId(e.target.value)}
                      disabled={role === 'unit_admin'}
                    >
                      <option value="">Select a unit</option>
                      {filteredUnits.map((unit) => (
                        <option key={unit.id} value={unit.id}>
                          {unit.name}
                        </option>
                      ))}
                    </Select>
                  )}
                </Field>

                <Field label="Event type" htmlFor="booking-event-type" required>
                  {(props) => (
                    <Select
                      {...props}
                      value={eventType}
                      onChange={(e) => setEventType(e.target.value)}
                    >
                      <option value="">Select event type</option>
                      <option value="wedding">Wedding</option>
                      <option value="funeral">Funeral</option>
                      <option value="political_rally">Political rally</option>
                      <option value="community_gathering">Community gathering</option>
                      <option value="concert">Concert / event</option>
                      <option value="corporate">Corporate event</option>
                      <option value="religious">Religious program</option>
                      <option value="other">Other</option>
                    </Select>
                  )}
                </Field>

                <Field label="Event date" htmlFor="booking-date" required>
                  {(props) => (
                    <Input
                      {...props}
                      type="datetime-local"
                      value={eventDate}
                      onChange={(e) => setEventDate(e.target.value)}
                      min={new Date().toISOString().slice(0, 16)}
                    />
                  )}
                </Field>

                <Field label="Event location" htmlFor="booking-location" required>
                  {(props) => (
                    <Input
                      {...props}
                      value={eventLocation}
                      onChange={(e) => setEventLocation(e.target.value)}
                      placeholder="e.g. 15 Ademola Aguda Street, Victoria Island Lagos"
                    />
                  )}
                </Field>

                <div className="grid gap-4 sm:grid-cols-2">
                  <Field label="Duration (hours)" htmlFor="booking-hours" required>
                    {(props) => (
                      <Input
                        {...props}
                        type="number"
                        min={0.5}
                        step={0.5}
                        value={durationHours}
                        onChange={(e) => setDurationHours(e.target.value)}
                      />
                    )}
                  </Field>

                  <Field label="Guards needed" htmlFor="booking-guards" required>
                    {(props) => (
                      <Input
                        {...props}
                        type="number"
                        min={1}
                        step={1}
                        value={guardsNeeded}
                        onChange={(e) => setGuardsNeeded(e.target.value)}
                      />
                    )}
                  </Field>
                </div>

                <Field label="Notes (optional)" htmlFor="booking-notes">
                  {(props) => (
                    <Textarea
                      {...props}
                      value={notes}
                      onChange={(e) => setNotes(e.target.value)}
                      rows={3}
                      placeholder="Additional details about the event or security needs…"
                    />
                  )}
                </Field>
              </CardBody>
            </Card>

            <Card>
              <CardHeader title="Quotation preview" />
              <CardBody>
                {quotationAmount > 0 ? (
                  <dl className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <dt className="text-ink-muted">
                        <span className="flex items-center gap-1">
                          <Users className="size-3" aria-hidden /> Guards × hours
                        </span>
                      </dt>
                      <dd className="text-ink font-medium">
                        {guards} guard(s) × {hours}h × {currency(baseRate)}
                      </dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-ink-muted">
                        <span className="flex items-center gap-1">
                          <ShieldCheck className="size-3" aria-hidden /> Service charge
                        </span>
                      </dt>
                      <dd className="text-ink">{currency(quotationAmount)}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-ink-muted">
                        <span className="flex items-center gap-1">
                          <Clock className="size-3" aria-hidden /> Platform fee (5%)
                        </span>
                      </dt>
                      <dd className="text-ink">{currency(platformFee)}</dd>
                    </div>
                    <div className="mt-3 border-t border-border pt-2 flex justify-between">
                      <dt className="font-semibold text-ink">Total</dt>
                      <dd className="font-semibold text-signal">{currency(totalAmount)}</dd>
                    </div>
                  </dl>
                ) : (
                  <p className="text-sm text-ink-muted">
                    Enter the duration and number of guards to see the quotation.
                  </p>
                )}
              </CardBody>
            </Card>

            {create.isError ? (
              <p role="alert" className="text-sm text-emergency">
                {create.error instanceof Error ? create.error.message : 'Could not submit the booking.'}
              </p>
            ) : null}

            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={create.isPending}
              disabled={
                !unitId || !eventType || !eventDate || !eventLocation || hours <= 0 || guards <= 0
              }
            >
              Submit booking request
            </Button>
          </form>
        </div>
      )}
    </main>
  )
}

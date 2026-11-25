import { useState, type FormEvent } from 'react'
import { UserPlus, Users } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { Field, Input } from '@/components/ui/Field'
import { useAgencyObservers, useCreateAgencyObserver } from '@/hooks/useAgencyTenants'
import { validateAgencyObserverForm } from '@/lib/agency'

const EMPTY_FORM = {
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  password: '',
  rank: 'Observer',
}

/** Division-admin team management; the server resolves and enforces the tenant. */
export function AgencyTeamPanel() {
  const observers = useAgencyObservers()
  const createObserver = useCreateAgencyObserver()
  const [form, setForm] = useState(EMPTY_FORM)
  const validationError = validateAgencyObserverForm(form)
  const update = (key: keyof typeof EMPTY_FORM, value: string) =>
    setForm((current) => ({ ...current, [key]: value }))

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (validationError) return
    createObserver.mutate(
      {
        firstName: form.firstName.trim(),
        lastName: form.lastName.trim(),
        email: form.email.trim().toLowerCase(),
        phone: form.phone.trim(),
        password: form.password,
        rank: form.rank.trim() || 'Observer',
      },
      { onSuccess: () => setForm(EMPTY_FORM) },
    )
  }

  return (
    <Card>
      <CardHeader
        title="Agency team"
        subtitle="Manage observer accounts assigned to your agency division."
      />
      <CardBody>
        <form onSubmit={submit} className="grid gap-3 sm:grid-cols-2" aria-label="Create agency observer">
          <Field label="First name" required>
            {(props) => <Input {...props} value={form.firstName} onChange={(event) => update('firstName', event.target.value)} autoComplete="given-name" />}
          </Field>
          <Field label="Last name" required>
            {(props) => <Input {...props} value={form.lastName} onChange={(event) => update('lastName', event.target.value)} autoComplete="family-name" />}
          </Field>
          <Field label="Email address" required>
            {(props) => <Input {...props} type="email" value={form.email} onChange={(event) => update('email', event.target.value)} autoComplete="email" />}
          </Field>
          <Field label="Phone number" required>
            {(props) => <Input {...props} type="tel" value={form.phone} onChange={(event) => update('phone', event.target.value)} autoComplete="tel" />}
          </Field>
          <Field label="Temporary password" required hint="At least 6 characters. Share it securely and ask the new observer to change it after sign-in.">
            {(props) => <Input {...props} type="password" value={form.password} onChange={(event) => update('password', event.target.value)} autoComplete="new-password" />}
          </Field>
          <Field label="Rank">
            {(props) => <Input {...props} value={form.rank} onChange={(event) => update('rank', event.target.value)} />}
          </Field>
          {createObserver.isError ? (
            <p className="text-sm text-emergency sm:col-span-2" role="alert">
              {(createObserver.error as Error).message || 'Could not create the observer account.'}
            </p>
          ) : null}
          <div className="sm:col-span-2 flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs text-ink-muted">New accounts are restricted to this division by the server.</p>
            <Button type="submit" variant="primary" loading={createObserver.isPending} disabled={Boolean(validationError)}>
              <UserPlus className="size-4" aria-hidden /> Create observer
            </Button>
          </div>
        </form>

        <div className="my-5 border-t border-border" />

        <div className="mb-3 flex items-center justify-between gap-2">
          <h3 className="text-sm font-semibold text-ink">Division observers</h3>
          <span className="text-xs text-ink-muted">{observers.data?.observers.length ?? 0} accounts</span>
        </div>
        {observers.isLoading ? (
          <div className="space-y-2"><Skeleton className="h-16 w-full" /><Skeleton className="h-16 w-full" /></div>
        ) : observers.isError ? (
          <ErrorState title="Could not load agency team" description="The team roster is unavailable." onRetry={() => void observers.refetch()} />
        ) : observers.data?.observers.length ? (
          <ul className="divide-y divide-border">
            {observers.data.observers.map((observer) => (
              <li key={observer.id} className="flex flex-wrap items-center justify-between gap-2 py-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-ink">{observer.firstName} {observer.lastName}</p>
                  <p className="truncate text-xs text-ink-muted">{observer.email} · {observer.phone}</p>
                  <p className="text-xs text-ink-faint">{observer.rank}</p>
                </div>
                <Badge tone={observer.status === 'active' ? 'ok' : 'neutral'}>{observer.status}</Badge>
              </li>
            ))}
          </ul>
        ) : (
          <EmptyState icon={<Users className="size-5" aria-hidden />} title="No observers yet" description="Create the first observer account for this division." />
        )}
      </CardBody>
    </Card>
  )
}

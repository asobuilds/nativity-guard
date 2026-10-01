import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Bell, Briefcase, Users, Loader2, Save, UserPlus } from 'lucide-react'
import { Link } from 'react-router-dom'
import { api, mediaURL } from '@/lib/apiClient'
import { initials } from '@/lib/format'
import { AvatarUpload } from '@/components/ui/AvatarUpload'
import { CoverUpload } from '@/components/ui/CoverUpload'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Field, Input } from '@/components/ui/Field'
import { Button } from '@/components/ui/Button'
import { useToast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import type { User as UserType } from '@/types/api'
import { useProfile, useUpdateProfile } from '@/hooks/useProfile'
import { useAuth } from '@/auth/AuthContext'
import { LinkedCases } from '@/components/profile/LinkedCases'
import { normalizeCoverPosition, type CoverPosition } from '@/lib/coverPosition'

interface ProfileWithImages extends UserType {
  avatarPath?: string
  coverPath?: string
  coverPosition?: string
}

/**
 * Profile page.
 *
 * Two large images anchor the top of the page:
 *   - Cover photo, full width of the content column. Users pick the crop
 *     anchor (Top/Mid/Bot), stored server-side so it follows the account.
 *   - Profile photo, large circle overlapping the bottom of the cover.
 *
 * Neither is shown to other citizens; both are used for facial
 * identification by responding officers.
 */
export function ProfilePage() {
  const queryClient = useQueryClient()
  const { data: profile, isLoading, error } = useProfile()
  const updateProfile = useUpdateProfile()
  const { notify } = useToast()
  const { role } = useAuth()
  const [tab, setTab] = useState<'details' | 'linked'>('details')

  const [form, setForm] = useState({
    firstName: '',
    lastName: '',
    phone: '',
  })

  const [coverPosition, setCoverPosition] = useState<CoverPosition>('top')

  const profileWithImages = profile as ProfileWithImages | undefined

  // Keep local state in sync with the server value on load / refetch.
  useEffect(() => {
    setCoverPosition(normalizeCoverPosition(profileWithImages?.coverPosition))
  }, [profileWithImages?.coverPosition])

  useEffect(() => {
    if (profile) {
      setForm((prev) => ({
        ...prev,
        firstName: profile.firstName ?? '',
        lastName: profile.lastName ?? '',
        phone: profile.phone ?? '',
      }))
    }
  }, [profile])

  const positionMutation = useMutation({
    mutationFn: (position: CoverPosition) =>
      api.put<{ coverPosition: CoverPosition }>('/users/me/cover-position', { position }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['profile'] })
    },
    onError: (err) => {
      notify(
        err instanceof Error ? err.message : 'Could not save cover position',
        'error',
      )
      // Revert to whatever the server has.
      void queryClient.invalidateQueries({ queryKey: ['profile'] })
    },
  })

  function handlePositionChange(next: CoverPosition) {
    setCoverPosition(next) // optimistic — instant visual feedback
    positionMutation.mutate(next)
  }

  const handleChange = (key: keyof typeof form, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const refreshProfile = () => {
    void queryClient.invalidateQueries({ queryKey: ['profile'] })
  }

  const handleSave = async () => {
    try {
      await updateProfile.mutateAsync(form)
      notify('Profile saved', 'success')
    } catch {
      // Error is shown via toast by the mutation
    }
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <div className="flex items-center justify-center h-64">
          <Loader2 className="size-8 text-signal animate-spin" aria-hidden />
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <div className="rounded-lg border border-emergency/30 bg-emergency/10 p-4 text-emergency">
          Could not load profile. Please try again.
        </div>
      </div>
    )
  }

  const avatarUrl =
    mediaURL(profileWithImages?.avatarPath ?? profile?.photoUrl) ?? undefined
  const coverUrl = mediaURL(profileWithImages?.coverPath) ?? undefined
  const initialsValue = initials(profile?.firstName, profile?.lastName)
  const fullName = [profile?.firstName, profile?.lastName].filter(Boolean).join(' ')

  return (
    <div className="mx-auto max-w-3xl p-4 sm:p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold text-ink">Profile</h1>
        <Link to="/sessions" className="text-sm text-signal hover:underline">
          Manage signed-in devices →
        </Link>
      </div>

      <Card className="mb-4 overflow-hidden">
        <CoverUpload
          currentUrl={coverUrl}
          position={coverPosition}
          savingPosition={positionMutation.isPending}
          onUploaded={refreshProfile}
          onDeleted={refreshProfile}
          onPositionChange={handlePositionChange}
        />
        <div className="px-6 pb-6">
          <div className="-mt-16 flex flex-col gap-4 sm:-mt-20 sm:flex-row sm:items-end">
            <AvatarUpload
              currentUrl={avatarUrl}
              size={140}
              fallbackInitials={initialsValue}
              onUploaded={refreshProfile}
              onDeleted={refreshProfile}
            />
            <div className="min-w-0 flex-1 pb-2">
              <h2 className="text-xl font-bold text-ink">
                {fullName || 'Your profile'}
              </h2>
              <p className="text-sm text-ink-muted">{profile?.email}</p>
              <p className="mt-1 text-xs text-ink-faint">
                Click either photo to change it. Max 5 MB · JPEG, PNG, WebP, GIF
              </p>
            </div>
          </div>
        </div>
      </Card>

      {role === 'citizen' ? (
        <div aria-label="Profile sections" className="mb-5 flex gap-2">
          <Button
            aria-pressed={tab === 'details'}
            onClick={() => setTab('details')}
            variant={tab === 'details' ? 'primary' : 'secondary'}
          >
            Personal details
          </Button>
          <Button
            aria-pressed={tab === 'linked'}
            onClick={() => setTab('linked')}
            variant={tab === 'linked' ? 'primary' : 'secondary'}
          >
            Cases linked to me
          </Button>
        </div>
      ) : null}

      {role === 'citizen' && tab === 'linked' ? (
        <LinkedCases />
      ) : (
        <>
          <Card>
            <CardHeader title="Personal details" />
            <CardBody className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="First name">
                  {({ id, ...aria }) => (
                    <Input
                      id={id}
                      {...aria}
                      value={form.firstName}
                      onChange={(e) => handleChange('firstName', e.target.value)}
                    />
                  )}
                </Field>
                <Field label="Last name">
                  {({ id, ...aria }) => (
                    <Input
                      id={id}
                      {...aria}
                      value={form.lastName}
                      onChange={(e) => handleChange('lastName', e.target.value)}
                    />
                  )}
                </Field>
              </div>
              <Field label="Contact phone">
                {({ id, ...aria }) => (
                  <Input
                    id={id}
                    {...aria}
                    type="tel"
                    value={form.phone}
                    onChange={(e) => handleChange('phone', e.target.value)}
                  />
                )}
              </Field>

              {updateProfile.isError && (
                <p className="text-sm text-emergency" role="alert">
                  {updateProfile.error instanceof Error
                    ? updateProfile.error.message
                    : 'Failed to save profile'}
                </p>
              )}

              <Button
                variant="primary"
                icon={<Save className="size-4" aria-hidden />}
                loading={updateProfile.isPending}
                onClick={handleSave}
              >
                Save changes
              </Button>
            </CardBody>
          </Card>

          <Card className="mt-4">
            <CardHeader title="Quick links" />
            <CardBody className="flex flex-col gap-2">
              <Link
                to="/invites"
                className={cn(
                  'flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors',
                )}
              >
                <UserPlus className="size-5 text-ink-muted shrink-0" aria-hidden />
                <span className="text-sm text-ink">Invite someone</span>
              </Link>
              <Link
                to="/notifications"
                className={cn(
                  'flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors',
                )}
              >
                <Bell className="size-5 text-ink-muted shrink-0" aria-hidden />
                <span className="text-sm text-ink">Notifications</span>
              </Link>
              <Link
                to="/"
                className={cn(
                  'flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors',
                )}
              >
                <Briefcase className="size-5 text-ink-muted shrink-0" aria-hidden />
                <span className="text-sm text-ink">My cases</span>
              </Link>
              <Link
                to="/map"
                className={cn(
                  'flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors',
                )}
              >
                <Users className="size-5 text-ink-muted shrink-0" aria-hidden />
                <span className="text-sm text-ink">My units</span>
              </Link>
            </CardBody>
          </Card>
        </>
      )}
    </div>
  )
}
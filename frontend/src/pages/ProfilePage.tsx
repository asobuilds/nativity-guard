import { useState, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Bell, Briefcase, Users, Loader2, Save, UserPlus } from 'lucide-react'
import { Link } from 'react-router-dom'
import { mediaURL } from '@/lib/apiClient'
import { initials } from '@/lib/format'
import { AvatarUpload } from '@/components/ui/AvatarUpload'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Field, Input } from '@/components/ui/Field'
import { Button } from '@/components/ui/Button'
import { useToast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import type { User as UserType } from '@/types/api'
import { useProfile, useUpdateProfile } from '@/hooks/useProfile'
import { useAuth } from '@/auth/AuthContext'
import { LinkedCases } from '@/components/profile/LinkedCases'

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

	const handleChange = (key: keyof typeof form, value: string) => {
		setForm((prev) => ({ ...prev, [key]: value }))
	}

	const handleAvatarUploaded = (newPath: string) => {
		queryClient.setQueryData<UserType>(['profile'], (old) => old ? { ...old, photoUrl: newPath } : undefined)
	}

	const handleAvatarDeleted = () => {
		queryClient.setQueryData<UserType>(['profile'], (old) => old ? { ...old, photoUrl: undefined } : undefined)
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
			<div className="mx-auto max-w-xl p-6">
				<div className="flex items-center justify-center h-64">
					<Loader2 className="size-8 text-signal animate-spin" aria-hidden />
				</div>
			</div>
		)
	}

	if (error) {
		return (
			<div className="mx-auto max-w-xl p-6">
				<div className="rounded-lg border border-emergency/30 bg-emergency/10 p-4 text-emergency">
					Could not load profile. Please try again.
				</div>
			</div>
		)
	}

  const avatarUrl = mediaURL(profile?.photoUrl) ?? undefined

	return (
		<div className="mx-auto max-w-xl p-6">
			<h1 className="text-2xl font-bold text-ink mb-6">Profile</h1>
			<Link to="/sessions" className="mb-4 inline-block text-sm text-signal hover:underline">Manage signed-in devices →</Link>
			{role === 'citizen' ? <div aria-label="Profile sections" className="mb-5 flex gap-2">
				<Button aria-pressed={tab === 'details'} onClick={() => setTab('details')} variant={tab === 'details' ? 'primary' : 'secondary'}>Personal details</Button>
				<Button aria-pressed={tab === 'linked'} onClick={() => setTab('linked')} variant={tab === 'linked' ? 'primary' : 'secondary'}>Cases linked to me</Button>
			</div> : null}
			{role === 'citizen' && tab === 'linked' ? <LinkedCases /> : <>

			<Card>
				<CardHeader title="Avatar" />
				<CardBody className="flex flex-col items-center gap-4">
					<AvatarUpload
						currentUrl={avatarUrl}
						size={120}
						fallbackInitials={initials(profile?.firstName, profile?.lastName)}
						onUploaded={handleAvatarUploaded}
						onDeleted={handleAvatarDeleted}
					/>
					<p className="text-xs text-ink-muted text-center max-w-xs">
						Click the avatar to change it. Maximum 5 MB · JPEG, PNG, WebP, GIF
					</p>
				</CardBody>
			</Card>

			<Card className="mt-4">
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
							{updateProfile.error instanceof Error ? updateProfile.error.message : 'Failed to save profile'}
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
					<Link to="/invites" className={cn('flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors')}>
						<UserPlus className="size-5 text-ink-muted shrink-0" aria-hidden />
						<span className="text-sm text-ink">Invite someone</span>
					</Link>
					<Link to="/notifications" className={cn('flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors')}>
						<Bell className="size-5 text-ink-muted shrink-0" aria-hidden />
						<span className="text-sm text-ink">Notifications</span>
					</Link>
					<Link to="/" className={cn('flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors')}>
						<Briefcase className="size-5 text-ink-muted shrink-0" aria-hidden />
						<span className="text-sm text-ink">My cases</span>
					</Link>
					<Link to="/map" className={cn('flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-hi transition-colors')}>
						<Users className="size-5 text-ink-muted shrink-0" aria-hidden />
						<span className="text-sm text-ink">My units</span>
					</Link>
				</CardBody>
			</Card>
			</>}
		</div>
	)
}

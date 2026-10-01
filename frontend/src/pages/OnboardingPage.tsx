import { useRef, useState, type ChangeEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { CheckCircle2, Image as ImageIcon, Shield } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { api, ApiError } from '@/lib/apiClient'

type Step = 'welcome' | 'avatar' | 'done'

/**
 * `/onboarding` — first-run setup for a freshly registered user.
 *
 * Signup redirects here after the account is created. Every step is
 * optional: the user can skip any of them and finish at any time. Progress
 * is stored server-side via `/settings/onboarding` so the flow is
 * resumable, and completing it records `status: completed` with a
 * timestamp.
 */
export function OnboardingPage() {
  const navigate = useNavigate()
  const [step, setStep] = useState<Step>('welcome')
  const [avatarUploaded, setAvatarUploaded] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [finishing, setFinishing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  async function uploadAvatar(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return
    setUploading(true)
    setError(null)
    try {
      await api.upload('/users/me/avatar', file)
      setAvatarUploaded(true)
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.message : 'Could not upload the picture. Try again.',
      )
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  async function finish() {
    setFinishing(true)
    setError(null)
    try {
      await api.put('/settings/onboarding', { step: 'completed', status: 'completed' })
      navigate('/', { replace: true })
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause.message : 'Could not complete setup. Try again.',
      )
      setFinishing(false)
    }
  }

  return (
    <div className="min-h-screen bg-base">
      <header className="mx-auto flex max-w-2xl items-center gap-2 px-6 py-6">
        <Shield className="size-6 text-signal" aria-hidden />
        <span className="text-sm font-bold tracking-wide text-ink">NATIVITY GUARD</span>
      </header>

      <main className="mx-auto max-w-2xl px-6 pb-12">
        <ol className="mb-8 flex flex-wrap items-center gap-2">
          {['Welcome', 'Picture', 'Done'].map((label, i) => {
            const active =
              (step === 'welcome' && i === 0) ||
              (step === 'avatar' && i === 1) ||
              (step === 'done' && i === 2)
            return (
              <li key={label} className="flex items-center gap-2">
                <span
                  className={`grid size-6 place-items-center rounded-full text-xs ${
                    active ? 'bg-signal text-signal-ink' : 'bg-surface-hi text-ink-muted'
                  }`}
                >
                  {i + 1}
                </span>
                <span className="text-xs text-ink-muted">{label}</span>
                {i < 2 ? (
                  <span className="mx-1 text-ink-faint" aria-hidden>
                    →
                  </span>
                ) : null}
              </li>
            )
          })}
        </ol>

        {step === 'welcome' ? (
          <Card className="space-y-4 p-6">
            <h1 className="text-xl font-semibold text-ink">Welcome to Nativity Guard</h1>
            <p className="text-sm text-ink-muted">
              Your account is ready. Let&apos;s set up the last few things so other users and
              responding units can identify you properly.
            </p>
            <p className="text-sm text-ink-muted">
              This is quick — you can skip anything you&apos;re not ready to add, and come back
              later in Settings.
            </p>
            <div className="flex flex-wrap gap-2 pt-2">
              <Button variant="primary" onClick={() => setStep('avatar')}>
                Continue
              </Button>
              <Button onClick={() => navigate('/', { replace: true })}>Skip for now</Button>
            </div>
          </Card>
        ) : null}

        {step === 'avatar' ? (
          <Card className="space-y-4 p-6">
            <h1 className="text-xl font-semibold text-ink">Add a profile picture</h1>
            <p className="text-sm text-ink-muted">
              A clear picture helps officers and other users recognise you on the platform.
              Optional — you can add this later.
            </p>

            <div className="flex items-center gap-4">
              <div className="grid size-20 place-items-center overflow-hidden rounded-full bg-surface-hi">
                {avatarUploaded ? (
                  <CheckCircle2 className="size-8 text-ok" aria-hidden />
                ) : (
                  <ImageIcon className="size-8 text-ink-faint" aria-hidden />
                )}
              </div>
              <div className="flex flex-col gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={uploadAvatar}
                />
                <Button
                  onClick={() => fileInputRef.current?.click()}
                  loading={uploading}
                  disabled={uploading}
                >
                  {avatarUploaded ? 'Replace picture' : 'Choose a picture'}
                </Button>
                {avatarUploaded ? <p className="text-xs text-ok">Picture uploaded.</p> : null}
              </div>
            </div>

            {error ? <p className="text-xs text-warn">{error}</p> : null}

            <div className="flex flex-wrap gap-2 pt-2">
              <Button variant="primary" onClick={() => setStep('done')} disabled={uploading}>
                Continue
              </Button>
              <Button onClick={() => setStep('done')} disabled={uploading}>
                Skip this step
              </Button>
            </div>
          </Card>
        ) : null}

        {step === 'done' ? (
          <Card className="space-y-4 p-6">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="size-6 text-ok" aria-hidden />
              <h1 className="text-xl font-semibold text-ink">You&apos;re all set</h1>
            </div>
            <p className="text-sm text-ink-muted">
              You can now report incidents, follow what happens to them, and see the cases your
              community has raised.
            </p>
            <p className="text-sm text-ink-muted">
              To verify your identity with a government ID (NIN, voter&apos;s card, or
              driver&apos;s license), head to your profile when you&apos;re ready.
            </p>
            <div className="pt-2">
              <Button
                variant="primary"
                onClick={finish}
                loading={finishing}
                disabled={finishing}
              >
                Go to my dashboard
              </Button>
            </div>
            {error ? <p className="text-xs text-warn">{error}</p> : null}
          </Card>
        ) : null}
      </main>
    </div>
  )
}
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  ArrowRight,
  CheckCircle2,
  Lock,
  Shield,
  ShieldAlert,
} from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { useToast } from '@/components/ui/Toast'
import { useAuth } from '@/auth/AuthContext'
import { ApiError } from '@/lib/apiClient'
import { usePendingAction } from '@/hooks/usePendingAction'
import type { ActionContext } from './types'

export interface ActionAuthModalProps {
  open: boolean
  onClose: () => void
  action: ActionContext | null
  onSuccess?: () => void
}

export function ActionAuthModal({
  open,
  onClose,
  action,
  onSuccess,
}: ActionAuthModalProps) {
  const { login } = useAuth()
  const navigate = useNavigate()
  const { notify } = useToast()
  const { saveAction } = usePendingAction()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!action) return null
  const currentAction = action

  const isEmergency = currentAction.type === 'sos_broadcast'

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!email.trim() || !password) {
      setError('Please enter your email and password.')
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      saveAction(currentAction)
      await login(email.trim(), password)
      notify('Authentication successful.', 'success')
      onClose()
      if (onSuccess) {
        onSuccess()
      } else if (currentAction.returnUrl) {
        navigate(currentAction.returnUrl, { state: currentAction.payload })
      }
    } catch (cause) {
      if (cause instanceof ApiError) {
        setError(cause.message || 'Invalid credentials. Please try again.')
      } else {
        setError('Could not connect to authentication service.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  function handleGoToSignup() {
    saveAction(currentAction)
    onClose()
    navigate('/auth/signup', {
      state: {
        from: currentAction.returnUrl || '/',
        pendingAction: currentAction,
      },
    })
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="md"
      title={isEmergency ? 'Sign in for Emergency SOS' : currentAction.title}
    >
      <div className="space-y-4">
        {/* Context explanation banner */}
        <div
          className={`flex items-start gap-3 rounded-lg border p-3.5 text-xs ${
            isEmergency
              ? 'border-emergency/30 bg-emergency/10 text-emergency'
              : 'border-signal/30 bg-signal/10 text-ink'
          }`}
        >
          {isEmergency ? (
            <ShieldAlert className="mt-0.5 size-4 shrink-0 text-emergency" aria-hidden />
          ) : (
            <Shield className="mt-0.5 size-4 shrink-0 text-signal" aria-hidden />
          )}
          <div className="space-y-1">
            <p className="font-semibold">{currentAction.description}</p>
            <p className="text-ink-muted leading-relaxed">
              An account lets you submit reports or SOS requests and follow available status updates.
              Signing in does not send an SOS, verify an incident, or guarantee a response.
              For urgent help in Nigeria, call 112.
            </p>
          </div>
        </div>

        {currentAction.badgeText ? (
          <div className="flex items-center gap-2 rounded-md bg-surface-hi px-3 py-1.5 text-xs text-ink-muted">
            <CheckCircle2 className="size-3.5 text-ok shrink-0" aria-hidden />
            <span>{currentAction.badgeText}</span>
          </div>
        ) : null}

        <p className="text-xs leading-relaxed text-ink-muted">
          Read the <Link className="text-signal underline" to="/terms#terms" onClick={onClose}>Terms of Service</Link> and <Link className="text-signal underline" to="/terms#privacy" onClick={onClose}>Privacy Notice</Link>
          {' '}or learn <Link className="text-signal underline" to="/about" onClick={onClose}>about the team and platform</Link> before joining.
        </p>
        {/* Quick Login Form */}
        <form onSubmit={handleSubmit} className="space-y-3 pt-1">
          {error ? (
            <p className="rounded-md border border-warn/30 bg-warn/10 p-2.5 text-xs text-warn">
              {error}
            </p>
          ) : null}

          <Field label="Email address">
            {(fieldProps) => (
              <Input
                {...fieldProps}
                type="email"
                autoComplete="email"
                required
                placeholder="you@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={submitting}
              />
            )}
          </Field>

          <Field label="Password">
            {(fieldProps) => (
              <Input
                {...fieldProps}
                type="password"
                autoComplete="current-password"
                required
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={submitting}
              />
            )}
          </Field>

          <div className="pt-2 flex flex-col gap-2">
            <Button
              type="submit"
              variant={isEmergency ? 'danger' : 'primary'}
              loading={submitting}
              className="w-full justify-center"
            >
              <Lock className="size-4" aria-hidden />
              <span>Sign In & Continue</span>
            </Button>

            <div className="relative my-2 text-center text-xs text-ink-faint">
              <span className="bg-surface px-2">or new to Nativity Guard?</span>
            </div>

            <Button
              type="button"
              variant="secondary"
              onClick={handleGoToSignup}
              className="w-full justify-center"
            >
              <span>Create a Citizen Account</span>
              <ArrowRight className="size-4" aria-hidden />
            </Button>
          </div>
        </form>

        <div className="border-t border-border pt-3 text-center">
          <button
            type="button"
            onClick={onClose}
            className="text-xs text-ink-faint hover:text-ink transition-colors"
          >
            Not ready? Return to guest exploration
          </button>
        </div>
      </div>
    </Modal>
  )
}

import type { ReactNode } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/lib/cn'

interface ProgressiveSignupStepProps {
  number: 1 | 2
  title: string
  description: string
  active: boolean
  complete: boolean
  onSelect?: () => void
  children?: ReactNode
}

/** Progress marker and optional active panel for the two-stage citizen signup. */
export function ProgressiveSignupStep({
  number,
  title,
  description,
  active,
  complete,
  onSelect,
  children,
}: ProgressiveSignupStepProps) {
  return (
    <li className="min-w-0 flex-1">
      <button
        type="button"
        onClick={onSelect}
        disabled={!onSelect}
        aria-current={active ? 'step' : undefined}
        className={cn(
          'group flex w-full items-start gap-2.5 rounded-xl px-2 py-2 text-left transition-colors',
          active ? 'bg-signal/10' : 'hover:bg-surface-hi/70',
          !onSelect && 'cursor-default',
        )}
      >
        <span
          className={cn(
            'grid size-7 shrink-0 place-items-center rounded-full border text-xs font-bold transition-colors',
            complete
              ? 'border-ok bg-ok text-white'
              : active
                ? 'border-signal bg-signal text-signal-ink'
                : 'border-border-hi bg-surface-hi text-ink-faint',
          )}
          aria-hidden
        >
          {complete ? <Check className="size-4" /> : number}
        </span>
        <span className="min-w-0">
          <span className={cn('block text-xs font-semibold', active ? 'text-ink' : 'text-ink-muted')}>
            {title}
          </span>
          <span className="mt-0.5 hidden text-[10px] leading-4 text-ink-faint sm:block">{description}</span>
        </span>
      </button>
      {active && children ? <div className="mt-5">{children}</div> : null}
    </li>
  )
}

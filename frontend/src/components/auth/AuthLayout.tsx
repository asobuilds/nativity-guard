import type { ReactNode } from 'react'
import { ArrowUpRight, MapPinned, Radio, Shield, ShieldCheck } from 'lucide-react'

interface AuthLayoutProps {
  title: string
  subtitle: string
  eyebrow?: string
  children: ReactNode
}

const assuranceItems = [
  {
    icon: Radio,
    title: 'One operational record',
    detail: 'Reports, assignments and case progress stay connected from first alert to review.',
  },
  {
    icon: MapPinned,
    title: 'Built around local context',
    detail: 'Community information follows verified place and role boundaries.',
  },
  {
    icon: ShieldCheck,
    title: 'Access is accountable',
    detail: 'Protected actions are role-checked and activity is recorded.',
  },
]

export function AuthLayout({ title, subtitle, eyebrow = 'Secure workspace', children }: AuthLayoutProps) {
  return (
    <div className="relative min-h-screen overflow-hidden bg-base">
      <div
        className="pointer-events-none absolute -left-24 top-24 size-72 rounded-full bg-signal/10 blur-3xl"
        aria-hidden
      />
      <div
        className="pointer-events-none absolute -right-24 bottom-16 size-80 rounded-full bg-ok/10 blur-3xl"
        aria-hidden
      />

      <div className="relative grid min-h-screen lg:grid-cols-[minmax(0,1.08fr)_minmax(28rem,0.92fr)]">
        <aside className="ng-public-scene relative hidden overflow-hidden border-r border-border px-10 py-9 lg:flex lg:flex-col xl:px-14 xl:py-12">
          <div className="relative z-10 flex items-center gap-3">
            <span className="grid size-11 place-items-center rounded-2xl border border-signal/30 bg-signal/10 shadow-panel">
              <Shield className="size-6 text-signal" aria-hidden />
            </span>
            <div>
              <p className="text-sm font-bold tracking-[0.16em] text-ink">NATIVITY GUARD</p>
              <p className="text-xs text-ink-muted">Community safety infrastructure</p>
            </div>
          </div>

          <div className="relative z-10 my-auto max-w-xl py-12">
            <span className="inline-flex items-center gap-2 rounded-full border border-signal/25 bg-signal/10 px-3 py-1 text-xs font-semibold text-signal">
              <span className="size-1.5 animate-pulse rounded-full bg-signal" aria-hidden />
              Connected community operations
            </span>
            <h1 className="mt-5 text-4xl font-semibold leading-[1.08] tracking-tight text-ink xl:text-5xl">
              Safety work moves faster when every action has a clear record.
            </h1>
            <p className="mt-4 max-w-lg text-sm leading-6 text-ink-muted xl:text-base">
              Return to the shared workspace for citizens, responders and administrators — designed
              for accountable coordination from report to resolution.
            </p>

            <div className="mt-8 grid gap-3 xl:grid-cols-3">
              {assuranceItems.map(({ icon: Icon, title: itemTitle, detail }) => (
                <div
                  key={itemTitle}
                  className="ng-public-glass rounded-panel border border-border bg-surface/70 p-4 shadow-panel"
                >
                  <Icon className="size-5 text-signal" aria-hidden />
                  <h2 className="mt-3 text-sm font-semibold text-ink">{itemTitle}</h2>
                  <p className="mt-1 text-xs leading-5 text-ink-muted">{detail}</p>
                </div>
              ))}
            </div>
          </div>

          <div className="relative z-10 flex items-center justify-between gap-4 border-t border-border pt-5 text-xs text-ink-faint">
            <span>Authorised access only. Protected activity is audited.</span>
            <a
              href="tel:112"
              className="inline-flex shrink-0 items-center gap-1 font-semibold text-emergency hover:underline"
            >
              Emergency 112
              <ArrowUpRight className="size-3.5" aria-hidden />
            </a>
          </div>
        </aside>

        <main className="flex min-w-0 items-center justify-center px-4 py-6 sm:px-8 sm:py-10 lg:px-10">
          <div className="w-full max-w-md">
            <div className="mb-5 flex items-center justify-between rounded-2xl border border-border bg-surface/75 px-4 py-3 shadow-panel backdrop-blur lg:hidden">
              <div className="flex items-center gap-2.5">
                <span className="grid size-9 place-items-center rounded-xl bg-signal/10">
                  <Shield className="size-5 text-signal" aria-hidden />
                </span>
                <div>
                  <p className="text-xs font-bold tracking-[0.14em] text-ink">NATIVITY GUARD</p>
                  <p className="text-[11px] text-ink-muted">Secure community workspace</p>
                </div>
              </div>
              <span className="flex items-center gap-1.5 text-[11px] font-medium text-ok">
                <span className="size-1.5 animate-pulse rounded-full bg-ok" aria-hidden />
                Ready
              </span>
            </div>

            <section className="ng-public-glass rounded-[1.5rem] border border-border bg-surface/85 p-5 shadow-panel backdrop-blur-xl sm:p-7">
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-signal">{eyebrow}</p>
              <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink">{title}</h2>
              <p className="mt-2 text-sm leading-6 text-ink-muted">{subtitle}</p>
              <div className="mt-6">{children}</div>
            </section>

            <div className="mt-4 flex items-center justify-center gap-2 text-center text-[11px] text-ink-faint">
              <ShieldCheck className="size-3.5 text-ok" aria-hidden />
              Access checks apply to every protected request.
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}

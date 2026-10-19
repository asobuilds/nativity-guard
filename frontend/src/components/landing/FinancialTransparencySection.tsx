import { useState } from 'react'
import {
  Check,
  Copy,
  CreditCard,
  Heart,
  Info,
  Lock,
  ShieldCheck,
  TrendingUp,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import {
  usePlatformDonationInfo,
  usePublicSupporters,
  type PlatformDonationInfo,
} from '@/hooks/usePublicSupporters'
import type { ActionContext } from './types'

export interface FinancialTransparencySectionProps {
  onActionTrigger: (action: ActionContext) => void
}

const DEFAULT_PLATFORM_DONATION: PlatformDonationInfo = {
  bankName: 'Providus Bank',
  accountNumber: '1029482019',
  accountName: 'Nativity Guard Open Infrastructure Trust',
  purpose: 'Platform Hosting, SMS Dispatch Gateways & Map Services Only',
}

export function FinancialTransparencySection({
  onActionTrigger,
}: FinancialTransparencySectionProps) {
  const { data: supporters = [] } = usePublicSupporters()
  const { data: donationInfo } = usePlatformDonationInfo()
  const [copiedField, setCopiedField] = useState<string | null>(null)

  const activeDonation = donationInfo ?? DEFAULT_PLATFORM_DONATION

  function handleCopy(text: string, field: string) {
    navigator.clipboard.writeText(text)
    setCopiedField(field)
    setTimeout(() => setCopiedField(null), 2500)
  }

  function handleDonateClick() {
    onActionTrigger({
      type: 'community_post',
      title: 'Become a Verified Platform Supporter',
      description:
        'Help us keep Nativity Guard running for all Nigerian communities. Log in or register to record your contribution on the verified supporter ledger.',
      badgeText: 'Append-Only Ledger',
      badgeVariant: 'ok',
      returnUrl: '/about',
      payload: { target: 'platform_donation' },
    })
  }

  function formatNaira(amount: number): string {
    return new Intl.NumberFormat('en-NG', {
      style: 'currency',
      currency: 'NGN',
      maximumFractionDigits: 0,
    }).format(amount)
  }

  function formatTimeAgo(isoString: string): string {
    try {
      const date = new Date(isoString)
      const diffMs = Date.now() - date.getTime()
      const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
      if (diffDays <= 0) return 'Today'
      if (diffDays === 1) return 'Yesterday'
      return `${diffDays} days ago`
    } catch {
      return 'Recently'
    }
  }

  return (
    <section id="transparency" className="border-t border-border py-14">
      {/* Header */}
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
            <ShieldCheck className="size-3.5 text-ok" aria-hidden />
            Non-Custodial Transparency
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Financial Transparency & Public Ledger
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            Nativity Guard never touches local security donations. Units manage their own community
            bank accounts directly. Here is how our open infrastructure is maintained and audited.
          </p>
        </div>

        <div className="pt-2 md:pt-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={handleDonateClick}
            icon={<Heart className="size-3.5 text-signal" />}
          >
            Support Open Infrastructure
          </Button>
        </div>
      </div>

      {/* Non-Custodial Safeguard Banner */}
      <div className="mt-6 rounded-panel border border-ok/30 bg-ok/5 p-4 sm:p-5">
        <div className="flex items-start gap-3">
          <div className="grid size-8 place-items-center rounded-lg bg-ok/15 text-ok shrink-0">
            <Lock className="size-4" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-ink">
              Strict Non-Custodial Architecture
            </h3>
            <p className="mt-1 text-xs leading-relaxed text-ink-muted">
              Nativity Guard holds <strong>zero citizen donations</strong> on behalf of neighborhood
              patrols. When you support a security unit, your funds transfer directly from your banking
              app into the unit’s verified Nigerian commercial bank account. Nativity Guard has no payment
              cut, no middleman escrow, and no withdrawal permissions.
            </p>
          </div>
        </div>
      </div>

      <div className="mt-8 grid gap-6 lg:grid-cols-12 items-start">
        {/* Left: Platform Maintenance Bank Account Card */}
        <div className="lg:col-span-5 space-y-4">
          <Card className="p-6 border-border">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <CreditCard className="size-4 text-signal" />
                <h3 className="text-sm font-bold text-ink">Platform Hosting Trust Account</h3>
              </div>
              <span className="rounded bg-surface-hi px-2 py-0.5 text-[10px] text-ink-faint">
                Open Source Ops
              </span>
            </div>

            <p className="mt-3 text-xs leading-relaxed text-ink-muted">
              Donations to this account exclusively maintain public servers, SMS emergency dispatch
              gateways, and satellite mapping APIs for all Nigerian communities.
            </p>

            <div className="mt-4 space-y-2.5 rounded-lg bg-surface-hi p-3.5 border border-border">
              <div>
                <span className="text-[10px] uppercase font-semibold text-ink-faint">Bank Name</span>
                <p className="text-xs font-bold text-ink">{activeDonation.bankName}</p>
              </div>

              <div>
                <span className="text-[10px] uppercase font-semibold text-ink-faint">Account Name</span>
                <p className="text-xs font-bold text-ink">{activeDonation.accountName}</p>
              </div>

              <div>
                <span className="text-[10px] uppercase font-semibold text-ink-faint">Account Number</span>
                <div className="mt-0.5 flex items-center justify-between">
                  <span className="font-mono text-sm font-bold tracking-wider text-signal">
                    {activeDonation.accountNumber}
                  </span>
                  <button
                    type="button"
                    onClick={() => handleCopy(activeDonation.accountNumber, 'account')}
                    className="inline-flex items-center gap-1 rounded bg-surface px-2 py-1 text-[11px] font-medium text-ink hover:text-signal border border-border"
                  >
                    {copiedField === 'account' ? (
                      <>
                        <Check className="size-3 text-ok" />
                        <span className="text-ok">Copied</span>
                      </>
                    ) : (
                      <>
                        <Copy className="size-3" />
                        <span>Copy</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
            </div>

            <div className="mt-4 flex items-center gap-2 text-[11px] text-ink-faint">
              <Info className="size-3.5 shrink-0 text-signal" />
              <span>Reference your community or email in the bank transfer memo.</span>
            </div>
          </Card>

          {/* Append-Only Ledger Concept Card */}
          <div className="rounded-panel border border-border bg-surface p-4 text-xs">
            <div className="flex items-center gap-2">
              <TrendingUp className="size-4 text-signal" />
              <h4 className="font-bold text-ink">Append-Only Case Audit Trail</h4>
            </div>
            <p className="mt-1 text-ink-muted leading-relaxed">
              Every case state transition, officer re-assignment, and dispatch event generates an
              immutable, sequential log entry. No administrator or commander can rewrite history.
            </p>
          </div>
        </div>

        {/* Right: Public Supporter Wall */}
        <div className="lg:col-span-7">
          <Card className="p-6 border-border">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div>
                <h3 className="text-sm font-bold text-ink">Verified Community Supporters</h3>
                <p className="text-xs text-ink-muted">
                  Citizens and organizations funding the public infrastructure.
                </p>
              </div>
              <span className="rounded-full bg-signal/10 px-2.5 py-0.5 text-xs font-semibold text-signal">
                {supporters.length} Confirmed
              </span>
            </div>

            <div className="mt-4 divide-y divide-border">
              {supporters.map((item, idx) => (
                <div key={idx} className="py-3.5 first:pt-0 last:pb-0">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-xs font-bold text-ink">{item.donorName}</p>
                      {item.message && (
                        <p className="mt-1 text-xs italic text-ink-muted">
                          "{item.message}"
                        </p>
                      )}
                    </div>
                    <div className="text-right shrink-0">
                      <span className="font-mono text-xs font-bold text-ok">
                        {formatNaira(item.amount)}
                      </span>
                      <span className="mt-0.5 block text-[10px] text-ink-faint">
                        {formatTimeAgo(item.createdAt)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-6 rounded-lg bg-surface-hi p-3.5 border border-border flex flex-col sm:flex-row items-center justify-between gap-3">
              <div className="text-center sm:text-left">
                <p className="text-xs font-semibold text-ink">Want to appear on the Supporter Wall?</p>
                <p className="text-[10px] text-ink-faint">
                  Verified platform donations above ₦10,000 receive an on-chain/ledger badge.
                </p>
              </div>
              <Button
                variant="primary"
                size="sm"
                onClick={handleDonateClick}
                className="whitespace-nowrap text-xs"
              >
                Join Supporter Wall
              </Button>
            </div>
          </Card>
        </div>
      </div>
    </section>
  )
}

import { PublicSafetyMapSection } from './PublicSafetyMapSection'
import { CaseLifecycleSection } from './CaseLifecycleSection'
import { UnitsDirectorySection } from './UnitsDirectorySection'
import { SosSimulatorSection } from './SosSimulatorSection'
import { CommunityAlertsSection } from './CommunityAlertsSection'
import { AiSafetySandboxSection } from './AiSafetySandboxSection'
import { GovernanceSection } from './GovernanceSection'
import { FinancialTransparencySection } from './FinancialTransparencySection'
import type { ActionContext } from './types'
import type { LandingPanel } from './LandingNavbar'

interface LandingToolContentProps {
  panel: LandingPanel
  onActionTrigger: (action: ActionContext) => void
}

/** Shared public tools for both legacy quick-view and dedicated feature pages. */
export function LandingToolContent({ panel, onActionTrigger }: LandingToolContentProps) {
  switch (panel) {
    case 'map': return <PublicSafetyMapSection onActionTrigger={onActionTrigger} />
    case 'track': return <CaseLifecycleSection onActionTrigger={onActionTrigger} />
    case 'units': return <UnitsDirectorySection onActionTrigger={onActionTrigger} />
    case 'sos-simulator': return <SosSimulatorSection onActionTrigger={onActionTrigger} />
    case 'community-alerts': return <CommunityAlertsSection onActionTrigger={onActionTrigger} />
    case 'ai-safety': return <AiSafetySandboxSection onActionTrigger={onActionTrigger} />
    case 'governance': return <GovernanceSection onActionTrigger={onActionTrigger} />
    case 'transparency': return <FinancialTransparencySection onActionTrigger={onActionTrigger} />
  }
}

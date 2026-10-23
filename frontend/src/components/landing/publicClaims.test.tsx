import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { AiSafetySandboxSection, SAFETY_PRESETS } from './AiSafetySandboxSection'
import { GovernanceSection } from './GovernanceSection'

const { mockUsePublicBlueprint } = vi.hoisted(() => ({
  mockUsePublicBlueprint: vi.fn(),
}))

vi.mock('@/hooks/usePublicBlueprint', () => ({
  usePublicBlueprint: mockUsePublicBlueprint,
}))

describe('Public governance claims', () => {
  beforeEach(() => {
    mockUsePublicBlueprint.mockReset()
  })

  it('fails closed when public governance data is unavailable', () => {
    mockUsePublicBlueprint.mockReturnValue({ data: undefined, isPending: false, isError: true })
    const html = renderToStaticMarkup(<GovernanceSection onActionTrigger={vi.fn()} />)
    expect(html).toContain('Published governance rules could not be loaded')
    expect(html).toContain('No substitute policy or figures')
    expect(html).not.toContain('Guaranteed constitutional fair hearing')
    expect(html).not.toContain('Quorum Votes (50%)')
  })

  it('clearly distinguishes illustrative policy numbers from real votes', () => {
    mockUsePublicBlueprint.mockReturnValue({
      isError: false,
      isPending: false,
      data: {
        name: 'Nativity Guard',
        governance: {
          elections: {
            term_months: 12,
            seat_bands: { '12-20': 5, '21-40': 7, '41-70': 9, '71-100': 10 },
            quorum_percent: 50,
          },
        },
      },
    })
    const html = renderToStaticMarkup(<GovernanceSection onActionTrigger={vi.fn()} />)
    expect(html).toContain('Illustrative Council')
    expect(html).toContain('hypothetical example')
    expect(html).toContain('Example council seats')
    expect(html).toContain('Example quorum at 50%')
    expect(html).toContain('does not start a petition')
    expect(html).not.toContain('Cast Your Unit Election Ballot')
  })
})

describe('Public AI safety examples', () => {
  it('labels prewritten responses as examples, never live verified intelligence', () => {
    const html = renderToStaticMarkup(<AiSafetySandboxSection onActionTrigger={vi.fn()} />)
    expect(SAFETY_PRESETS).toHaveLength(4)
    expect(html).toContain('Static educational sample')
    expect(html).toContain('not verified live advice')
    expect(html).toContain('Real-time service availability is not guaranteed')
    expect(html).toContain('For urgent help in Nigeria, call 112')
    expect(html).not.toContain('Live Advisor Available 24/7')
    expect(html).not.toContain('Verified Community Safety Protocol')
    expect(html).not.toContain('8-digit Nativity Guard badge')
  })
})

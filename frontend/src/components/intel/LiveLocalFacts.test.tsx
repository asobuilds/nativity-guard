import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { LocalFactsList } from './LiveLocalFacts'
import type { LocalIntelFact } from '@/hooks/useLocalIntel'

const fact: LocalIntelFact = {
  id: 'intel-1',
  fact: 'Road blocked at Ikeja roundabout due to construction.',
  category: 'road_block',
  confidence: 'high',
  state: 'Lagos',
  lga: 'Ikeja',
  upvotes: 4,
  hasUpvoted: false,
  expiresAt: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
  createdAt: new Date().toISOString(),
}

describe('LiveLocalFacts list', () => {
  it('renders the fact, its area, confidence and vote count', () => {
    const html = renderToStaticMarkup(
      <LocalFactsList items={[fact]} onUpvote={() => {}} />,
    )

    expect(html).toContain('Road blocked at Ikeja roundabout')
    expect(html).toContain('Road block')
    expect(html).toContain('high confidence')
    expect(html).toContain('Ikeja, Lagos')
    expect(html).toContain('>4<')
    expect(html).toContain('aria-pressed="false"')
    expect(html).toContain('Upvote this fact')
  })

  it('disables the button when the caller already upvoted', () => {
    const html = renderToStaticMarkup(
      <LocalFactsList items={[{ ...fact, hasUpvoted: true }]} onUpvote={() => {}} />,
    )

    expect(html).toContain('aria-pressed="true"')
    expect(html).toContain('Already upvoted')
    expect(html).toContain('disabled')
  })

  it('renders nothing when there are no fresh facts', () => {
    expect(renderToStaticMarkup(<LocalFactsList items={[]} onUpvote={() => {}} />)).toBe('')
  })
})

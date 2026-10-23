import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const base = 'http://127.0.0.1:4173'
const unit = {
  id: 'browser-qa-unit-1',
  name: 'Browser QA Unit (test fixture)',
  type: 'community',
  state: 'Lagos',
  lga: 'Ikeja',
  status: 'active',
  isVerified: true,
  operationalRadius: 3,
  totalMembers: 14,
  latitude: 6.6,
  longitude: 3.35,
}
const document = kind => ({
  document: {
    id: 'qa-' + kind,
    kind,
    role: kind === 'privacy' ? 'all' : 'citizen',
    title: kind === 'privacy' ? 'QA Privacy Notice' : 'QA Terms of Service',
    version: 'qa-v1',
    effectiveAt: '2026-10-23T00:00:00Z',
    isActive: true,
    content: 'Browser-only test text for ' + kind + '. Not a real legal document.',
  },
  version: 'qa-v1',
})

const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] })
const context = await browser.newContext({
  viewport: { width: 1365, height: 850 },
  serviceWorkers: 'block',
})
let unitApiFails = false
await context.route('**/api/v1/**', async route => {
  const requestUrl = new URL(route.request().url())
  const path = requestUrl.pathname
  const respond = (status, body) => route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
    headers: { 'Access-Control-Allow-Origin': '*' },
  })
  if (path.endsWith('/public/units/browser-qa-unit-1/bank-accounts')) {
    return respond(200, { accounts: [] })
  }
  if (path.endsWith('/public/units')) {
    return unitApiFails ? respond(503, { error: 'Service unavailable' }) : respond(200, { units: [unit] })
  }
  if (path.endsWith('/public/cases')) return respond(200, { cases: [] })
  if (path.endsWith('/geo/states')) {
    return respond(200, { states: [{ name: 'Lagos', capital: 'Ikeja', zone: 'South West', lgaCount: 20 }] })
  }
  if (path.endsWith('/geo/lgas')) return respond(200, { state: 'Lagos', lgas: ['Ikeja'] })
  if (path.endsWith('/public/platform/supporters')) return respond(200, { supporters: [] })
  if (path.endsWith('/public/platform/donation-info')) return respond(200, { donationInfo: null })
  if (path.endsWith('/public/blueprint')) {
    return respond(200, {
      name: 'Nativity Guard',
      governance: {
        elections: {
          term_months: 12, stagger_months: 6, term_limit: 2, cooling_off_months: 6,
          seat_bands: { '12-20': 5, '21-40': 7, '41-70': 9, '71-100': 10 }, quorum_percent: 50,
        },
      },
    })
  }
  if (path.endsWith('/terms/citizen')) {
    return respond(200, document(requestUrl.searchParams.get('kind') === 'privacy' ? 'privacy' : 'terms'))
  }
  return respond(404, { error: 'Not configured in isolated browser QA' })
})

const page = await context.newPage()
const checks = []
const check = async (name, run) => {
  await run()
  checks.push(name)
  console.log('PASS:', name)
}
const landing = async (hash = '') => {
  await page.goto(base + '/' + hash, { waitUntil: 'domcontentloaded' })
  await page.locator('.ng-ref-headline').waitFor({ state: 'visible', timeout: 20000 })
}
const modal = name => page.getByRole('dialog', { name })

try {
  await check('Desktop hero renders all three primary actions', async () => {
    await landing()
    assert.equal(await page.locator('.ng-ref-action').count(), 3)
    assert.equal(await page.getByRole('button', { name: /Emergency SOS/i }).count() > 0, true)
  })

  await check('Map dialog Escape closes and removes the hash', async () => {
    await page.getByRole('button', { name: /Explore Safety Map/ }).click()
    await modal('Safety Map').waitFor({ state: 'visible', timeout: 12000 })
    assert.equal(new URL(page.url()).hash, '#map')
    await page.keyboard.press('Escape')
    await page.locator('.ng-ref-modal-window').waitFor({ state: 'detached' })
    assert.equal(new URL(page.url()).hash, '')
  })

  await check('Eight public feature URLs open a named dialog and close', async () => {
    const variants = [
      ['map', 'Safety Map'],
      ['case-lifecycle', 'Track a Case'],
      ['units', 'Security Units'],
      ['sos-simulator', 'Emergency SOS Guidance'],
      ['community-alerts', 'Community Alerts'],
      ['ai-safety', 'AI Safety Advisor'],
      ['governance', 'Governance'],
      ['transparency', 'Financial Transparency'],
    ]
    for (const [hash, title] of variants) {
      await landing('#' + hash)
      await modal(title).waitFor({ state: 'visible', timeout: 15000 })
      await page.keyboard.press('Escape')
      await page.locator('.ng-ref-modal-window').waitFor({ state: 'detached' })
    }
  })

  await check('Nested donation dialog closes child only and restores parent', async () => {
    await landing('#units')
    await modal('Security Units').waitFor({ state: 'visible', timeout: 15000 })
    const donate = page.getByRole('button', { name: 'View donation details' })
    await donate.waitFor({ state: 'visible', timeout: 12000 })
    await donate.click()
    await modal('Support Browser QA Unit (test fixture)').waitFor({ state: 'visible' })
    assert.equal(await page.locator('[role="dialog"]').count(), 2)
    await page.keyboard.press('Escape')
    await modal('Support Browser QA Unit (test fixture)').waitFor({ state: 'detached' })
    await modal('Security Units').waitFor({ state: 'visible' })
    assert.equal(await page.locator('[role="dialog"]').count(), 1)
    await page.keyboard.press('Escape')
    await page.locator('.ng-ref-modal-window').waitFor({ state: 'detached' })
    assert.notEqual(await page.evaluate(() => document.body.style.overflow), 'hidden')
  })

  await check('Public Terms and Privacy documents render from stubbed API', async () => {
    await page.goto(base + '/terms', { waitUntil: 'domcontentloaded' })
    await page.getByText('Browser-only test text for terms.', { exact: false }).waitFor({ timeout: 12000 })
    await page.getByText('Browser-only test text for privacy.', { exact: false }).waitFor({ timeout: 12000 })
  })

  await check('Missing live units do not create fictional verified patrols', async () => {
    unitApiFails = true
    await landing('#units')
    await page.getByText('Unable to load units from the service', { exact: false }).waitFor({ timeout: 20000 })
    assert.equal(await page.getByText('Browser QA Unit (test fixture)', { exact: true }).count(), 0)
    unitApiFails = false
  })

  await check('Mobile navigation opens, closes and reaches tracking dialog', async () => {
    await page.setViewportSize({ width: 375, height: 812 })
    await landing()
    await page.getByRole('button', { name: 'Open navigation menu' }).click()
    await modal('Navigation menu').waitFor({ state: 'visible' })
    await modal('Navigation menu').getByRole('button', { name: 'Track Case' }).click()
    await modal('Track a Case').waitFor({ state: 'visible' })
    assert.equal(await modal('Navigation menu').count(), 0)
    await page.keyboard.press('Escape')
    await page.locator('.ng-ref-modal-window').waitFor({ state: 'detached' })
  })

  console.log(`Browser smoke checks passed: ${checks.length}`)
} catch (error) {
  console.error('Browser smoke failure after:', checks.join(', '))
  console.error(error)
  await page.screenshot({ path: 'landing-browser-smoke-failure.png', fullPage: true }).catch(() => {})
  process.exitCode = 1
} finally {
  await browser.close()
}

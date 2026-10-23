import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const origin = process.env.NG_LIVE_ORIGIN || 'https://nativityguard.com'
const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.PLAYWRIGHT_BROWSER_EXECUTABLE || undefined,
  args: ['--no-sandbox'],
})
const context = await browser.newContext({
  viewport: { width: 1365, height: 850 },
  serviceWorkers: 'block',
})
const page = await context.newPage()
const observedPolicyRequests = []
page.on('request', request => {
  if (request.url().includes('/terms/citizen')) {
    const url = new URL(request.url())
    observedPolicyRequests.push({ phase: 'started', host: url.host, path: url.pathname, kind: url.searchParams.get('kind') || 'terms' })
  }
})
page.on('response', response => {
  if (response.url().includes('/terms/citizen')) {
    observedPolicyRequests.push({ status: response.status(), url: new URL(response.url()).pathname })
  }
})
page.on('requestfailed', request => {
  if (request.url().includes('/terms/citizen')) {
    observedPolicyRequests.push({ url: new URL(request.url()).pathname, failure: request.failure()?.errorText })
  }
})
const checked = []
async function verify(name, action) {
  await action()
  checked.push(name)
  console.log('LIVE PASS:', name)
}

try {
  await verify('Published homepage renders City Gate hero', async () => {
    const response = await page.goto(origin + '/', { waitUntil: 'domcontentloaded', timeout: 35000 })
    assert.ok(response && response.ok(), 'Homepage did not return a successful HTTP status')
    await page.locator('.ng-ref-headline').waitFor({ state: 'visible', timeout: 30000 })
    assert.equal(await page.locator('.ng-ref-action').count(), 3)
  })

  await verify('Published map quick-view opens and Escape closes it', async () => {
    await page.getByRole('button', { name: /Explore Safety Map/ }).click()
    await page.getByRole('dialog', { name: 'Safety Map' }).waitFor({ state: 'visible', timeout: 15000 })
    await page.keyboard.press('Escape')
    await page.locator('.ng-ref-modal-window').waitFor({ state: 'detached', timeout: 15000 })
  })

  await verify('Public About page renders without sign-in', async () => {
    const response = await page.goto(origin + '/about', { waitUntil: 'domcontentloaded', timeout: 35000 })
    assert.ok(response && response.ok(), 'About page did not return a successful HTTP status')
    await page.getByRole('heading', { name: /About Nativity Guard/i }).first().waitFor({ state: 'visible', timeout: 30000 })
  })

  await verify('Public Terms and Privacy page renders without sign-in', async () => {
    const response = await page.goto(origin + '/terms', { waitUntil: 'domcontentloaded', timeout: 35000 })
    assert.ok(response && response.ok(), 'Terms page did not return a successful HTTP status')
    await page.getByRole('heading', { name: /Terms.*Privacy/i }).first().waitFor({ state: 'visible', timeout: 30000 })
    await page.locator('main h2').first().waitFor({ state: 'visible', timeout: 20000 })
    const headings = await page.locator('main h2').allTextContents()
    assert.ok(headings.length >= 2, 'Expected both Terms and Privacy document sections')
    const content = await page.locator('main').innerText()
    assert.ok(!content.includes('Could not load the terms.'), 'Terms backend reported a document loading problem')
    assert.ok(content.includes('Version'), 'No versioned published legal document was displayed')
    assert.ok(content.length > 1000, 'Published legal documents appear empty')
  })

  console.log(`Live read-only checks passed: ${checked.length}`)
} catch (error) {
  console.error('LIVE READ-ONLY CHECK FAILED after:', checked.join(', '))
  console.error('Observed policy requests:', JSON.stringify(observedPolicyRequests))
  console.error('Rendered public page:', (await page.locator('main').innerText().catch(() => 'No main found')).slice(0, 700))
  console.error(error)
  await page.screenshot({ path: 'landing-live-readonly-failure.png', fullPage: true }).catch(() => {})
  process.exitCode = 1
} finally {
  await browser.close()
}

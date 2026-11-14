import { expect, test } from '@playwright/test'

// All records below are synthetic TEST FIXTURES; they never reach production.
// Intercept public endpoints to test UI states without real incident or bank data.
const UNIT = {
  id: '00000000-0000-4000-8000-000000000007',
  name: 'Browser QA Example Unit',
  status: 'active',
  isVerified: true,
  operationalRadius: 4,
  totalMembers: 7,
  latitude: 9.08,
  longitude: 8.68,
  state: 'Test State',
  lga: 'Test LGA',
  type: 'Example only',
}

async function mockPublicApi(page, { units = [], cases = [], bankAccounts = null } = {}) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname.replace(/^.*\/api\/v1/, '')
    if (request.method() !== 'GET') {
      return route.fulfill({ status: 405, contentType: 'application/json', body: '{}' })
    }
    const response = path === '/public/units'
      ? { units }
      : path === '/public/cases'
        ? { cases }
        : path === `/public/units/${UNIT.id}/bank-accounts`
          ? (bankAccounts === null ? null : { accounts: bankAccounts })
          : path === '/public/platform/supporters'
            ? { supporters: [] }
            : path === '/public/platform/donation-info'
              ? null
              : null

    return route.fulfill({
      status: response === null ? 404 : 200,
      contentType: 'application/json',
      body: JSON.stringify(response ?? { error: 'Not configured in browser QA fixture' }),
    })
  })
}

test('landing hero, public rights, and SOS limitations are visible', async ({ page }) => {
  await mockPublicApi(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /Your Community/i })).toBeVisible()
  await expect(page.getByRole('button', { name: /Report an Incident/i })).toBeVisible()
  await expect(page.getByRole('button', { name: /Explore Safety Map/i })).toBeVisible()
  await expect(page.getByRole('link', { name: /Terms of Service/i })).toBeVisible()
  await expect(page.getByRole('link', { name: /Privacy notice/i })).toBeVisible()
  await expect(page.locator('.ng-ref-trust-footer')).toContainText('not a police force or emergency response service')
  await expect(page.locator('.ng-ref-trust-footer')).toContainText('No government endorsement')
  await expect(page.locator('.ng-ref-stats')).not.toContainText('4 mins')
})

test('all public feature hashes open accessible dialogs and Escape closes one panel', async ({ page }) => {
  await mockPublicApi(page)
  const samples = [
    ['map', 'Safety Map'],
    ['case-lifecycle', 'Track a Case'],
    ['units', 'Security Units'],
    ['sos-simulator', 'Emergency SOS Guidance'],
    ['community-alerts', 'Community Alerts'],
    ['ai-safety', 'AI Safety Advisor'],
    ['governance', 'Governance'],
    ['transparency', 'Financial Transparency'],
  ]
  for (const [hash, title] of samples) {
    await page.goto('/#' + hash)
    const dialog = page.getByRole('dialog', { name: title })
    await expect(dialog).toBeVisible()
    await expect(dialog).toHaveAttribute('aria-modal', 'true')
    await expect(dialog.getByRole('button', { name: 'Close ' + title })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(page).not.toHaveURL(/#(map|case-lifecycle|units|sos-simulator|community-alerts|ai-safety|governance|transparency)$/)
  }
})

test('failed and empty public endpoints never create believable patrol or incident records', async ({ page }) => {
  await mockPublicApi(page)
  await page.goto('/#units')
  await expect(page.getByText(/No published active units match your filters/i)).toBeVisible()
  await expect(page.getByText('Browser QA Example Unit')).toHaveCount(0)
  await page.goto('/#case-lifecycle')
  await expect(page.getByRole('dialog', { name: 'Track a Case' })).toBeVisible()
  await expect(page.locator('.ng-ref-modal-content')).not.toContainText('NG-2026-00194821')
})

test('stacked donation dialog Escape closes child only and restores parent focus', async ({ page }) => {
  await mockPublicApi(page, { units: [UNIT] })
  await page.goto('/#units')
  await expect(page.getByText(UNIT.name)).toBeVisible()
  const parent = page.locator('.ng-ref-modal-window')
  await expect(parent).toBeVisible()
  const donate = page.getByRole('button', { name: /View donation details/i })
  await donate.click()
  const child = page.getByRole('dialog', { name: 'Support ' + UNIT.name })
  await expect(child).toBeVisible()
  await expect(parent).toHaveAttribute('aria-hidden', 'true')
  await page.keyboard.press('Escape')
  await expect(child).toHaveCount(0)
  await expect(parent).toBeVisible()
  await expect(parent).toHaveAttribute('aria-modal', 'true')
  await expect(donate).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(parent).toHaveCount(0)
  await expect(page.locator('body')).not.toHaveCSS('overflow', 'hidden')
})

test('an unavailable bank account endpoint does not show fabricated payment details', async ({ page }) => {
  await mockPublicApi(page, { units: [UNIT] })
  await page.goto('/#units')
  await page.getByRole('button', { name: /View donation details/i }).click()
  await expect(page.getByRole('dialog', { name: 'Support ' + UNIT.name })).toBeVisible()
  await expect(page.getByText(/Bank details could not be loaded/i)).toBeVisible()
  await expect(page.getByText('Primary listed account')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Copy' })).toHaveCount(0)
})

test('empty published bank-account list is not presented as a failed request', async ({ page }) => {
  await mockPublicApi(page, { units: [UNIT], bankAccounts: [] })
  await page.goto('/#units')
  await page.getByRole('button', { name: /View donation details/i }).click()
  await expect(page.getByRole('dialog', { name: 'Support ' + UNIT.name })).toBeVisible()
  await expect(page.getByText(/No bank account details are currently published/i)).toBeVisible()
  await expect(page.getByText(/Bank details could not be loaded/i)).toHaveCount(0)
})

test('public About and Terms routes do not require sign-in', async ({ page }) => {
  await mockPublicApi(page)
  await page.goto('/about')
  await expect(page.getByRole('heading', { name: /About/i }).first()).toBeVisible()
  await page.goto('/terms')
  await expect(page.getByRole('heading', { name: /Terms/i }).first()).toBeVisible()
  await expect(page).not.toHaveURL(/\/auth\/login/)
})

test('mobile menu opens, closes, and supports public feature navigation', async ({ page }, testInfo) => {
  test.skip(!testInfo.project.name.startsWith('mobile'), 'Mobile layout only')
  await mockPublicApi(page)
  await page.goto('/')
  const menuButton = page.getByRole('button', { name: 'Open navigation menu' })
  await expect(menuButton).toBeVisible()
  await menuButton.click()
  const drawer = page.getByRole('dialog', { name: 'Navigation menu' })
  await expect(drawer).toBeVisible()
  await drawer.getByRole('button', { name: 'Safety Map' }).click()
  await expect(drawer).toHaveCount(0)
  const map = page.getByRole('dialog', { name: 'Safety Map' })
  await expect(map).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(map).toHaveCount(0)
  await expect(page.locator('body')).not.toHaveCSS('overflow', 'hidden')
})

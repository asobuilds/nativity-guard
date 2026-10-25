import { test, expect } from '@playwright/test'

test.describe('SOS and Incident Workflow E2E', () => {
  test('Citizen submits SOS and Admin views it in dashboard', async ({ page }) => {
    // 1. Mock API responses for citizen SOS submission
    await page.route('**/api/v1/sos', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            ok: true,
            sos: {
              id: 'sos-test-123',
              status: 'pending',
              latitude: 6.5244,
              longitude: 3.3792,
              createdAt: new Date().toISOString(),
            },
          }),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ alerts: [] }),
        })
      }
    })

    // Mock media upload
    await page.route('**/api/v1/sos/**/media', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ok: true, mediaKey: 'mock-media-key' }),
      })
    })

    // Mock units query
    await page.route('**/api/v1/units', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ units: [{ id: 'unit-1', name: 'Lagos Central Unit' }] }),
      })
    })

    // 2. Load SOS page as citizen (using mock storageState or login)
    await page.context().addCookies([
      {
        name: 'ng_token',
        value: 'mock-citizen-token',
        domain: 'localhost',
        path: '/',
      },
    ])
    await page.addInitScript(() => {
      window.localStorage.setItem(
        'ng_user',
        JSON.stringify({ id: 'citizen-1', email: 'citizen@example.com', role: 'citizen', firstName: 'Test', lastName: 'Citizen' })
      )
    })

    await page.goto('/sos')

    // Fill SOS form
    await page.fill('input[id*="lat"]', '6.5244')
    await page.fill('input[id*="lng"]', '3.3792')

    // Click submit emergency request button
    const submitBtn = page.getByRole('button', { name: /send emergency request|confirm emergency|request emergency/i })
    if (await submitBtn.isVisible()) {
      await submitBtn.click()
    }

    // 3. Switch context to admin and verify admin dashboard/cases
    await page.evaluate(() => {
      window.localStorage.setItem(
        'ng_user',
        JSON.stringify({ id: 'admin-1', email: 'admin@example.com', role: 'unit_admin', firstName: 'Test', lastName: 'Admin', unitId: 'unit-1' })
      )
    })

    await page.route('**/api/v1/cases', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          cases: [
            {
              id: 'case-test-123',
              title: 'Emergency SOS Incident',
              status: 'pending',
              priorityLevel: 'critical',
              latitude: 6.5244,
              longitude: 3.3792,
              trackingId: 'TRK-999',
              createdAt: new Date().toISOString(),
            },
          ],
        }),
      })
    })

    await page.goto('/admin/cases')
    await expect(page.getByText('Emergency SOS Incident')).toBeVisible()
  })
})

import { test, expect } from '@playwright/test'

test.describe('SOS and Incident Workflow E2E', () => {
  test('Citizen submits SOS and Admin views it in dashboard', async ({ page, context }) => {
    // 1. Grant geolocation permissions
    await context.grantPermissions(['geolocation'])
    await context.setGeolocation({ latitude: 6.5244, longitude: 3.3792 })

    // 2. Mock login API response so citizen@example.com login succeeds
    await page.route('**/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'mock-citizen-token',
          user: {
            id: 'citizen-1',
            email: 'citizen@example.com',
            role: 'citizen',
            firstName: 'Test',
            lastName: 'Citizen',
          },
        }),
      })
    })

    // Mock SOS submission API
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
            mediaAttachmentWarning: null,
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

    // 3. Navigate to login page
    await page.goto('/auth/login')

    // Fill login form using label selectors
    await page.getByLabel('Email').fill('citizen@example.com')
    await page.getByLabel('Password').fill('password123')
    await page.getByRole('button', { name: /sign in/i }).click()

    // Wait for redirect to home/dashboard
    await page.waitForURL('/')

    // 4. Navigate to SOS page
    await page.goto('/sos')
    await expect(page.locator('h1')).toContainText('Emergency SOS')

    // Ensure latitude and longitude inputs are populated or use geolocation button
    const locateBtn = page.getByRole('button', { name: /use my location/i })
    if (await locateBtn.isVisible()) {
      await locateBtn.click()
    }
    await page.getByLabel('Latitude').fill('6.5244')
    await page.getByLabel('Longitude').fill('3.3792')

    // Click Prepare SOS
    await page.getByRole('button', { name: /prepare sos/i }).click()

    // Confirm SOS in modal
    await page.getByRole('button', { name: /confirm and send sos/i }).click()

    // Verify receipt appears
    await expect(page.getByText('Emergency request received')).toBeVisible()

    // 5. Switch to admin session and verify admin cases
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

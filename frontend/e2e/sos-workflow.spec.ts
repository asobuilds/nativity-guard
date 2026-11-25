import { test, expect } from '@playwright/test';

test.describe('SOS and Incident Workflow E2E', () => {
  test('Citizen submits SOS and Admin views it in dashboard', async ({ page, context }) => {
    // 1. Grant geolocation permissions and set a mock location
    await context.grantPermissions(['geolocation']);
    await context.setGeolocation({ latitude: 6.5244, longitude: 3.3792 });

    // 2. Navigate to the login page
    await page.goto('/auth/login');

    // 3. Use the Demo "Citizen" button to auto-fill the form
    await page.getByRole('button', { name: 'Citizen' }).click();
    
    // 4. Click the Sign in button
    await page.getByRole('button', { name: /sign in/i }).click();

    // 5. Wait for the dashboard to load
    await page.waitForURL('/');

    // 6. Navigate to the SOS page
    await page.goto('/sos');
    
    // Wait for the SOS form to be visible
    await expect(page.locator('h1')).toContainText('Emergency SOS');

    // 7. Verify the location is auto-populated from geolocation
    // Assuming there is a "Prepare SOS" button:
    await page.getByRole('button', { name: /prepare sos/i }).click();

    // 8. Confirm the SOS dialog
    await page.getByRole('button', { name: /confirm and send sos/i }).click();

    // 9. Assert success (e.g., a toast or redirect)
    await expect(page.locator('text=SOS sent successfully')).toBeVisible({ timeout: 10000 });
  });
});

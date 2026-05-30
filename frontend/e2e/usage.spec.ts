import { test, expect } from "@playwright/test";

test.describe("Usage page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/usage");
    // Wait for the page shell to render.
    await expect(
      page.locator(".usage-page"),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("shows toolbar and summary cards with data", async ({
    page,
  }) => {
    await expect(
      page.locator(".usage-toolbar").first(),
    ).toBeVisible();

    // Summary cards should appear with at least one value.
    await expect(
      page.locator(".summary-cards"),
    ).toBeVisible();
    await expect(
      page.locator(".card-value").first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("shows cost time series chart", async ({ page }) => {
    // Wait for summary data to load.
    await expect(
      page.locator(".summary-cards .card-value").first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      page.locator(".chart-container"),
    ).toBeVisible();
    // SVG chart should render.
    await expect(
      page.locator(".chart-container svg"),
    ).toBeVisible();
  });

  test("shows attribution panel with treemap", async ({
    page,
  }) => {
    await expect(
      page.locator(".summary-cards .card-value").first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      page.locator(".attribution-panel"),
    ).toBeVisible();
    // Treemap SVG should be rendered.
    await expect(
      page.locator(".treemap-container svg"),
    ).toBeVisible();
  });

  test("filter dropdown opens and shows items", async ({
    page,
  }) => {
    // Wait for data so filter items are populated.
    await expect(
      page.locator(".summary-cards .card-value").first(),
    ).toBeVisible({ timeout: 10_000 });

    // Click the first filter dropdown (Project).
    const trigger = page
      .locator(".filter-dropdown .filter-trigger")
      .first();
    await trigger.click();

    // Dropdown panel should appear with rows.
    await expect(
      page.locator(".dropdown-panel").first(),
    ).toBeVisible();
    await expect(
      page.locator(".dropdown-row").first(),
    ).toBeVisible();
  });

  test("excluding a project updates total cost", async ({
    page,
  }) => {
    // Wait for data to load.
    await expect(
      page.locator(".summary-cards .card-value").first(),
    ).toBeVisible({ timeout: 10_000 });

    // Grab the initial total cost text.
    const totalCostBefore = await page
      .locator(".card.featured .card-value")
      .textContent();

    // Open the project filter and exclude the first item.
    const trigger = page
      .locator(".filter-dropdown .filter-trigger")
      .first();
    await trigger.click();
    await page
      .locator(".dropdown-row")
      .filter({ hasText: "project-delta" })
      .first()
      .click();

    // Close dropdown by clicking outside the menu.
    await page.mouse.click(10, 10);

    // Total cost should change after refetch.
    await expect(async () => {
      const after = await page
        .locator(".card.featured .card-value")
        .textContent();
      expect(after).not.toBe(totalCostBefore);
    }).toPass({ timeout: 5_000 });
  });

  test("select all / deselect all buttons work", async ({
    page,
  }) => {
    // Wait for data so items populate.
    await expect(
      page.locator(".summary-cards .card-value").first(),
    ).toBeVisible({ timeout: 10_000 });

    // Open the project filter.
    const trigger = page
      .locator(".filter-dropdown .filter-trigger")
      .first();
    await trigger.click();

    // Click "Deselect all".
    await page
      .locator(".bulk-btn")
      .filter({ hasText: "Deselect all" })
      .first()
      .click();

    // Trigger label should show "None".
    await expect(trigger).toContainText("None");

    // Click "Select all".
    await page
      .locator(".bulk-btn")
      .filter({ hasText: "Select all" })
      .first()
      .click();

    // Trigger label should show "All".
    await expect(trigger).toContainText("All");
  });

  test("top nav shows Usage button as active", async ({
    page,
  }) => {
    const usageBtn = page.locator(
      '.nav-btn[aria-label="Usage"]',
    );
    await expect(usageBtn).toBeVisible();
    await expect(usageBtn).toHaveClass(/active/);
  });

  test("preserves explicit one-day window URL", async ({
    page,
  }) => {
    await page.goto("/usage?window_days=1");
    await expect(
      page.locator(".usage-page"),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page).toHaveURL(/\/usage\?window_days=1$/);
  });

  test("preserves pinned date range across a tab switch", async ({
    page,
  }) => {
    // Regression guard for commit 8c733be. Pinning a date range and then
    // leaving and re-entering /usage via the in-app nav (a bare /usage
    // navigation with no URL params) must PRESERVE the pin. The pre-fix
    // bug reset isPinned on bare navigation, rolling the range back to the
    // default rolling window and dropping from/to from the URL.

    // Deep-link a pinned date range; the write-back keeps from/to in the URL.
    await page.goto("/usage?from=2026-01-01&to=2026-01-31");
    await expect(
      page.locator(".usage-page"),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page).toHaveURL(/from=2026-01-01/);
    await expect(page).toHaveURL(/to=2026-01-31/);

    // Switch to another tab via client-side nav (no full reload), so the
    // in-memory usage store keeps isPinned while UsagePage unmounts.
    await page
      .locator('.nav-btn[aria-label="Sessions"]')
      .click();
    await expect(page).toHaveURL(/\/sessions/);

    // Switch back via the Usage nav button: router.navigate("usage") pushes
    // a bare /usage (from/to are not sticky params), re-running URL-init
    // with empty params.
    await page
      .locator('.nav-btn[aria-label="Usage"]')
      .click();
    await expect(
      page.locator(".usage-page"),
    ).toBeVisible({ timeout: 10_000 });

    // The pin survived: isPinned stayed true, so the write-back re-adds
    // from/to to the URL. With the pre-fix bug these params would be gone.
    await expect(page).toHaveURL(/from=2026-01-01/, {
      timeout: 10_000,
    });
    await expect(page).toHaveURL(/to=2026-01-31/, {
      timeout: 10_000,
    });
  });

  test("URL updates when filter changes", async ({ page }) => {
    // Wait for data.
    await expect(
      page.locator(".summary-cards .card-value").first(),
    ).toBeVisible({ timeout: 10_000 });

    // Exclude a project.
    const trigger = page
      .locator(".filter-dropdown .filter-trigger")
      .first();
    await trigger.click();
    await page
      .locator(".dropdown-row")
      .filter({ hasText: "project-delta" })
      .first()
      .click();
    await page.mouse.click(10, 10);

    // URL should contain the exclude_project param.
    await expect(page).toHaveURL(/exclude_project=/);
  });
});

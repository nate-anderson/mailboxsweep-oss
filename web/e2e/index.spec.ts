import { test, expect } from '@playwright/test';

test.describe('index page sidebar', () => {
    
    test('sidebar is not initially visible, becomes visible when clicking menu', async ({ page }) => {
        await page.goto('/');
        const menu = await page.locator("#menu-drawer");
        await expect(menu).toBeHidden();

        await page.locator('#sidebar-menu-open').click();
        await expect(menu).toBeVisible();

        await page.locator('#sidebar-menu-close').click();
        await expect(menu).toBeHidden()
    });

    test('sidebar links scroll to appropriate sections', async ({ page }) => {
        // test IDs
        const buttonToHeaderMap = {
            "menu-drawer-item-0": "header-how-it-works",
            "menu-drawer-item-1": "header-plans",
        }

        await page.goto('/');

        for (const [link, scrollElement] of Object.entries(buttonToHeaderMap)) {
            await page.locator('#sidebar-menu-open').click();
            const button = await page.getByTestId(link);
            await button.click();
            await expect(await page.getByTestId(scrollElement)).toBeInViewport()
        }

    });

    test('sidebar navigation links lead to correct pages', async ({ page }) => {
        // test IDs
        const buttonToUrlMap = {
            "menu-drawer-item-2": /landing\/faq/,
        };

        for (const [link, targetUrl] of Object.entries(buttonToUrlMap)) {
            await page.goto('/');
            await page.locator('#sidebar-menu-open').click();
            const button = await page.getByTestId(link);
            await button.click();
            await page.waitForURL(targetUrl);
        }
    })
    
});

test.describe('index page login links', () => {
    test("all links on index page lead to login", async ({ page }) => {
        await page.goto('/');

        // currently all links lead to login page
        const loginLinkTexts = [
            "Start Cleaning Now",
            "Get started now",
            "Sign in and get started",
            "Get started",
        ];
        for (const linkText of loginLinkTexts) {
            await page.goto('/');
            const link = await page.getByText(linkText, { exact: true });
            await link.click();
            await page.waitForURL(/login/);
        }
    });
});
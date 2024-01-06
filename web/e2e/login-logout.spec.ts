import { test, expect, Page, Locator } from '@playwright/test';
import axios from "axios";

const testUser = "test.classified@mailboxsweep.local";
const testLoginLinkServerHTTPPort = 8082;

test.describe("return after classification complete", () => {
    test("imap user", async ({ page, context }) => {
        // log console messages
        page.on('console', msg => console.log(msg.text()));

        await page.goto("/login");

        await page.getByLabel("Email").fill(testUser);
        await page.getByText("Sign in").click();

        await expect(page.getByText("Check your email!")).toBeVisible();

        // get login link from temporary server
        const loginLinkRes = await axios.get(`http://localhost:${testLoginLinkServerHTTPPort}?address=${testUser}`);
        const link = loginLinkRes.data.link;
        
        await page.goto(link);

        await page.waitForURL(/home/);
        await expect(page.getByText("Your inbox")).toBeVisible();
        
        const menuButton = await page.getByTestId("sidebar-menu-button");
        await menuButton.click();
        const logoutButton = await page.getByTestId("menu-item-logout");
        await expect(logoutButton).toBeVisible();
        await logoutButton.click();
        const logoutConfirmCheckbox = await page.getByTestId("logout-all-checkbox");
        await expect(logoutConfirmCheckbox).toBeVisible();
        await expect(logoutConfirmCheckbox).toBeChecked();
        await logoutConfirmCheckbox.click();
        await expect(logoutConfirmCheckbox).not.toBeChecked();
        

        const logoutConfirmButton = await page.getByTestId("logout-confirm-button");
        await expect(logoutConfirmButton).toBeVisible();
        await logoutConfirmButton.click();
        const cookies = await context.cookies();
        const sessionCookie = cookies.find(c => c.name === "X-MailboxSweep-Session");
        await expect(sessionCookie?.value).toBeFalsy();
    })
})
import { test, expect, Page, Locator } from '@playwright/test';
import axios from "axios";

const testUser = "test.classified@mailboxsweep.local";
const testLoginLinkServerHTTPPort = 8082;

test.describe("return after classification complete", () => {
    test("imap user", async ({ page }) => {
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
        await expect(page.getByText("Inbox Scan Complete!")).toBeVisible();
        const continueButton = await page.getByText("Let's go", { exact: true });
        await expect(continueButton).toBeVisible();
        await continueButton.click();

        await page.waitForURL(/changeset\/select/)
    })
})
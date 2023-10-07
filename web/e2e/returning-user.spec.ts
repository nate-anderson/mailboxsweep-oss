import { test, expect, Page, Locator } from '@playwright/test';
import axios from "axios";

const testUser = "test.classified@mailboxsweep.local";
const testLoginLinkServerHTTPPort = 8082;

test.describe("return after classification complete", () => {
    test.skip(); // @TODO: finish the test
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

        await page.waitForURL(/changeset\/select/);

        // test toggling mode
        const commitButton = await page.getByTestId("button-commit");
        await expect(commitButton).toHaveText("Mark 0 read");
        await expect(commitButton).toBeDisabled();
        const toggle = await page.getByLabel("toggle-selection-mode");
        await toggle.click();
        await expect(commitButton).toHaveText("Delete 0");
        await expect(commitButton).toBeDisabled();
        await toggle.click();
        await expect(commitButton).toHaveText("Mark 0 read");
        await expect(commitButton).toBeDisabled();
        
        // add mark read messages
        await page.getByLabel("Select messages in group test9.com to mark read").click();
        await expect(commitButton).toHaveText("Mark 154 read");

        // add delete messages
        await toggle.click();
        await expect(commitButton).toHaveText("Delete 0");
        await expect(commitButton).toBeDisabled();
        await page.getByLabel("Select messages in group test3.com to mark read").click();
        await expect(commitButton).toHaveText("Delete 12");
        
    })
})
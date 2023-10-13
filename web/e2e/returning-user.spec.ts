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

        await page.waitForURL(/changeset\/select/);

        // test toggling mode
        const commitButton = await page.getByTestId("button-commit");
        await expect(commitButton).toHaveText("Mark 0 read");
        await expect(commitButton).toBeDisabled();
        const toggle = await page.getByTestId("toggle-selection-mode");
        await toggle.click();
        await expect(commitButton).toHaveText("Delete 0");
        await expect(commitButton).toBeDisabled();
        await toggle.click();
        await expect(commitButton).toHaveText("Mark 0 read");
        await expect(commitButton).toBeDisabled();
        
        // add mark read messages
        const checkBox1 = await page.getByLabel("Select messages in group test9.com to mark read")
        await checkBox1.click();
        await expect(checkBox1).toBeChecked();
        await expect(commitButton).toHaveText("Mark 154 read");

        // add and remove root node
        await page.getByLabel("Select messages in group test4.com to mark read").click();
        await expect(commitButton).toHaveText("Mark 154 read");

        await page.getByLabel("Select messages in group test1.com to mark read").click();
        await expect(commitButton).toHaveText("Mark 209 read");
        await page.getByLabel("Deselect messages in group test1.com to mark read").click();
        await expect(commitButton).toHaveText("Mark 154 read");

        // root node selects children
        await page.getByLabel("Expand category test4.com").click();
        await expect(page.getByLabel("Deselect messages in group Notifications (notifications@test4.com) to mark read")).toBeChecked();
        await page.getByLabel("Go back one level").click();
        await expect(page.getByText("Select Messages", { exact: true })).toBeVisible();

        // add and remove leaf node
        await page.getByLabel("Expand category test8.com", { exact: true }).click();
        await expect(page.getByText("test8.com", { exact: true })).toBeVisible();
        await page.getByLabel("Expand category  (save@test8.com)", { exact: true }).click();
        await expect(page.getByText("(save@test8.com)", { exact: true })).toBeVisible();
        await page.getByLabel("Select messages in group On or after 01-01-2020 to mark read", { exact: true }).click();
        await expect(commitButton).toHaveText("Mark 155 read");
        await page.getByLabel("Deselect messages in group On or after 01-01-2020 to mark read", { exact: true }).click();
        await expect(commitButton).toHaveText("Mark 154 read");
        await page.getByLabel("Go back one level").click();
        await page.getByLabel("Go back one level").click();

        // add delete messages
        await toggle.click();
        await expect(commitButton).toHaveText("Delete 0");
        await expect(commitButton).toBeDisabled();

        // add and remove top-level node
        await page.getByLabel("Select messages in group test3.com to delete").click();
        await expect(commitButton).toHaveText("Delete 17");

        await page.getByLabel("Select messages in group test4.com to delete").click();
        await expect(commitButton).toHaveText("Delete 26");
        await page.getByLabel("Deselect messages in group test4.com to delete").click();
        await expect(commitButton).toHaveText("Delete 17");

        // add and remove leaf
        await page.getByLabel("Expand category test1.com", { exact: true }).click();
        await expect(page.getByText("test1.com", { exact: true })).toBeVisible();
        await page.getByLabel("Expand category Google (no-reply@test1.com)", { exact: true }).click();
        await expect(page.getByText("Google (no-reply@test1.com)", { exact: true })).toBeVisible();
        await page.getByLabel("Select messages in group Before 01-01-2020 to delete", { exact: true }).click();
        await expect(commitButton).toHaveText("Delete 24");
        await page.getByLabel("Deselect messages in group Before 01-01-2020 to delete", { exact: true }).click();
        await expect(commitButton).toHaveText("Delete 17");
        
        // delete - confirm page
        await commitButton.click();
        await page.waitForURL(/changeset\/confirm\/delete/);

        await expect(page.getByText("Deleting 17 messages")).toBeVisible();
        
        // delete button is disabled until box is checked
        const deleteButton = await page.getByLabel("Permanently delete the 17 messages you have selected", { exact: true });
        await expect(deleteButton).toBeDisabled();
        const confirmationCheckBox = await page.getByLabel("Check to confirm you want to permanently delete the 17 messages you have selected", { exact: true });
        await confirmationCheckBox.click();
        await expect(deleteButton).toBeEnabled();

        // mark read - confirm page
        await page.goBack();
        const newCommitButton = await page.getByText("Mark 154 read", { exact: true });
        await expect(newCommitButton).toBeEnabled();
        await newCommitButton.click();
        await page.waitForURL(/changeset\/confirm\/mark\-read/);

        await expect(page.getByText("Marking 154 messages read")).toBeVisible();
        
        // mark read button is disabled until box is checked
        const markReadButton = await page.getByLabel("Permanently mark read the 154 messages you have selected", { exact: true });
        await expect(markReadButton).toBeDisabled();
        const confirmationCheckBoxMarkRead = await page.getByLabel("Check to confirm you want to permanently mark read the 154 messages you have selected", { exact: true });
        await confirmationCheckBoxMarkRead.click();
        await expect(markReadButton).toBeEnabled();
    })
})
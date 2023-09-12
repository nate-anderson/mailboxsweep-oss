import { test, expect } from '@playwright/test';
import axios from "axios";

const testUser = "e2e@mailboxsweep.com";
const testLoginLinkServerHTTPPort = 8082;
const testCardNumber = "4242424242424242";
const testCardExp = "12/30";
const testCardCvv = "123";
const testUserName = "MailboxSweep E2E Test User";
const testImapHost = "localhost";
const testImapPort = 3143;
const testImapPassword = process.env.E2E_TEST_USER_IMAP_PASS;

test.describe('signup', () => {
    test.describe('imap user', () => {
        test('first login', async ({ page }) => {
            
            await page.goto("/login");

            // dialog is invisible
            await expect(page.getByText("Check your email!")).toBeHidden();

            await page.getByLabel("Email").fill(testUser);
            await page.getByText("Sign in").click();

            await expect(page.getByText("Check your email!")).toBeVisible();

            // get login link from temporary server
            const loginLinkRes = await axios.get(`http://localhost:${testLoginLinkServerHTTPPort}`);
            const link = loginLinkRes.data.link;
            
            await page.goto(link);

            await expect(page.getByText("Terms And Conditions")).toBeDefined();
            
            // click accept
            const acceptButton = await page.getByTestId("button-agree-submit");
            await acceptButton.click();
            await expect(page).toHaveURL("/plan/select/");
            await expect(page.getByText("Choose a Plan")).toBeDefined();

            // plan titles, prices, and checkout buttons present
            const plans = ["basic", "annual"];
            for (const plan of plans) {
                const name = await page.getByTestId(`plan-name-${plan}`);
                await expect(name).toBeDefined();

                const priceDollars = await page.getByTestId(`price-${plan}-dollars`);
                await expect(priceDollars).toBeDefined()

                const checkout = await page.getByTestId(`plan-${plan}-checkout`);
                await expect(checkout).toBeDefined();
            }

            const checkout = await page.getByTestId(`plan-basic-checkout`);
            await checkout.click();
            await page.waitForURL(/checkout.stripe.com/);

            // perform checkout
            const cardNumber = await page.locator("[name='cardNumber']");
            await expect(cardNumber).toBeDefined();
            await cardNumber.fill(testCardNumber);

            const cardExp = await page.locator("[name='cardExpiry']");
            await expect(cardExp).toBeDefined();
            await cardExp.fill(testCardExp);

            const cardCvv = await page.locator("[name='cardCvc']");
            await expect(cardCvv).toBeDefined();
            await cardCvv.fill(testCardCvv);

            const name = await page.locator("[name='billingName']");
            await expect(name).toBeDefined();
            await name.fill(testUserName);

            const zip = await page.locator("[name='billingPostalCode']");
            await expect(zip).toBeDefined();
            await zip.fill("90210");

            const rememberMe = await page.getByRole("checkbox");
            await expect(rememberMe).toBeDefined();
            await rememberMe.uncheck();

            const submit = await page.getByText("Pay", { exact: true });
            await submit.click();
            
            // first redirect submits Stripe data to backend to confirm payment
            await page.waitForURL(/plan\/payment-success/);

            // second redirect to classify page
            await page.waitForURL(/email-account\/login/);

            // attempt login
            if (!testImapPassword) {
                throw new Error('E2E_TEST_USER_IMAP_PASS must be set!');
            }
            await page.getByTestId("imap-password-input-no-server").fill(testImapPassword);
            await page.getByText("Save").click();

            // unrecognized server modal is displayed
            await expect(page.getByText("We need a bit more info")).toBeVisible();
            await expect(page.getByText("We don't recognize your email provider")).toBeVisible();
            await page.getByText("Continue", { exact: true }).click();

            // add IMAP server details
            const serverInput = await page.getByTestId("imap-server-input");
            await expect(serverInput).toBeVisible();
            await serverInput.fill(testImapHost);

            const portInput = await page.getByTestId("imap-port-input");
            await expect(portInput).toBeVisible();
            await portInput.fill(testImapPort.toString());

            // login succeeds
            await page.getByText("Save").click();
            
            // redirected to /classify/
            await page.waitForURL(/classify/);
        });
    });
});
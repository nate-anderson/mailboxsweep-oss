import { test, expect } from '@playwright/test';
import axios from "axios";

const testUser = "e2e@mailboxsweep.com";
const testLoginLinkServerHTTPPort = 8082;
const testCardNumber = "4242424242424242";
const testCardExp = "12/30";
const testCardCvv = "123";
const testUserName = "MailboxSweep E2E Test User";

test.describe('signup', () => {
    test.describe('imap user', () => {
        test('first login', async ({ page }) => {
            
            await page.goto("/login");

            // dialog is invisible
            await expect(page.getByText("Check your email!")).toBeHidden();

            await page.getByLabel("Email").type(testUser);
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
            cardNumber.type(testCardNumber);

            const cardExp = await page.locator("[name='cardExpiry']");
            await expect(cardExp).toBeDefined();
            cardExp.type(testCardExp);

            const cardCvv = await page.locator("[name='cardCvc']");
            await expect(cardCvv).toBeDefined();
            cardCvv.type(testCardCvv);

            const name = await page.locator("[name='billingName']");
            await expect(name).toBeDefined();
            name.type(testUserName);

            const zip = await page.locator("[name='billingPostalCode']");
            await expect(zip).toBeDefined();
            zip.type("90210");

            const rememberMe = await page.getByRole("checkbox");
            await expect(rememberMe).toBeDefined();
            await rememberMe.uncheck();

            const submit = await page.getByText("Pay", { exact: true });
            await submit.click();
            
            
        });
    });
});
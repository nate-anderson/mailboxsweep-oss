import { test, expect, Page, Locator } from '@playwright/test';
import axios from "axios";

const testUser = "testuser@mailboxsweep.local";
const testLoginLinkServerHTTPPort = 8082;
const testCardNumber = "4242424242424242";
const testCardExp = "12/30";
const testCardCvv = "123";
const testUserName = "MailboxSweep E2E Test User";
const testImapHost = "127.0.0.1";
const testImapPort = 993;
const testImapPassword = "pass123";

test.describe('signup', () => {
    test.describe('imap user', () => {
        test('first login', async ({ page }) => {
            // log console messages
            page.on('console', msg => console.log(msg.text()));
            
            await page.goto("/login");

            // dialog is invisible
            await expect(page.getByText("Check your email!")).toBeHidden();

            await page.getByLabel("Email").fill(testUser);
            await page.getByText("Sign in").click();

            await expect(page.getByText("Check your email!")).toBeVisible();

            // get login link from temporary server
            const loginLinkRes = await axios.get(`http://localhost:${testLoginLinkServerHTTPPort}?address=${testUser}`);
            const link = loginLinkRes.data.link;
            
            await page.goto(link);

            await expect(page.getByText("Terms And Conditions")).toBeDefined();
            
            // accept terms
            const acceptButton = await page.getByTestId("button-agree-submit");
            await acceptButton.click();
            await page.waitForURL(/plan\/select/);
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

            await page.getByTestId("btn-configure-classify").click();
            await page.waitForURL(/classify\/configure/);

            const classifierTexts = [
                "Sender domain",
                "Sender email",
                "Send date",
            ];
            for (const text of classifierTexts) {
                await expect(page.getByText(text)).toBeVisible();
            }

            const classifierTypeKeys = [
                "SENDER_DOMAIN",
                "SENDER_USERNAME",
                "SEND_DATE"
            ];

            // date input responds to input
            const dateInput = await page.getByTestId('classifier-date-input-SEND_DATE');
            await dateInput.focus();
            const date = "02/02/22";
            await dateInput.type(date);
            await expect(dateInput).toHaveValue(date);

            // expand buttons work
            const toggleTextMap: Record<string, string> = {
                "classifier-expand-toggle-SENDER_DOMAIN": "Classify messages by the website of the sender",
                "classifier-expand-toggle-SENDER_USERNAME": "Classify messages by the sender",
                "classifier-expand-toggle-SEND_DATE": "Divide messages before and after a given date"
            };
            for (const [testId, toggleText] of Object.entries(toggleTextMap)) {
                const toggle = await page.getByTestId(testId);
                await toggle.click();
                const textDisplay = await page.getByText(toggleText);
                await expect(textDisplay).toBeVisible();
                await toggle.click();
                await expect(textDisplay).toBeHidden();
            }

            // drag and drop re-arrange works
            const firstDragHandle = await page.getByTestId("classifier-drag-handle-SENDER_DOMAIN");
            const dropZone = await page.getByTestId("classifier-drag-drop-region");
            const newFirst = await page.getByTestId("classifier-card-SENDER_USERNAME");
            const newSecond = await page.getByTestId("classifier-card-SENDER_DOMAIN");
            const boundingBox = await dropZone.boundingBox();
            
            await dragAndDropSwap(page, firstDragHandle, newFirst);

            const firstY = (await (await newFirst.elementHandle())?.boundingBox())?.y;
            
            const secondY = (await (await newSecond.elementHandle())?.boundingBox())?.y;
            await expect(secondY).toBeGreaterThan(firstY!);

            await page.goBack();
            await page.getByText("Start").click();
            await page.waitForURL(/home/);
            await expect(page.getByText("Inbox Scan In Progress", { exact: true })).toBeVisible();
            await expect(page.getByText("Now is a great time to back up your inbox.", { exact: true })).toBeVisible();
            const learnHowButton = await page.getByText("Learn how")
            await expect(learnHowButton).toBeVisible();
            await learnHowButton.click();
            await page.waitForURL(/article\/backup/);

            const backupLinkTextTargetMap: Record<string, string> = {
                "Learn about Gmail Backup": "https://support.google.com/mail/answer/10016932?hl=en",
                "Learn about Outlook Backup": "https://support.microsoft.com/en-us/office/back-up-your-email-e5845b0b-1aeb-424f-924c-aa1c33b18833",
                "Learn about Backup with Thunderbird": "https://www.ionos.com/help/email/other-email-programs/mozilla-thunderbird-exporting-emails/",
            };

            for (const [text, target] of Object.entries(backupLinkTextTargetMap)) {
                const link = await page.getByText(text);
                const [newTab] = await Promise.all([
                    page.waitForEvent('popup'),
                    link.click(),
                ]);
                await expect(newTab).toHaveURL(target);
            }

            
        });
    });
});

const dragAndDropSwap = async (page: Page, subject: Locator, target: Locator) => {

	const subjectElementBound = {
		x: await subject.boundingBox().then(bound => bound?.x ?? 0),
		y: await subject.boundingBox().then(bound => bound?.y ?? 0),
		width: await subject.boundingBox().then(bound => bound?.width ?? 0),
		height: await subject.boundingBox().then(bound => bound?.height ?? 0),
	};

	const targetElementBound = {
		x: await target.boundingBox().then(bound => bound?.x ?? 0),
		y: await target.boundingBox().then(bound => bound?.y ?? 0),
		width: await target.boundingBox().then(bound => bound?.width ?? 0),
		height: await target.boundingBox().then(bound => bound?.height ?? 0),
	};

	await page.mouse.move(subjectElementBound.x, subjectElementBound.y, { steps: 10 });
    
	await subject.dispatchEvent('mousedown', { button: 0, force: true });

	const x = targetElementBound.x + targetElementBound.width / 2;
	const y = targetElementBound.y + targetElementBound.height / 2;

	await page.mouse.move(x, y, { steps: 10 });

	await target.dispatchEvent('mouseup', { button: 0 });
}
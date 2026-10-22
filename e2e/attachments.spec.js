import { expect, test } from "@playwright/test";
import { PNG_BASE64, attachmentIcon, clearMailbox, messageRows, row, sendMessage } from "./helpers.js";

const toBase64 = (s) => Buffer.from(s).toString("base64");

test.beforeEach(async ({ request }) => {
	await clearMailbox(request);
});

test("a message with an attachment is marked in the list, and one without is not", async ({ page, request }) => {
	await sendMessage(request, {
		subject: "Has attachment",
		attachments: [{ Content: toBase64("hello attachment"), Filename: "hello.txt", ContentType: "text/plain" }],
	});
	await sendMessage(request, { subject: "No attachment" });

	await page.goto("./");
	await expect(messageRows(page)).toHaveCount(2);

	await expect(attachmentIcon(row(page, "Has attachment"))).toBeVisible();
	await expect(attachmentIcon(row(page, "No attachment"))).toHaveCount(0);
});

test("attachments are listed on the message and download with the right content", async ({ page, request }) => {
	await sendMessage(request, {
		subject: "Download me",
		attachments: [
			{ Content: toBase64("hello attachment"), Filename: "hello.txt", ContentType: "text/plain" },
			{ Content: PNG_BASE64, Filename: "pixel.png", ContentType: "image/png" },
		],
	});

	await page.goto("./");
	await row(page, "Download me").click();

	const textCard = page.getByTestId("attachment-card").filter({ hasText: "hello.txt" });
	const imageCard = page.getByTestId("attachment-card").filter({ hasText: "pixel.png" });
	await expect(textCard).toBeVisible();
	await expect(imageCard).toBeVisible();

	// the links point at the API and serve the original bytes
	const textRes = await request.get(await textCard.getAttribute("href"));
	expect(textRes.ok()).toBe(true);
	expect(await textRes.text()).toBe("hello attachment");

	const imageRes = await request.get(await imageCard.getAttribute("href"));
	expect(imageRes.headers()["content-type"]).toContain("image/png");
	expect((await imageRes.body()).toString("base64")).toBe(PNG_BASE64);

	// images get a thumbnail, other types do not
	await expect(imageCard.getByTestId("attachment-thumb")).toHaveAttribute("src", /\/thumb$/);
	await expect(textCard.getByTestId("attachment-thumb")).toHaveCount(0);
});

test("an inline image is rewritten from cid: and actually loads in the preview", async ({ page, request }) => {
	await sendMessage(request, {
		subject: "Inline image",
		html: '<p>Logo:</p><img id="logo" src="cid:mailpit-logo" alt="logo">',
		attachments: [
			{ Content: PNG_BASE64, Filename: "logo.png", ContentType: "image/png", ContentID: "mailpit-logo" },
		],
	});

	await page.goto("./");
	await row(page, "Inline image").click();

	const img = page.frameLocator("#preview-html").locator("#logo");
	await expect(img).toHaveAttribute("src", /\/api\/v1\/message\/.+\/part\/.+/);
	await expect(img).not.toHaveAttribute("src", /^cid:/);
	// naturalWidth is only non-zero once the browser has decoded the image
	await expect.poll(() => img.evaluate((i) => i.complete && i.naturalWidth)).toBeGreaterThan(0);
});

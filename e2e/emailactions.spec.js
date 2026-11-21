import { expect, test } from "@playwright/test";
import { clearMailbox, row, sendMessage } from "./helpers.js";

// Email markup quick actions: schema.org JSON-LD markup in the HTML body is parsed by the server (exposed as `Actions` in the
// API) and shown as a simulated button bar in the message view.

const markup = (json) =>
	`<html><head><script type="application/ld+json">${json}</script></head><body><p>Hi</p></body></html>`;

test.beforeEach(async ({ request }) => {
	await clearMailbox(request);
});

test("valid markup shows the action buttons", async ({ page, request }) => {
	await sendMessage(request, {
		subject: "Approve it",
		html: markup(
			JSON.stringify({
				"@context": "http://schema.org",
				"@type": "EmailMessage",
				potentialAction: [
					{
						"@type": "ConfirmAction",
						name: "Approve",
						target: { "@type": "HttpActionHandler", url: "https://example.com/ok" },
					},
					{ "@type": "ViewAction", name: "View order", target: "https://example.com/order" },
				],
			}),
		),
	});

	await page.goto("./");
	await row(page, "Approve it").click();

	const buttons = page.getByTestId("email-action");
	await expect(buttons).toHaveCount(2);
	await expect(buttons.nth(0)).toContainText("Approve");
	await expect(buttons.nth(1)).toContainText("View order");
	await expect(buttons.nth(0)).toHaveAttribute("title", /https:\/\/example\.com\/ok/);
	await expect(page.getByTestId("email-actions")).not.toContainText("must be HTTPS");
});

test("invalid markup is reported", async ({ page, request }) => {
	await sendMessage(request, {
		subject: "Broken markup",
		html: markup(
			JSON.stringify({
				"@type": "EmailMessage",
				potentialAction: { "@type": "ViewAction", target: "http://example.com" },
			}),
		),
	});

	await page.goto("./");
	await row(page, "Broken markup").click();

	const bar = page.getByTestId("email-actions");
	await expect(bar).toContainText("missing name");
	await expect(bar).toContainText("must be HTTPS");
});

test("a message without markup shows no actions", async ({ page, request }) => {
	await sendMessage(request, { subject: "Plain", html: "<p>nothing</p>" });

	await page.goto("./");
	await row(page, "Plain").click();
	await expect(page.getByText("Plain").first()).toBeVisible();
	await expect(page.getByTestId("email-actions")).toHaveCount(0);
});

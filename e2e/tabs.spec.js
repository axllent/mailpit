import { expect, test } from "@playwright/test";
import { clearMailbox, row, sendMessage } from "./helpers.js";

// Smoke tests for the message view tabs. These check that each tab exists, activates when clicked, and that
// its data loads without an error. They deliberately do not check what the data says: scores, wording and
// results come from the message and from datasets (such as caniemail) that change independently of the UI.

// Some tab ids are rendered twice (a compact dropdown and the full tab bar), so only the visible one counts.
const tab = (page, id) => page.locator(`#${id}-tab:visible`);
const pane = (page, id) => page.locator(`#${id}`);
const hasContent = /\S/;

// a message with HTML and no links, so no check needs to make outbound requests
const html = "<p>Tab smoke test</p>";

const openMessage = async (page, subject) => {
	await page.goto("./");
	await row(page, subject).click();
	await expect(page).toHaveURL(/\/view\/[^/]+$/);
};

test.beforeEach(async ({ request }) => {
	await clearMailbox(request);
});

test.describe("message with HTML and text", () => {
	test.beforeEach(async ({ request }) => {
		await sendMessage(request, { subject: "Tabs", text: "Plain body", html });
	});

	test("every tab is offered", async ({ page }) => {
		await openMessage(page, "Tabs");

		for (const id of [
			"nav-html",
			"nav-html-source",
			"nav-plain-text",
			"nav-headers",
			"nav-raw",
			"nav-html-check",
			"nav-link-check",
		]) {
			await expect(tab(page, id), id).toBeVisible();
		}
	});

	test("the HTML tab is active by default and shows content", async ({ page }) => {
		await openMessage(page, "Tabs");

		await expect(pane(page, "nav-html")).toHaveClass(/\bactive\b/);
		await expect(page.frameLocator("#preview-html").locator("body")).toContainText(hasContent);
	});

	// text-based tabs: clicking activates the pane, and it is not empty
	for (const [id, name] of [
		["nav-html-source", "HTML Source"],
		["nav-plain-text", "Text"],
		["nav-headers", "Headers"],
	]) {
		test(`the ${name} tab activates and loads content`, async ({ page }) => {
			await openMessage(page, "Tabs");

			await tab(page, id).click();

			await expect(pane(page, id)).toHaveClass(/\bactive\b/);
			await expect(pane(page, id)).toContainText(hasContent);
			// control: the previously active pane has been deactivated
			await expect(pane(page, "nav-html")).not.toHaveClass(/\bactive\b/);
		});
	}

	test("the Raw tab activates and loads the message source", async ({ page }) => {
		await openMessage(page, "Tabs");

		const raw = page.waitForResponse((r) => r.url().endsWith("/raw"), { timeout: 5000 });
		await tab(page, "nav-raw").click();

		expect((await raw).ok()).toBe(true);
		await expect(pane(page, "nav-raw")).toHaveClass(/\bactive\b/);
		await expect(page.frameLocator("#nav-raw iframe").locator("body")).toContainText(hasContent);
	});

	test("the HTML Check tab activates and its data loads without error", async ({ page }) => {
		// the check is requested as soon as the message opens, so start listening before navigating
		const check = page.waitForResponse((r) => r.url().endsWith("/html-check"), { timeout: 5000 });
		await openMessage(page, "Tabs");
		expect((await check).ok()).toBe(true);

		await tab(page, "nav-html-check").click();

		await expect(pane(page, "nav-html-check")).toHaveClass(/\bactive\b/);
		await expect(pane(page, "nav-html-check")).toContainText(hasContent);
		await expect(pane(page, "nav-html-check")).not.toContainText("failed to load");
	});

	test("the Link Check tab runs on demand and reports that the message has no links", async ({ page }) => {
		await openMessage(page, "Tabs");

		await tab(page, "nav-link-check").click();
		await expect(pane(page, "nav-link-check")).toHaveClass(/\bactive\b/);

		// before the check runs: a prompt, no result and no status icon on the tab
		const scan = page.getByRole("button", { name: "Check message links" });
		await expect(scan).toBeVisible();
		await expect(scan).toBeEnabled();
		await expect(pane(page, "nav-link-check")).not.toContainText("No links detected");
		await expect(tab(page, "nav-link-check").getByTestId("link-check-ok")).toHaveCount(0);

		// clicking fires the request, and the server reports nothing to check
		const response = page.waitForResponse((r) => r.url().includes("/link-check"), { timeout: 5000 });
		await scan.click();
		const res = await response;
		expect(res.ok()).toBe(true);
		const result = await res.json();
		expect(result.Links).toHaveLength(0);
		expect(result.Errors).toBe(0);

		// the pane replaces the prompt with the result
		await expect(pane(page, "nav-link-check")).toContainText("No links detected");
		await expect(scan).toBeHidden();

		// the component's event reached the message view: the tab now shows the "no errors" tick
		await expect(tab(page, "nav-link-check").getByTestId("link-check-ok")).toBeVisible();
	});
});

// The HTML tabs and HTML Check only apply when a message has HTML, so a text-only message must not offer them.
// This is also what makes the "every tab is offered" test meaningful.
test.describe("text-only message", () => {
	test("offers the text tabs but not the HTML ones", async ({ page, request }) => {
		await sendMessage(request, { subject: "Text only", text: "Just plain text" });
		await openMessage(page, "Text only");

		for (const id of ["nav-plain-text", "nav-headers", "nav-raw", "nav-link-check"]) {
			await expect(tab(page, id), id).toBeVisible();
		}
		for (const id of ["nav-html", "nav-html-source", "nav-html-check"]) {
			await expect(tab(page, id), id).toHaveCount(0);
		}
	});
});

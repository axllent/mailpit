import { expect, test } from "@playwright/test";
import { clearMailbox, markRead, messageRows, row, sendMessage } from "./helpers.js";

test.beforeEach(async ({ page, request }) => {
	await clearMailbox(request);
	await sendMessage(request, { subject: "Alpha invoice", tags: ["billing"] });
	const beta = await sendMessage(request, { subject: "Beta receipt", tags: ["billing"] });
	await sendMessage(request, { subject: "Gamma newsletter" });
	// leaves Alpha and Gamma unread
	await markRead(request, [beta]);

	await page.goto("./");
	await expect(messageRows(page)).toHaveCount(3);
});

const searchBox = (page) => page.getByLabel("Search mailbox");
// the URL changes asynchronously after a search, so always poll it rather than reading it once
const query = (page) => expect.poll(() => new URL(page.url()).searchParams.get("q"));

test("typing a term and pressing Enter filters the list and sets the URL", async ({ page }) => {
	await searchBox(page).fill("invoice");
	await searchBox(page).press("Enter");

	await expect(page).toHaveURL(/\/search\?/);
	await query(page).toBe("invoice");
	await expect(messageRows(page)).toHaveCount(1);
	await expect(row(page, "Alpha invoice")).toBeVisible();
});

test("a search with no matches says so, without listing anything", async ({ page }) => {
	await searchBox(page).fill("zzzznomatch");
	await searchBox(page).press("Enter");

	await expect(page.getByText("No results for")).toBeVisible();
	await expect(messageRows(page)).toHaveCount(0);
});

test("focusing an empty search box offers the popular filters", async ({ page }) => {
	await searchBox(page).focus();

	await expect(page.getByRole("option", { name: /is:unread/ })).toBeVisible();
	await expect(page.getByRole("option", { name: /has:attachment/ })).toBeVisible();
});

test("keyboard: arrow down then Enter applies a complete filter immediately", async ({ page }) => {
	await searchBox(page).fill("is:u");
	// scoped to the listbox: the page-size <select> also exposes role=option elements
	await expect(page.getByRole("listbox").getByRole("option")).toHaveCount(1);

	await searchBox(page).press("ArrowDown");
	await searchBox(page).press("Enter");

	await query(page).toBe("is:unread");
	// Beta was read, so only the two unread messages match
	await expect(messageRows(page)).toHaveCount(2);
	await expect(row(page, "Beta receipt")).toHaveCount(0);
});

test("keyboard: Tab completes the token but does not run the search", async ({ page }) => {
	await searchBox(page).fill("is:u");
	await searchBox(page).press("ArrowDown");
	await searchBox(page).press("Tab");

	await expect(searchBox(page)).toHaveValue("is:unread ");
	await query(page).toBeNull();
	await expect(messageRows(page)).toHaveCount(3);
});

test("keyboard: Escape closes the suggestions", async ({ page }) => {
	await searchBox(page).focus();
	await expect(page.getByRole("listbox")).toBeVisible();

	await searchBox(page).press("Escape");
	await expect(page.getByRole("listbox")).toBeHidden();
});

test("clicking a tag suggestion searches for that tag", async ({ page }) => {
	await searchBox(page).fill("tag:bil");
	await page.getByRole("option", { name: /tag:billing/ }).click();

	await query(page).toBe("tag:billing");
	await expect(messageRows(page)).toHaveCount(2);
	await expect(row(page, "Gamma newsletter")).toHaveCount(0);
});

test("the clear button returns to the full inbox", async ({ page }) => {
	await searchBox(page).fill("invoice");
	await searchBox(page).press("Enter");
	await expect(messageRows(page)).toHaveCount(1);

	await page.getByRole("button", { name: "Clear search" }).click();

	await expect(page).not.toHaveURL(/\/search/);
	await expect(searchBox(page)).toHaveValue("");
	await expect(messageRows(page)).toHaveCount(3);
});

test("search history survives a reload, and entries can be removed or cleared", async ({ page }) => {
	for (const term of ["invoice", "receipt"]) {
		await searchBox(page).fill(term);
		await searchBox(page).press("Enter");
		await expect(page).toHaveURL(new RegExp(`q=${term}`));
	}

	await page.goto("./");
	await searchBox(page).focus();
	await expect(page.getByText("Recent searches")).toBeVisible();
	// newest first
	await expect(page.getByRole("option").filter({ hasText: "Recent search" })).toHaveText([/receipt/, /invoice/]);

	// remove one entry
	await page
		.getByRole("option", { name: /receipt/ })
		.getByLabel("Remove from history")
		.click();
	await expect(page.getByRole("option", { name: /receipt/ })).toHaveCount(0);
	await expect(page.getByRole("option", { name: /invoice/ })).toBeVisible();

	// the removal is persisted
	await page.goto("./");
	await searchBox(page).focus();
	await expect(page.getByRole("option", { name: /receipt/ })).toHaveCount(0);

	// clear everything
	await page.getByRole("button", { name: "Clear all" }).click();
	await page.goto("./");
	await searchBox(page).focus();
	await expect(page.getByText("Recent searches")).toHaveCount(0);
});

test("clicking a history entry reruns that search", async ({ page }) => {
	await searchBox(page).fill("invoice");
	await searchBox(page).press("Enter");
	await expect(messageRows(page)).toHaveCount(1);

	await page.goto("./");
	await searchBox(page).focus();
	await page.getByRole("option", { name: /invoice/ }).click();

	await query(page).toBe("invoice");
	await expect(messageRows(page)).toHaveCount(1);
});

import { expect, test } from "@playwright/test";
import { clearMailbox, listedSubjects, messageRows, sendMessage, sendMessages } from "./helpers.js";

const pageSize = (page) => page.getByTitle("The number of messages displayed per page");
const next = (page) => page.getByTitle(/View next \d+ messages/);
const prev = (page) => page.getByTitle(/View previous \d+ messages/);

test.beforeEach(async ({ page, request }) => {
	await clearMailbox(request);
	const subjects = Array.from({ length: 30 }, (_, i) => `Msg ${String(i + 1).padStart(2, "0")}`);
	await sendMessages(request, subjects);
	await page.goto("./");
	await expect(messageRows(page)).toHaveCount(30);
	await pageSize(page).selectOption("25");
	await expect(messageRows(page)).toHaveCount(25);
});

test("the page size limits the list, and paging moves through it", async ({ page }) => {
	await expect(page.getByText(/1-25\s+of\s+30/)).toBeVisible();
	await expect(prev(page)).toBeDisabled();
	await expect(next(page)).toBeEnabled();
	await expect(page).toHaveURL(/[?&]limit=25(&|$)/);

	const first = await listedSubjects(page);

	await next(page).click();
	await expect(messageRows(page)).toHaveCount(5);
	await expect(page.getByText(/26-30\s+of\s+30/)).toBeVisible();
	await expect(next(page)).toBeDisabled();
	await expect(prev(page)).toBeEnabled();
	await expect(page).toHaveURL(/[?&]start=25(&|$)/);

	// no message appears on both pages
	const second = await listedSubjects(page);
	expect(second.filter((s) => first.includes(s))).toEqual([]);

	await prev(page).click();
	await expect(messageRows(page)).toHaveCount(25);
	expect(await listedSubjects(page)).toEqual(first);
	await expect(page).not.toHaveURL(/[?&]start=/);
});

test("a deep link to a later page restores that page after a reload", async ({ page }) => {
	await page.goto("./?start=25&limit=25");

	await expect(messageRows(page)).toHaveCount(5);
	await expect(page.getByText(/26-30\s+of\s+30/)).toBeVisible();
	await expect(pageSize(page)).toHaveValue("25");
});

test("a message arriving while viewing a later page does not shift that page's contents", async ({ page, request }) => {
	await next(page).click();
	await expect(messageRows(page)).toHaveCount(5);
	const before = await listedSubjects(page);

	await sendMessage(request, { subject: "Arrives while paging" });

	// the offset moves with the new message (and the URL follows, rate limited), so the page stays put
	await expect(page).toHaveURL(/start=26/);
	expect(await listedSubjects(page)).toEqual(before);
	await expect(page.getByText("Arrives while paging")).toHaveCount(0);
});

test("a message arriving on the first page is inserted and the page stays at the limit", async ({ page, request }) => {
	await sendMessage(request, { subject: "Arrives on page one" });

	await expect(messageRows(page).first().getByTestId("message-subject")).toHaveText("Arrives on page one");
	await expect(messageRows(page)).toHaveCount(25);
});

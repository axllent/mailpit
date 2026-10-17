import { expect, test } from "@playwright/test";
import { apiMessages, clearMailbox, listedSubjects, messageRows, sendMessages, unreadBadge } from "./helpers.js";

// Messages are listed newest first, so after sending A to D the display order is D, C, B, A.
// Positions are used rather than names so the tests do not depend on that ordering assumption.

test.beforeEach(async ({ page, request }) => {
	await clearMailbox(request);
	await sendMessages(request, ["Msg A", "Msg B", "Msg C", "Msg D"]);
	await page.goto("./");
	await expect(messageRows(page)).toHaveCount(4);
});

const ctrlClick = (page, i) =>
	messageRows(page)
		.nth(i)
		.click({ modifiers: ["Control"] });
const shiftClick = (page, i) =>
	messageRows(page)
		.nth(i)
		.click({ modifiers: ["Shift"] });
const selectedIndexes = async (page) =>
	(await messageRows(page).evaluateAll((els) => els.map((e) => e.classList.contains("selected")))).flatMap((s, i) =>
		s ? [i] : [],
	);
const sidebar = (page, name) => page.getByRole("button", { name });

test("ctrl-click toggles individual messages and shows the selection actions", async ({ page }) => {
	// control: nothing selected, no selection actions
	await expect(sidebar(page, "Delete selected")).toHaveCount(0);

	await ctrlClick(page, 0);
	await ctrlClick(page, 2);
	await expect.poll(() => selectedIndexes(page)).toEqual([0, 2]);
	await expect(sidebar(page, "Delete selected")).toBeVisible();
	// selecting must not navigate away
	await expect(page).not.toHaveURL(/\/view\//);

	// toggling again deselects
	await ctrlClick(page, 0);
	await expect.poll(() => selectedIndexes(page)).toEqual([2]);

	await sidebar(page, "Cancel selection").click();
	await expect.poll(() => selectedIndexes(page)).toEqual([]);
	await expect(sidebar(page, "Delete selected")).toHaveCount(0);
});

test("shift-click selects the range from the last selected message", async ({ page }) => {
	await ctrlClick(page, 3);
	await shiftClick(page, 1);

	// rows 1 to 3 are selected, row 0 is not
	await expect.poll(() => selectedIndexes(page)).toEqual([1, 2, 3]);
});

test("shift-click works upwards as well as downwards", async ({ page }) => {
	await ctrlClick(page, 0);
	await shiftClick(page, 2);

	await expect.poll(() => selectedIndexes(page)).toEqual([0, 1, 2]);
});

test("mark read and mark unread apply only to the selection, and are only offered when relevant", async ({
	page,
	request,
}) => {
	await ctrlClick(page, 0);
	await ctrlClick(page, 1);

	// all selected messages are unread
	await expect(sidebar(page, "Mark read")).toBeEnabled();
	await expect(sidebar(page, "Mark unread")).toBeDisabled();

	await sidebar(page, "Mark read").click();
	await expect(messageRows(page).nth(0)).toHaveClass(/\bread\b/);
	await expect(messageRows(page).nth(1)).toHaveClass(/\bread\b/);
	await expect(messageRows(page).nth(2)).not.toHaveClass(/\bread\b/); // control: not selected

	const state = await apiMessages(request);
	expect(state.unread).toBe(2);
	expect(state.messages.filter((m) => m.Read)).toHaveLength(2);
	await expect(unreadBadge(page)).toHaveText("2");

	// now a read message is selected: the actions flip
	await ctrlClick(page, 0);
	await expect(sidebar(page, "Mark read")).toBeDisabled();
	await expect(sidebar(page, "Mark unread")).toBeEnabled();

	await sidebar(page, "Mark unread").click();
	await expect(unreadBadge(page)).toHaveText("3");
	expect((await apiMessages(request)).unread).toBe(3);
});

test("delete selected removes only the selected messages", async ({ page, request }) => {
	const before = await listedSubjects(page);

	await ctrlClick(page, 0);
	await ctrlClick(page, 3);
	await sidebar(page, "Delete selected").click();

	await expect(messageRows(page)).toHaveCount(2);
	// the two middle messages remain, in order
	expect(await listedSubjects(page)).toEqual([before[1], before[2]]);

	const state = await apiMessages(request);
	expect(state.total).toBe(2);
	expect(state.messages.map((m) => m.Subject).sort()).toEqual([before[1], before[2]].sort());
});

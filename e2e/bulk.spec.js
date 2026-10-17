import { expect, test } from "@playwright/test";
import { apiMessages, clearMailbox, messageRows, row, sendMessage, sendMessages, unreadBadge } from "./helpers.js";

// "Mark all read" and "Delete all" are destructive, so each is checked three ways: cancelling does nothing
// (the control), confirming does the action, and the server agrees with what the UI shows.

const sidebar = (page, name) => page.getByRole("button", { name });

test.describe("inbox", () => {
	test.beforeEach(async ({ page, request }) => {
		await clearMailbox(request);
		await sendMessages(request, ["One", "Two", "Three"]);
		await page.goto("./");
		await expect(messageRows(page)).toHaveCount(3);
	});

	test("mark all read asks for confirmation, and cancelling changes nothing", async ({ page, request }) => {
		await sidebar(page, "Mark all read").click();
		const modal = page.locator("#MarkAllReadModal");
		await expect(modal).toBeVisible();
		await expect(modal).toContainText("3 messages");

		await modal.getByRole("button", { name: "Cancel" }).click();
		await expect(modal).toBeHidden();
		expect((await apiMessages(request)).unread).toBe(3);
	});

	test("confirming mark all read marks every message read", async ({ page, request }) => {
		await sidebar(page, "Mark all read").click();
		await page.locator("#MarkAllReadModal").getByRole("button", { name: "Confirm" }).click();

		await expect(unreadBadge(page)).toHaveCount(0);
		await expect(sidebar(page, "Mark all read")).toBeDisabled();
		expect((await apiMessages(request)).unread).toBe(0);
		// nothing was deleted
		await expect(messageRows(page)).toHaveCount(3);
	});

	test("delete all asks for confirmation, and cancelling changes nothing", async ({ page, request }) => {
		await sidebar(page, "Delete all").click();
		const modal = page.locator("#DeleteAllModal");
		await expect(modal).toBeVisible();
		await expect(modal).toContainText("permanently delete 3 messages");

		await modal.getByRole("button", { name: "Cancel" }).click();
		await expect(modal).toBeHidden();
		expect((await apiMessages(request)).total).toBe(3);
		await expect(messageRows(page)).toHaveCount(3);
	});

	test("confirming delete all empties the mailbox", async ({ page, request }) => {
		await sidebar(page, "Delete all").click();
		await page.locator("#DeleteAllModal").getByRole("button", { name: "Delete" }).click();

		await expect(page.getByText("No messages in your mailbox")).toBeVisible();
		expect((await apiMessages(request)).total).toBe(0);
	});
});

// In a search the same buttons act on the matching messages only, which is the part most worth protecting.
test.describe("search results", () => {
	test.beforeEach(async ({ page, request }) => {
		await clearMailbox(request);
		await sendMessage(request, { subject: "Alpha invoice", tags: ["billing"] });
		await sendMessage(request, { subject: "Beta receipt", tags: ["billing"] });
		await sendMessage(request, { subject: "Gamma newsletter" });
		await page.goto("search?q=tag%3Abilling");
		await expect(messageRows(page)).toHaveCount(2);
	});

	test("delete all deletes only the messages matching the search", async ({ page, request }) => {
		await sidebar(page, "Delete all").click();
		const modal = page.locator("#DeleteAllModal");
		await expect(modal).toContainText("matching");
		await expect(modal).toContainText("permanently delete 2 messages");
		await modal.getByRole("button", { name: "Delete" }).click();

		// the non-matching message survives, and the UI returns to the inbox showing it
		await expect(row(page, "Gamma newsletter")).toBeVisible();
		await expect(messageRows(page)).toHaveCount(1);
		const state = await apiMessages(request);
		expect(state.messages.map((m) => m.Subject)).toEqual(["Gamma newsletter"]);
	});

	test("mark all read marks only the messages matching the search", async ({ page, request }) => {
		await sidebar(page, "Mark all read").click();
		const modal = page.locator("#MarkAllReadModal");
		await expect(modal).toContainText("2 messages");
		await modal.getByRole("button", { name: "Confirm" }).click();

		await expect.poll(async () => (await apiMessages(request)).unread).toBe(1);
		const state = await apiMessages(request);
		const bySubject = Object.fromEntries(state.messages.map((m) => [m.Subject, m.Read]));
		expect(bySubject).toEqual({ "Alpha invoice": true, "Beta receipt": true, "Gamma newsletter": false });
	});
});

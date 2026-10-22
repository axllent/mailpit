import { expect, test } from "@playwright/test";
import { clearMailbox, messageTags, row, sendMessage } from "./helpers.js";

// Editing a message's tags in the message view. The editor is the bootstrap5-tags widget, which turns a <select> into
// badges plus a combobox. Each change is saved with a PUT to /api/v1/tags, so every test checks what the server
// stored as well as what the page shows.

let target;

test.beforeEach(async ({ request }) => {
	await clearMailbox(request);
	// gives "billing" to the tag suggestions, as an existing tag on another message
	await sendMessage(request, { subject: "Other", tags: ["billing"] });
});

// Opens a message and waits until its tag editor will save changes.
const openForTagEditing = async (page, subject) => {
	await page.goto("./");
	const tagsLoaded = page.waitForResponse((r) => r.url().endsWith("/api/v1/tags") && r.request().method() === "GET", {
		timeout: 5000,
	});
	await row(page, subject).click();
	await tagsLoaded;
	// The message view ignores tag changes until 200ms after the editor initialises, so that loading the existing tags
	// is not mistaken for an edit. There is no event to wait for, so wait that long (with margin) before editing.
	await page.waitForTimeout(400);

	const cell = page.getByTestId("message-tags");
	return {
		badges: cell.locator(".tags-badge"),
		input: cell.getByRole("combobox", { name: "Type a value" }),
		suggestion: (name) => cell.getByRole("menuitem", { name }),
		remove: (tag) => cell.locator(`.tags-badge[data-value="${tag}"]`).getByRole("button", { name: "Clear" }),
	};
};

// the PUT requests the page makes to save tags
const trackTagSaves = (page) => {
	const saves = [];
	page.on("request", (r) => {
		if (r.method() === "PUT" && r.url().endsWith("/api/v1/tags")) saves.push(r.postDataJSON().Tags);
	});
	return saves;
};

const sorted = (tags) => [...tags].sort();

test("adding a new tag saves it, and it survives a reload and shows in the list", async ({ page, request }) => {
	target = await sendMessage(request, { subject: "Target", tags: ["alpha"] });
	const editor = await openForTagEditing(page, "Target");
	await expect(editor.badges).toHaveText(["alpha"]);

	await editor.input.fill("newtag");
	await editor.input.press("Enter");

	await expect(editor.badges).toHaveText(["alpha", "newtag"]);
	await expect.poll(async () => sorted(await messageTags(request, target))).toEqual(["alpha", "newtag"]);

	// still there after a reload, so it came from the server and not just the widget
	await page.reload();
	await expect(page.getByTestId("message-tags").locator(".tags-badge")).toHaveText(["alpha", "newtag"]);

	// and the message list shows it as a tag on the message
	await page.goto("./");
	await expect(row(page, "Target").getByRole("link", { name: "newtag" })).toBeVisible();
	await expect(row(page, "Other").getByRole("link", { name: "newtag" })).toHaveCount(0); // control
});

test("an existing tag can be added from the suggestions", async ({ page, request }) => {
	target = await sendMessage(request, { subject: "Target" });
	const editor = await openForTagEditing(page, "Target");
	await expect(editor.badges).toHaveCount(0);

	await editor.input.fill("bil");
	await editor.suggestion("billing").click();

	await expect(editor.badges).toHaveText(["billing"]);
	await expect.poll(() => messageTags(request, target)).toEqual(["billing"]);
});

test("removing a tag saves the change and leaves the other tags alone", async ({ page, request }) => {
	target = await sendMessage(request, { subject: "Target", tags: ["alpha", "beta"] });
	const editor = await openForTagEditing(page, "Target");
	await expect(editor.badges).toHaveText(["alpha", "beta"]);

	await editor.remove("alpha").click();

	await expect(editor.badges).toHaveText(["beta"]);
	await expect.poll(() => messageTags(request, target)).toEqual(["beta"]);
});

test("removing the last tag leaves the message with no tags", async ({ page, request }) => {
	target = await sendMessage(request, { subject: "Target", tags: ["alpha"] });
	const editor = await openForTagEditing(page, "Target");

	await editor.remove("alpha").click();

	await expect(editor.badges).toHaveCount(0);
	await expect.poll(() => messageTags(request, target)).toEqual([]);

	// the list no longer shows the tag on that message
	await page.goto("./");
	await expect(row(page, "Target")).toBeVisible();
	await expect(row(page, "Target").getByRole("link", { name: "alpha" })).toHaveCount(0);
});

test("an invalid tag name is rejected and never saved", async ({ page, request }) => {
	target = await sendMessage(request, { subject: "Target", tags: ["alpha"] });
	const editor = await openForTagEditing(page, "Target");
	const saves = trackTagSaves(page);

	await editor.input.fill("bad!tag");
	await editor.input.press("Enter");

	// scoped to the tag cell: other tag editors on the page (such as the release dialog) have the same message
	await expect(page.getByTestId("message-tags").getByText("Invalid tag name")).toBeVisible();
	await expect(editor.badges).toHaveText(["alpha"]);

	// control: a valid name is accepted straight afterwards, so the rejection was about the name
	await editor.input.fill("good-tag");
	await editor.input.press("Enter");
	await expect(editor.badges).toHaveText(["alpha", "good-tag"]);
	await expect.poll(async () => sorted(await messageTags(request, target))).toEqual(["alpha", "good-tag"]);

	// exactly one save happened, and the invalid name was never sent
	expect(saves).toHaveLength(1);
	expect(JSON.stringify(saves)).not.toContain("bad!tag");
});

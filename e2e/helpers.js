// Shared helpers. All URLs are relative to the project's baseURL, which includes any webroot.

// a valid 1x1 pixel PNG
export const PNG_BASE64 =
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==";

export const clearMailbox = async (request) => {
	const res = await request.delete("api/v1/messages");
	if (!res.ok()) throw new Error(`failed to clear mailbox: ${res.status()}`);
};

export const sendMessage = async (
	request,
	{ subject, text = "Plain text body", html = "", tags = [], attachments = [] },
) => {
	const res = await request.post("api/v1/send", {
		data: {
			From: { Email: "sender@example.com", Name: "Test Sender" },
			To: [{ Email: "recipient@example.com", Name: "Test Recipient" }],
			Subject: subject,
			Text: text,
			HTML: html,
			Tags: tags,
			Attachments: attachments,
		},
	});
	if (!res.ok()) throw new Error(`failed to send message: ${res.status()} ${await res.text()}`);
	return (await res.json()).ID;
};

// Sends messages one at a time, so their order in the mailbox is deterministic.
export const sendMessages = async (request, subjects, options = {}) => {
	for (const subject of subjects) {
		await sendMessage(request, { subject, ...options });
	}
};

export const markRead = async (request, ids) => {
	const res = await request.put("api/v1/messages", { data: { Read: true, IDs: ids } });
	if (!res.ok()) throw new Error(`failed to mark messages read: ${res.status()}`);
};

// The tags the server has stored for a message.
export const messageTags = async (request, id) => {
	const res = await request.get(`api/v1/message/${id}`);
	if (!res.ok()) throw new Error(`failed to read message: ${res.status()}`);
	return (await res.json()).Tags;
};

// The server's view of the mailbox, to confirm an action really happened rather than just looked like it did.
export const apiMessages = async (request) => {
	const res = await request.get("api/v1/messages?limit=1000");
	if (!res.ok()) throw new Error(`failed to list messages: ${res.status()}`);
	return res.json();
};

// Selectors. These use data-testid attributes on the Vue components rather than styling classes, so a restyle
// does not break the tests. Several elements are rendered twice (mobile and desktop), so only visible ones count.
export const messageRows = (page) => page.getByTestId("message-row");

export const row = (page, subject) => messageRows(page).filter({ hasText: subject });

// Subjects in the order they are displayed (newest first).
export const listedSubjects = (page) => page.getByTestId("message-subject").allTextContents();

export const attachmentIcon = (rowLocator) => rowLocator.locator('[data-testid="message-attachment-icon"]:visible');

export const unreadBadge = (page) => page.locator('[data-testid="unread-badge"]:visible');

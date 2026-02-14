import { getCurrentInstance } from "vue";
import { onKeyStroke } from "@vueuse/core";
import { mailbox } from "../stores/mailbox";
import { isInputFocused } from "../utils/keyboard";

/**
 * Keyboard navigation for message list views (MailboxView, SearchView).
 * Handles j/k and arrow keys to navigate through the message list.
 */
export function useMessageListKeyboardNav() {
	const instance = getCurrentInstance();

	// Navigate to next message (j or ArrowDown)
	onKeyStroke(["j", "ArrowDown"], (e) => {
		if (isInputFocused()) return;
		e.preventDefault();
		const messages = mailbox.messages;
		if (!messages || !messages.length) return;

		// Find current index based on lastMessage or start at -1
		let currentIndex = -1;
		if (mailbox.lastMessage) {
			currentIndex = messages.findIndex((m) => m.ID === mailbox.lastMessage);
		}

		const nextIndex = currentIndex + 1;
		if (nextIndex < messages.length) {
			const nextMessage = messages[nextIndex];
			mailbox.lastMessage = nextMessage.ID;
			instance.proxy.$router.push("/view/" + nextMessage.ID);
		}
	});

	// Navigate to previous message (k or ArrowUp)
	onKeyStroke(["k", "ArrowUp"], (e) => {
		if (isInputFocused()) return;
		e.preventDefault();
		const messages = mailbox.messages;
		if (!messages || !messages.length) return;

		// Find current index based on lastMessage
		let currentIndex = messages.length;
		if (mailbox.lastMessage) {
			currentIndex = messages.findIndex((m) => m.ID === mailbox.lastMessage);
		}

		const prevIndex = currentIndex - 1;
		if (prevIndex >= 0) {
			const prevMessage = messages[prevIndex];
			mailbox.lastMessage = prevMessage.ID;
			instance.proxy.$router.push("/view/" + prevMessage.ID);
		}
	});
}

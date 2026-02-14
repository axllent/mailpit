import { getCurrentInstance } from "vue";
import { onKeyStroke } from "@vueuse/core";
import { isInputFocused } from "../utils/keyboard";

/**
 * Keyboard navigation for message detail view (MessageView).
 * Handles j/k and arrow keys to navigate between messages,
 * and Escape/u to go back to the list.
 */
export function useMessageViewKeyboardNav() {
	const instance = getCurrentInstance();

	// Navigate to next message (j or ArrowDown)
	onKeyStroke(["j", "ArrowDown"], (e) => {
		if (isInputFocused()) return;
		const nextID = instance.proxy.nextID;
		if (nextID) {
			e.preventDefault();
			instance.proxy.$router.push("/view/" + nextID);
		}
	});

	// Navigate to previous message (k or ArrowUp)
	onKeyStroke(["k", "ArrowUp"], (e) => {
		if (isInputFocused()) return;
		const previousID = instance.proxy.previousID;
		if (previousID) {
			e.preventDefault();
			instance.proxy.$router.push("/view/" + previousID);
		}
	});

	// Go back to inbox/search (Escape or u)
	onKeyStroke(["Escape", "u"], (e) => {
		if (isInputFocused()) return;
		e.preventDefault();
		instance.proxy.goBack();
	});
}

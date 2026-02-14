/**
 * Check if user is currently focused on an input element.
 * Used to prevent keyboard shortcuts from triggering while typing.
 */
export function isInputFocused() {
	const el = document.activeElement;
	if (!el) return false;
	const tag = el.tagName.toLowerCase();
	return tag === "input" || tag === "textarea" || tag === "select" || el.isContentEditable;
}

// HTML sanitisation for the message preview iframe

import DOMPurify from "dompurify";

const hooked = new WeakSet();

/**
 * Sanitises message HTML for display. All JavaScript, forms and unsafe attributes are removed,
 * and external links are opened in a new tab with `noopener noreferrer`.
 *
 * A DOMPurify instance may be supplied (e.g. one bound to a jsdom window for testing).
 */
export const sanitizeHTML = (html, purify = DOMPurify) => {
	// register the link hook once per instance, not on every call
	if (!hooked.has(purify)) {
		hooked.add(purify);
		purify.addHook("afterSanitizeAttributes", (node) => {
			// meta refresh navigates the preview iframe, and is not governed by CSP
			if (node.tagName === "META" && (node.getAttribute("http-equiv") || "").trim().toLowerCase() === "refresh") {
				node.remove();
				return;
			}
			if (
				node.tagName !== "A" ||
				(node.hasAttribute("href") && node.getAttribute("href").substring(0, 1) === "#")
			) {
				return;
			}
			if ("target" in node) {
				node.setAttribute("target", "_blank");
				node.setAttribute("rel", "noopener noreferrer");
			}
			if (!node.hasAttribute("target") && (node.hasAttribute("xlink:href") || node.hasAttribute("href"))) {
				node.setAttribute("xlink:show", "_blank");
			}
		});
	}

	return purify.sanitize(html, {
		WHOLE_DOCUMENT: true,
		SANITIZE_DOM: false,
		ADD_TAGS: ["link", "meta", "o:p", "style"],
		ADD_ATTR: [
			"bordercolor",
			"charset",
			"content",
			"hspace",
			"http-equiv",
			"itemprop",
			"itemscope",
			"itemtype",
			"link",
			"vertical-align",
			"vlink",
			"vspace",
			"xml:lang",
		],
		FORBID_TAGS: ["script", "form"], // all JavaScript should be removed
		ALLOW_UNKNOWN_PROTOCOLS: true, // allow link href protocols like myapp:// etc
	});
};

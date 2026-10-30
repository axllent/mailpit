// Plain text to HTML conversion for the message text view

import DOMPurify from "dompurify";

/**
 * Converts plain text to HTML including anchor links.
 * Only <a> tags are permitted in the output (enforced by DOMPurify).
 *
 * A DOMPurify instance may be supplied (e.g. one bound to a jsdom window for testing).
 */
export const textToHTML = (s, purify = DOMPurify) => {
	// Strip the Unicode placeholder characters used below so that attacker-
	// controlled input cannot pre-inject fake HTML tags via those chars.
	let html = s.replace(/(˱˱˱|ˠˠˠ|˲˲˲)/gu, "");

	// RFC2396 appendix E states angle brackets are recommended for text/plain emails to
	// recognize potential spaces in between the URL
	// @see https://www.rfc-editor.org/rfc/rfc2396#appendix-E
	const angleLinks = /<((https?|ftp):\/\/[-\w@:%_+'!.~#?,&//=; ][^>]+)>/gim;
	html = html.replace(angleLinks, "<˱˱˱a href=ˠˠˠ$1ˠˠˠ target=_blank rel=noopener˲˲˲$1˱˱˱/a˲˲˲>");

	// find links without angle brackets, starting with http(s) or ftp (a link may also start the text)
	const regularLinks = /(^|[^ˠ˲]\b)(((https?|ftp):\/\/[-\w@:%_+'!.~#?,&//=;]+))/gim;
	html = html.replace(regularLinks, "$1˱˱˱a href=ˠˠˠ$2ˠˠˠ target=_blank rel=noopener˲˲˲$2˱˱˱/a˲˲˲");

	// plain www links without https?:// prefix
	const shortLinks = /(^|[^/])(www\.[\S]+(\b|$))/gim;
	html = html.replace(shortLinks, "$1˱˱˱a href=ˠˠˠhttp://$2ˠˠˠ target=ˠˠˠ_blankˠˠˠ rel=ˠˠˠnoopenerˠˠˠ˲˲˲$2˱˱˱/a˲˲˲");

	// escape to HTML & convert <>" characters back
	html = html
		.replace(/&/g, "&amp;")
		.replace(/</g, "&lt;")
		.replace(/>/g, "&gt;")
		.replace(/"/g, "&quot;")
		.replace(/'/g, "&#039;")
		.replace(/˱˱˱/g, "<")
		.replace(/˲˲˲/g, ">")
		.replace(/ˠˠˠ/g, '"');

	return purify.sanitize(html, {
		ALLOWED_TAGS: ["a"],
		ALLOWED_ATTR: ["href", "target", "rel"],
	});
};

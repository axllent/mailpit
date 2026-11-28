// Structured data (JSON-LD) found in the HTML of a message

// the most blocks and the largest block (in characters) that is read, so a large message stays cheap to view
export const MAX_BLOCKS = 20;
export const MAX_BLOCK_LENGTH = 64 << 10;
// Pretty-printing indents every level, so deeply nested JSON grows much larger than its source (8 KB can become
// 32 MB). A block is shown as written if its formatted version is over MAX_FORMATTED_LENGTH, and no more than
// MAX_TOTAL_LENGTH characters are shown in all.
export const MAX_FORMATTED_LENGTH = 128 << 10;
export const MAX_TOTAL_LENGTH = 512 << 10;

const hint = /ld\+json/i;

/**
 * Returns the `application/ld+json` script blocks in an HTML message, as
 * `{ text, error, note }`: `text` is the pretty-printed JSON, or the original text if it
 * is not valid JSON (`error` says why) or its formatted version would be too large
 * (`note` says so). The content is only read, never validated or interpreted. Blocks
 * over the limits are not included, and `truncated` is set if any block was left out.
 *
 * A DOMParser may be supplied (e.g. one from a jsdom window for testing).
 */
export const extractStructuredData = (html, parser = new DOMParser()) => {
	const result = { blocks: [], truncated: false };
	let total = 0;
	if (!html || !hint.test(html)) {
		return result;
	}

	// a DOMParser document never runs scripts
	const doc = parser.parseFromString(html, "text/html");

	for (const script of doc.querySelectorAll("script")) {
		if ((script.getAttribute("type") || "").trim().toLowerCase() !== "application/ld+json") {
			continue;
		}
		if (result.blocks.length >= MAX_BLOCKS || script.textContent.length > MAX_BLOCK_LENGTH) {
			result.truncated = true;
			continue;
		}

		const raw = script.textContent.trim();
		const block = { text: raw, error: "", note: "" };
		let value;
		try {
			value = JSON.parse(raw);
		} catch (e) {
			block.error = e.message;
		}

		if (!block.error) {
			try {
				const formatted = JSON.stringify(value, null, 2);
				if (formatted.length <= MAX_FORMATTED_LENGTH) {
					block.text = formatted;
				} else {
					block.note = "Not formatted, as the formatted JSON would be too large";
				}
			} catch {
				// very deeply nested JSON is valid, but exceeds the call stack when formatted
				block.note = "Not formatted, as the JSON is too deeply nested";
			}
		}

		if (total + block.text.length > MAX_TOTAL_LENGTH) {
			result.truncated = true;
			continue;
		}
		total += block.text.length;
		result.blocks.push(block);
	}

	return result;
};

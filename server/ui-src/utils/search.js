// Pure search helpers (token parsing, autocomplete suggestions, history) used by SearchForm.vue

export const MAX_HISTORY = 50;

export const BASE_FILTERS = [
	// Header & Address filters
	{
		key: "to:",
		type: "field",
		group: "Fields",
		icon: "bi-person",
		label: "to:",
		description: "Filter results matching recipient (To)",
		insertText: "to:",
	},
	{
		key: "from:",
		type: "field",
		group: "Fields",
		icon: "bi-person-fill",
		label: "from:",
		description: "Filter results matching sender (From)",
		insertText: "from:",
	},
	{
		key: "subject:",
		type: "field",
		group: "Fields",
		icon: "bi-card-text",
		label: "subject:",
		description: "Filter results matching subject",
		insertText: "subject:",
	},
	{
		key: "tag:",
		type: "field",
		group: "Fields",
		icon: "bi-tag-fill",
		label: "tag:",
		description: "Filter messages tagged with tag name",
		insertText: "tag:",
	},
	{
		key: "cc:",
		type: "field",
		group: "Fields",
		icon: "bi-people",
		label: "cc:",
		description: "Filter results matching Cc",
		insertText: "cc:",
	},
	{
		key: "bcc:",
		type: "field",
		group: "Fields",
		icon: "bi-people-fill",
		label: "bcc:",
		description: "Filter results matching Bcc",
		insertText: "bcc:",
	},
	{
		key: "reply-to:",
		type: "field",
		group: "Fields",
		icon: "bi-reply",
		label: "reply-to:",
		description: "Filter results matching Reply-To",
		insertText: "reply-to:",
	},
	{
		key: "addressed:",
		type: "field",
		group: "Fields",
		icon: "bi-at",
		label: "addressed:",
		description: "Filter across To, From, Cc, Bcc & Reply-To",
		insertText: "addressed:",
	},
	{
		key: "message-id:",
		type: "field",
		group: "Fields",
		icon: "bi-hash",
		label: "message-id:",
		description: "Filter by exact Message-Id",
		insertText: "message-id:",
	},

	// Status & Content filters
	{
		key: "is:unread",
		type: "value",
		group: "Status & content",
		icon: "bi-eye-slash",
		label: "is:unread",
		description: "Filter messages marked as unread",
		insertText: "is:unread",
	},
	{
		key: "is:read",
		type: "value",
		group: "Status & content",
		icon: "bi-eye",
		label: "is:read",
		description: "Filter messages marked as read",
		insertText: "is:read",
	},
	{
		key: "is:tagged",
		type: "value",
		group: "Status & content",
		icon: "bi-tags",
		label: "is:tagged",
		description: "Filter messages having at least one tag",
		insertText: "is:tagged",
	},
	{
		key: "has:attachment",
		type: "value",
		group: "Status & content",
		icon: "bi-paperclip",
		label: "has:attachment",
		description: "Filter messages with attachments",
		insertText: "has:attachment",
	},
	{
		key: "has:inline",
		type: "value",
		group: "Status & content",
		icon: "bi-image",
		label: "has:inline",
		description: "Filter messages with inline images",
		insertText: "has:inline",
	},

	// Dates & Sizes
	{
		key: "before:",
		type: "field",
		group: "Dates & sizes",
		icon: "bi-calendar-event",
		label: "before:",
		description: "Filter messages before date (YYYY/MM/DD)",
		insertText: "before:",
	},
	{
		key: "after:",
		type: "field",
		group: "Dates & sizes",
		icon: "bi-calendar-event",
		label: "after:",
		description: "Filter messages after date (YYYY/MM/DD)",
		insertText: "after:",
	},
	{
		key: "larger:",
		type: "field",
		group: "Dates & sizes",
		icon: "bi-hdd",
		label: "larger:",
		description: "Filter messages larger than size (e.g. 2M, 500KB)",
		insertText: "larger:",
	},
	{
		key: "smaller:",
		type: "field",
		group: "Dates & sizes",
		icon: "bi-hdd-fill",
		label: "smaller:",
		description: "Filter messages smaller than size (e.g. 1.5MB)",
		insertText: "smaller:",
	},

	// Common Negations
	{
		key: "!is:read",
		type: "negation",
		group: "Negations",
		icon: "bi-eye-slash-fill",
		label: "!is:read",
		description: "Filter messages that are unread",
		insertText: "!is:read",
	},
	{
		key: "!is:unread",
		type: "negation",
		group: "Negations",
		icon: "bi-eye-fill",
		label: "!is:unread",
		description: "Filter messages that are read",
		insertText: "!is:unread",
	},
	{
		key: "!has:attachment",
		type: "negation",
		group: "Negations",
		icon: "bi-paperclip",
		label: "!has:attachment",
		description: "Filter messages without attachments",
		insertText: "!has:attachment",
	},
	{
		key: "!has:inline",
		type: "negation",
		group: "Negations",
		icon: "bi-image",
		label: "!has:inline",
		description: "Filter messages without inline images",
		insertText: "!has:inline",
	},
	{
		key: "!is:tagged",
		type: "negation",
		group: "Negations",
		icon: "bi-tags-fill",
		label: "!is:tagged",
		description: "Filter messages having no tags",
		insertText: "!is:tagged",
	},
	{
		key: "!from:",
		type: "negation",
		group: "Negations",
		icon: "bi-person-x",
		label: "!from:",
		description: "Exclude messages matching sender",
		insertText: "!from:",
	},
	{
		key: "!to:",
		type: "negation",
		group: "Negations",
		icon: "bi-person-x-fill",
		label: "!to:",
		description: "Exclude messages matching recipient",
		insertText: "!to:",
	},
	{
		key: "!subject:",
		type: "negation",
		group: "Negations",
		icon: "bi-card-text",
		label: "!subject:",
		description: "Exclude messages matching subject",
		insertText: "!subject:",
	},
	{
		key: "!tag:",
		type: "negation",
		group: "Negations",
		icon: "bi-tag",
		label: "!tag:",
		description: "Exclude messages tagged with tag name",
		insertText: "!tag:",
	},
];

/**
 * Extracts the token/word under or immediately before the text cursor,
 * along with its start and end indices in the full search query string.
 */
export const getTokenInfo = (search, cursorPosition) => {
	const text = search || "";
	const pos = cursorPosition;
	const beforeCursor = text.slice(0, pos);
	const match = beforeCursor.match(/([^\s]+)$/);
	if (!match) {
		return {
			token: "",
			prefix: "",
			startIndex: pos,
			endIndex: pos,
		};
	}
	const prefix = match[1];
	const startIndex = pos - prefix.length;
	const afterCursor = text.slice(pos);
	const endMatch = afterCursor.match(/^([^\s]*)/);
	const suffix = endMatch ? endMatch[1] : "";
	const endIndex = pos + suffix.length;
	return {
		token: prefix + suffix,
		prefix,
		startIndex,
		endIndex,
	};
};

/**
 * Computes categorized autocomplete suggestions based on the active token and search input:
 * 1. Recent searches (limited to 5 in default view, or all matches when typing).
 * 2. Status & content values (e.g. is:unread, has:attachment).
 * 3. Dynamic mailbox tags (tag:<name>).
 * 4. Field filters and common negations (!field:).
 */
export const buildSuggestions = ({
	search = "",
	cursorPosition = 0,
	history = [],
	tags: tagList = [],
	messages: messageList = [],
	colorHash = () => "",
	now = new Date(),
}) => {
	// the store values may not be arrays before the first load
	const tags = Array.isArray(tagList) ? tagList : [];
	const messages = Array.isArray(messageList) ? messageList : [];

	const info = getTokenInfo(search, cursorPosition);
	const token = info.prefix.toLowerCase();
	const groups = [];

	// 1. Recent searches:
	// When search query is empty (default view), limit to 5 recent searches.
	// When search query is not empty, all matching searches from history appear.
	const fullQuery = (search || "").trim().toLowerCase();
	const matchingHistory =
		fullQuery === "" ? history.slice(0, 5) : history.filter((h) => h.toLowerCase().includes(fullQuery));

	if (matchingHistory.length > 0) {
		groups.push({
			name: "Recent searches",
			isHistory: true,
			items: matchingHistory.map((h) => ({
				key: "history-" + h,
				type: "history",
				isHistory: true,
				label: h,
				value: h,
				insertText: h,
				icon: "bi-clock-history",
			})),
		});
	}

	// 2. Filter matching
	let matchingFilters = [];

	if (token === "") {
		// Show popular default filters when cursor is at start of a token
		const popularKeys = [
			"is:unread",
			"is:read",
			"has:attachment",
			"to:",
			"from:",
			"subject:",
			"tag:",
			"before:",
			"after:",
			"larger:",
		];
		matchingFilters = BASE_FILTERS.filter((f) => popularKeys.includes(f.key));
	} else if (token.startsWith("is:") || token.startsWith("!is:") || token.startsWith("-is:")) {
		const isNegated = token.startsWith("!") || token.startsWith("-");
		const sub = token.replace(/^[!-]?is:/, "");
		const valid = [
			{
				val: "read",
				desc: isNegated ? "Filter messages that are unread" : "Filter messages marked as read",
			},
			{
				val: "unread",
				desc: isNegated ? "Filter messages that are read" : "Filter messages marked as unread",
			},
			{
				val: "tagged",
				desc: isNegated ? "Filter messages having no tags" : "Filter messages having at least one tag",
			},
		];
		matchingFilters = valid
			.filter((v) => v.val.startsWith(sub))
			.map((v) => {
				const k = (isNegated ? "!" : "") + "is:" + v.val;
				return {
					key: k,
					type: isNegated ? "negation" : "value",
					group: "Status & content",
					icon: v.val === "read" ? "bi-eye" : v.val === "unread" ? "bi-eye-slash" : "bi-tags",
					label: k,
					description: v.desc,
					insertText: k,
				};
			});
	} else if (token.startsWith("has:") || token.startsWith("!has:") || token.startsWith("-has:")) {
		const isNegated = token.startsWith("!") || token.startsWith("-");
		const sub = token.replace(/^[!-]?has:/, "");
		const valid = [
			{
				val: "attachment",
				desc: isNegated ? "Filter messages without attachments" : "Filter messages with attachments",
				icon: "bi-paperclip",
			},
			{
				val: "inline",
				desc: isNegated ? "Filter messages without inline images" : "Filter messages with inline images",
				icon: "bi-image",
			},
		];
		matchingFilters = valid
			.filter((v) => v.val.startsWith(sub))
			.map((v) => {
				const k = (isNegated ? "!" : "") + "has:" + v.val;
				return {
					key: k,
					type: isNegated ? "negation" : "value",
					group: "Status & content",
					icon: v.icon,
					label: k,
					description: v.desc,
					insertText: k,
				};
			});
	} else if (token.startsWith("tag:") || token.startsWith("!tag:") || token.startsWith("-tag:")) {
		const isNegated = token.startsWith("!") || token.startsWith("-");
		const sub = token.replace(/^[!-]?tag:/, "");
		matchingFilters = tags
			.filter((t) => t.toLowerCase().includes(sub))
			.map((t) => {
				const formatted = t.includes(" ") ? `"${t}"` : t;
				const k = (isNegated ? "!" : "") + "tag:" + formatted;
				return {
					key: k,
					type: "tag-value",
					group: "Tags",
					icon: "bi-tag-fill",
					iconStyle: { color: colorHash(t) },
					label: k,
					description: isNegated
						? `Exclude messages tagged with "${t}"`
						: `Filter messages tagged with "${t}"`,
					insertText: k,
				};
			});
		if (!matchingFilters.length && sub === "") {
			matchingFilters.push({
				key: (isNegated ? "!" : "") + "tag:",
				type: "field",
				group: "Tags",
				icon: "bi-tag-fill",
				label: (isNegated ? "!" : "") + "tag:",
				description: "Filter messages tagged with tag name",
				insertText: (isNegated ? "!" : "") + "tag:",
			});
		}
	} else if (/^[!-]?(before|after):/.test(token)) {
		const m = token.match(/^([!-]?)(before|after):(.*)$/);
		const sub = m[3];
		const fmt = (d) =>
			`${d.getFullYear()}/${String(d.getMonth() + 1).padStart(2, "0")}/${String(d.getDate()).padStart(2, "0")}`;
		const daysAgo = (n) => {
			const d = new Date(now);
			d.setDate(d.getDate() - n);
			return d;
		};
		matchingFilters = [
			{ label: "Today", d: daysAgo(0) },
			{ label: "Yesterday", d: daysAgo(1) },
			{ label: "7 days ago", d: daysAgo(7) },
			{ label: "30 days ago", d: daysAgo(30) },
		]
			.filter((v) => fmt(v.d).startsWith(sub))
			.map((v) => {
				const k = `${m[1]}${m[2]}:${fmt(v.d)}`;
				return {
					key: k,
					type: "field-value",
					group: "Dates & sizes",
					icon: "bi-calendar-event",
					label: k,
					description: v.label,
					insertText: k,
				};
			});
	} else if (/^[!-]?(larger|smaller):/.test(token)) {
		const m = token.match(/^([!-]?)(larger|smaller):(.*)$/);
		const sub = m[3];
		matchingFilters = ["100KB", "500KB", "1MB", "5MB", "10MB"]
			.filter((v) => v.toLowerCase().startsWith(sub))
			.map((v) => {
				const k = `${m[1]}${m[2]}:${v}`;
				return {
					key: k,
					type: "field-value",
					group: "Dates & sizes",
					icon: m[2] === "larger" ? "bi-hdd" : "bi-hdd-fill",
					label: k,
					description: `Messages ${m[2]} than ${v}`,
					insertText: k,
				};
			});
	} else if (/^[!-]?(from|to|cc|bcc|reply-to|addressed):/.test(token)) {
		const m = token.match(/^([!-]?)(from|to|cc|bcc|reply-to|addressed):(.*)$/);
		const sub = m[3].replace(/^"/, "");
		const prop = {
			from: ["From"],
			to: ["To"],
			cc: ["Cc"],
			bcc: ["Bcc"],
			"reply-to": ["ReplyTo"],
			addressed: ["From", "To", "Cc", "Bcc", "ReplyTo"],
		}[m[2]];
		const seen = new Set();
		for (const msg of messages) {
			for (const p of prop) {
				const list = Array.isArray(msg[p]) ? msg[p] : msg[p] ? [msg[p]] : [];
				for (const a of list) {
					if (a && a.Address && a.Address.toLowerCase().includes(sub)) {
						seen.add(a.Address);
					}
				}
			}
		}
		matchingFilters = [...seen].slice(0, 8).map((addr) => {
			const k = `${m[1]}${m[2]}:${addr.includes(" ") ? `"${addr}"` : addr}`;
			return {
				key: k,
				type: "field-value",
				group: "Addresses",
				icon: "bi-person",
				label: k,
				description: "From current messages",
				insertText: k,
			};
		});
		if (!matchingFilters.length && sub === "") {
			matchingFilters.push(BASE_FILTERS.find((f) => f.key === m[2] + ":"));
		}
	} else if (token.startsWith("!") || token.startsWith("-")) {
		const sub = token.slice(1);
		matchingFilters = BASE_FILTERS.filter(
			(f) =>
				(f.type === "negation" || f.key.startsWith("!")) &&
				(f.key.toLowerCase().includes(token) || (sub.length >= 3 && f.description.toLowerCase().includes(sub))),
		);
	} else {
		// General matching across all filters
		matchingFilters = BASE_FILTERS.filter(
			(f) =>
				f.type !== "negation" &&
				(f.key.toLowerCase().includes(token) ||
					(token.length >= 3 && f.description.toLowerCase().includes(token))),
		);

		// Also check if token matches any existing tags in mailbox
		if (Array.isArray(tags)) {
			const matchedTags = tags.filter((t) => t.toLowerCase().includes(token));
			matchedTags.forEach((t) => {
				const formatted = t.includes(" ") ? `"${t}"` : t;
				matchingFilters.push({
					key: "tag:" + formatted,
					type: "tag-value",
					group: "Tags",
					icon: "bi-tag-fill",
					iconStyle: { color: colorHash(t) },
					label: "tag:" + formatted,
					description: `Filter messages tagged with "${t}"`,
					insertText: "tag:" + formatted,
				});
			});
		}
	}

	// Group filters by their group property
	const groupOrder = ["Fields", "Addresses", "Status & content", "Dates & sizes", "Tags", "Negations"];
	groupOrder.forEach((grpName) => {
		const items = matchingFilters.filter((f) => f.group === grpName);
		if (items.length > 0) {
			groups.push({
				name: grpName,
				items,
			});
		}
	});

	return groups;
};

/**
 * Parses a stored search history value, returning only valid, non-empty strings
 * (capped at the latest 50) or [] for anything malformed.
 */
export const parseHistory = (raw) => {
	if (!raw) {
		return [];
	}
	try {
		const parsed = JSON.parse(raw);
		if (Array.isArray(parsed) && parsed.every((v) => typeof v === "string")) {
			return parsed.filter((s) => s.trim().length > 0).slice(0, MAX_HISTORY);
		}
	} catch {
		// ignore malformed JSON
	}
	return [];
};

/**
 * Returns a new history list with the term at the front, deduplicated case-insensitively
 * and capped at 50 items. Blank terms leave the history unchanged.
 */
export const addToHistory = (history, term) => {
	const q = (term || "").trim();
	if (!q) {
		return history;
	}
	const next = history.filter((item) => item.toLowerCase() !== q.toLowerCase());
	next.unshift(q);
	return next.slice(0, MAX_HISTORY);
};

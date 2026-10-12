<script>
import CommonMixins from "../mixins/CommonMixins";
import { mailbox } from "../stores/mailbox";
import { pagination } from "../stores/pagination";

const BASE_FILTERS = [
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

export default {
	mixins: [CommonMixins],

	inject: ["eventBus"],

	emits: ["loadMessages"],

	data() {
		return {
			mailbox,
			search: "",
			showDropdown: false,
			selectedIndex: -1,
			searchHistory: [],
			cursorPosition: 0,
		};
	},

	computed: {
		/**
		 * Extracts the token/word currently under or immediately before the text cursor,
		 * along with its start and end indices in the full search query string.
		 */
		currentTokenInfo() {
			const text = this.search || "";
			const pos = this.cursorPosition;
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
		},

		/**
		 * Computes categorized autocomplete suggestions based on the active token and search input:
		 * 1. Recent searches (limited to 5 in default view, or all matches when typing).
		 * 2. Status & content values (e.g. is:unread, has:attachment).
		 * 3. Dynamic mailbox tags (tag:<name>).
		 * 4. Field filters and common negations (!field:).
		 */
		groupedSuggestions() {
			const info = this.currentTokenInfo;
			const token = info.prefix.toLowerCase();
			const groups = [];

			// 1. Recent searches:
			// When search query is empty (default view), limit to 5 recent searches.
			// When search query is not empty, all matching searches from history appear.
			const fullQuery = (this.search || "").trim().toLowerCase();
			const matchingHistory =
				fullQuery === ""
					? this.searchHistory.slice(0, 5)
					: this.searchHistory.filter((h) => h.toLowerCase().includes(fullQuery));

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
						desc: isNegated
							? "Filter messages without inline images"
							: "Filter messages with inline images",
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
				const tags = Array.isArray(mailbox.tags) ? mailbox.tags : [];
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
							iconStyle: { color: this.colorHash(t) },
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
			} else if (token.startsWith("!") || token.startsWith("-")) {
				const sub = token.slice(1);
				matchingFilters = BASE_FILTERS.filter(
					(f) =>
						(f.type === "negation" || f.key.startsWith("!")) &&
						(f.key.toLowerCase().includes(token) ||
							(sub.length >= 3 && f.description.toLowerCase().includes(sub))),
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
				if (Array.isArray(mailbox.tags)) {
					const matchedTags = mailbox.tags.filter((t) => t.toLowerCase().includes(token));
					matchedTags.forEach((t) => {
						const formatted = t.includes(" ") ? `"${t}"` : t;
						matchingFilters.push({
							key: "tag:" + formatted,
							type: "tag-value",
							group: "Tags",
							icon: "bi-tag-fill",
							iconStyle: { color: this.colorHash(t) },
							label: "tag:" + formatted,
							description: `Filter messages tagged with "${t}"`,
							insertText: "tag:" + formatted,
						});
					});
				}
			}

			// Group filters by their group property
			const groupOrder = ["Fields", "Status & content", "Dates & sizes", "Tags", "Negations"];
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
		},

		flatSuggestions() {
			const flat = [];
			for (const group of this.groupedSuggestions) {
				for (const item of group.items) {
					item.flatIndex = flat.length;
					flat.push(item);
				}
			}
			return flat;
		},
	},

	watch: {
		$route() {
			this.searchFromURL();
		},
	},

	mounted() {
		this.searchFromURL();
		this.loadSearchHistory();
		document.addEventListener("click", this.handleClickOutside);

		this.eventBus.on("apply-search", (q) => {
			this.search = q;
			this.showDropdown = false;
			this.addToSearchHistory(q);
			this.doSearch();
		});
	},

	unmounted() {
		document.removeEventListener("click", this.handleClickOutside);
		this.eventBus.off("apply-search");
	},

	methods: {
		searchFromURL() {
			const urlParams = new URLSearchParams(window.location.search);
			this.search = urlParams.get("q") ? urlParams.get("q") : "";
		},

		/**
		 * Loads stored search history from localStorage (mp-search-history),
		 * capped at the latest 50 entries.
		 */
		loadSearchHistory() {
			try {
				const raw = localStorage.getItem("mp-search-history");
				if (!raw) {
					this.searchHistory = [];
					return;
				}
				const parsed = JSON.parse(raw);
				if (Array.isArray(parsed) && parsed.every((v) => typeof v === "string")) {
					this.searchHistory = parsed.filter((s) => s.trim().length > 0).slice(0, 50);
					return;
				}
			} catch {
				// ignore malformed JSON
			}
			this.searchHistory = [];
		},

		/**
		 * Adds a searched term to the history list (deduplicating case-insensitively)
		 * and persists up to 50 items to localStorage.
		 */
		addToSearchHistory(term) {
			const q = (term || "").trim();
			if (!q) {
				return;
			}
			let history = this.searchHistory.filter((item) => item.toLowerCase() !== q.toLowerCase());
			history.unshift(q);
			history = history.slice(0, 50);
			this.searchHistory = history;
			try {
				localStorage.setItem("mp-search-history", JSON.stringify(history));
			} catch {
				// ignore
			}
		},

		removeFromSearchHistory(term) {
			const history = this.searchHistory.filter((item) => item !== term);
			this.searchHistory = history;
			try {
				localStorage.setItem("mp-search-history", JSON.stringify(history));
			} catch {
				// ignore
			}
			if (this.selectedIndex >= this.flatSuggestions.length) {
				this.selectedIndex = Math.max(-1, this.flatSuggestions.length - 1);
			}
		},

		clearSearchHistory() {
			this.searchHistory = [];
			try {
				localStorage.removeItem("mp-search-history");
			} catch {
				// ignore
			}
			this.selectedIndex = -1;
		},

		handleClickOutside(e) {
			if (!this.$el.contains(e.target)) {
				this.showDropdown = false;
			}
		},

		onInput() {
			if (!mailbox.showSearchSuggestions) {
				return;
			}
			this.updateCursorPosition();
			this.showDropdown = true;
			this.selectedIndex = -1;
		},

		onFocus() {
			if (!mailbox.showSearchSuggestions) {
				return;
			}
			this.updateCursorPosition();
			this.showDropdown = true;
			this.selectedIndex = -1;
		},

		onInputClick() {
			if (!mailbox.showSearchSuggestions) {
				return;
			}
			this.updateCursorPosition();
			this.showDropdown = true;
		},

		updateCursorPosition() {
			if (this.$refs.searchInput) {
				this.cursorPosition = this.$refs.searchInput.selectionStart || 0;
			}
		},

		/**
		 * Keyboard navigation handler for the search input:
		 * - Arrow Down/Up: cycle through suggestions and back to unselected (-1).
		 * - Enter/Tab: select currently highlighted suggestion, if any.
		 * - Escape: close suggestion dropdown.
		 */
		onKeyDown(e) {
			if (!mailbox.showSearchSuggestions) {
				return;
			}
			if (e.key === "ArrowDown") {
				e.preventDefault();
				if (!this.showDropdown) {
					this.showDropdown = true;
					this.selectedIndex = 0;
				} else if (this.flatSuggestions.length > 0) {
					if (this.selectedIndex < this.flatSuggestions.length - 1) {
						this.selectedIndex++;
					} else {
						this.selectedIndex = -1;
					}
					this.scrollToActive();
				}
			} else if (e.key === "ArrowUp") {
				e.preventDefault();
				if (!this.showDropdown) {
					this.showDropdown = true;
					this.selectedIndex = this.flatSuggestions.length - 1;
				} else if (this.flatSuggestions.length > 0) {
					if (this.selectedIndex > 0) {
						this.selectedIndex--;
					} else if (this.selectedIndex === 0) {
						this.selectedIndex = -1;
					} else {
						this.selectedIndex = this.flatSuggestions.length - 1;
					}
					this.scrollToActive();
				}
			} else if (e.key === "Enter") {
				// Only intercept Enter to select a suggestion if the user has explicitly
				// navigated to one via arrow keys. Otherwise let the form submit normally.
				if (this.showDropdown && this.selectedIndex >= 0 && this.selectedIndex < this.flatSuggestions.length) {
					e.preventDefault();
					this.selectItem(this.flatSuggestions[this.selectedIndex]);
				} else {
					this.showDropdown = false;
				}
			} else if (e.key === "Tab") {
				// Only complete with Tab if the user has explicitly highlighted a suggestion.
				if (this.showDropdown && this.selectedIndex >= 0 && this.selectedIndex < this.flatSuggestions.length) {
					e.preventDefault();
					this.selectItem(this.flatSuggestions[this.selectedIndex]);
				}
			} else if (e.key === "Escape") {
				if (this.showDropdown) {
					e.preventDefault();
					this.showDropdown = false;
				}
			}
		},

		scrollToActive() {
			this.$nextTick(() => {
				if (!this.$refs.dropdownMenu) {
					return;
				}
				const activeItem = this.$refs.dropdownMenu.querySelector(".dropdown-item.active");
				if (activeItem) {
					activeItem.scrollIntoView({ block: "nearest" });
				}
			});
		},

		/**
		 * Applies the selected suggestion. For recent searches, the entire query is replaced
		 * and executed. For filter tokens, replaces only the active token under the cursor,
		 * appending a trailing space for completed values, and restores focus and caret position.
		 */
		selectItem(item) {
			if (item.isHistory) {
				this.search = item.value;
				this.showDropdown = false;
				this.addToSearchHistory(item.value);
				this.doSearch();
				return;
			}

			const input = this.$refs.searchInput;
			const tokenInfo = this.currentTokenInfo;

			const before = this.search.slice(0, tokenInfo.startIndex);
			const after = this.search.slice(tokenInfo.endIndex);

			let insertText = item.insertText;
			const isCompleteValue = item.type === "value" || item.type === "negation" || item.type === "tag-value";

			if (isCompleteValue && !after.startsWith(" ")) {
				insertText += " ";
			}

			this.search = before + insertText + after;
			this.showDropdown = false;

			const newCursorPos = before.length + insertText.length;
			this.$nextTick(() => {
				if (input) {
					input.focus();
					input.setSelectionRange(newCursorPos, newCursorPos);
					this.cursorPosition = newCursorPos;
				}
			});
		},

		doSearch(e) {
			this.showDropdown = false;
			pagination.start = 0;
			const query = this.search.trim();

			if (query !== "") {
				this.addToSearchHistory(query);
			}

			if (query === "") {
				this.$router.push("/");
			} else {
				const urlParams = new URLSearchParams(window.location.search);
				const curr = urlParams.get("q");
				if (curr && curr === query) {
					pagination.start = 0;
					this.$emit("loadMessages");
				}
				const p = {
					q: query,
				};
				if (pagination.start > 0) {
					p.start = pagination.start.toString();
				}
				if (pagination.limit !== pagination.defaultLimit) {
					p.limit = pagination.limit.toString();
				}

				const params = new URLSearchParams(p);
				this.$router.push("/search?" + params.toString());
			}

			if (e && e.preventDefault) {
				e.preventDefault();
			}
		},

		resetSearch() {
			this.search = "";
			this.showDropdown = false;
			this.$router.push("/");
		},
	},
};
</script>

<template>
	<form @submit="doSearch">
		<div class="input-group flex-nowrap">
			<div class="ms-md-2 d-flex border bg-body rounded-start flex-fill position-relative">
				<input
					ref="searchInput"
					v-model="search"
					type="text"
					class="form-control border-0"
					aria-label="Search mailbox"
					placeholder="Search mailbox"
					autocomplete="off"
					@input="onInput"
					@focus="onFocus"
					@click="onInputClick"
					@keydown="onKeyDown"
				/>
				<button
					v-if="search != ''"
					type="button"
					class="btn btn-link position-absolute end-0 text-muted"
					aria-label="Clear search"
					@click="resetSearch"
				>
					<i class="bi bi-x-circle"></i>
				</button>

				<!-- Autocomplete suggestions dropdown -->
				<div
					v-if="mailbox.showSearchSuggestions && showDropdown && flatSuggestions.length"
					ref="dropdownMenu"
					class="search-autocomplete dropdown-menu show shadow-sm py-1"
					role="listbox"
					@mousedown.prevent
					@mouseleave="selectedIndex = -1"
				>
					<template v-for="(group, gIdx) in groupedSuggestions" :key="group.name">
						<hr v-if="gIdx > 0" class="dropdown-divider my-1" />
						<div
							class="dropdown-header text-uppercase small fw-bold d-flex justify-content-between align-items-center py-1"
						>
							<span>{{ group.name }}</span>
							<button
								v-if="group.isHistory"
								type="button"
								class="btn btn-link btn-sm p-0 text-decoration-none text-muted"
								style="font-size: 0.72rem"
								@click.stop="clearSearchHistory"
							>
								Clear all
							</button>
						</div>
						<button
							v-for="item in group.items"
							:key="item.key"
							type="button"
							class="dropdown-item d-flex align-items-center justify-content-between py-1 px-3 text-start"
							:class="{ active: selectedIndex === item.flatIndex }"
							role="option"
							:aria-selected="selectedIndex === item.flatIndex"
							@mouseenter="selectedIndex = item.flatIndex"
							@click="selectItem(item)"
						>
							<div class="d-flex align-items-center text-truncate me-2">
								<i
									class="bi me-2 flex-shrink-0"
									:class="[
										item.icon,
										selectedIndex === item.flatIndex ? 'text-white' : 'text-body-secondary',
									]"
									:style="item.iconStyle || ''"
								></i>
								<span class="font-monospace fw-semibold text-truncate me-2">{{ item.label }}</span>
							</div>
							<div class="d-flex align-items-center flex-shrink-0">
								<span
									class="small text-truncate d-none d-sm-inline"
									:class="selectedIndex === item.flatIndex ? 'text-white-50' : 'text-body-secondary'"
								>
									{{ item.isHistory ? "Recent search" : item.description }}
								</span>
								<span
									v-if="item.isHistory"
									class="btn btn-sm btn-link p-0 border-0 ms-2"
									:class="selectedIndex === item.flatIndex ? 'text-white' : 'text-body-secondary'"
									title="Remove from history"
									aria-label="Remove from history"
									role="button"
									@click.stop="removeFromSearchHistory(item.value)"
								>
									<i class="bi bi-x-lg"></i>
								</span>
							</div>
						</button>
					</template>

					<!-- Dropdown footer with keyboard hints -->
					<hr class="dropdown-divider my-1" />
					<div
						class="dropdown-item-text py-1 px-3 d-flex justify-content-between align-items-center text-body-secondary small"
					>
						<span style="font-size: 0.75rem">
							<kbd class="bg-body border text-body">↑</kbd>
							<kbd class="bg-body border text-body">↓</kbd> navigate
							<span class="mx-1">·</span>
							<kbd class="bg-body border text-body">↵</kbd> select
							<span class="mx-1">·</span>
							<kbd class="bg-body border text-body">esc</kbd> close
						</span>
						<button
							type="button"
							class="btn btn-link btn-sm p-0 text-decoration-none text-body-secondary"
							style="font-size: 0.75rem"
							data-bs-toggle="modal"
							data-bs-target="#SearchFiltersModal"
							@click="showDropdown = false"
						>
							<i class="bi bi-question-circle me-1"></i>Help
						</button>
					</div>
				</div>
			</div>
			<button class="btn btn-outline-secondary" type="submit" aria-label="Search" title="Search">
				<i class="bi bi-search"></i>
			</button>
		</div>
	</form>
</template>

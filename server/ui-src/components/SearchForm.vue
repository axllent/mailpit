<script>
import CommonMixins from "../mixins/CommonMixins";
import { mailbox } from "../stores/mailbox";
import { pagination } from "../stores/pagination";
import { addToHistory, buildSuggestions, getTokenInfo, parseHistory } from "../utils/search";

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
		currentTokenInfo() {
			return getTokenInfo(this.search, this.cursorPosition);
		},

		groupedSuggestions() {
			return buildSuggestions({
				search: this.search,
				cursorPosition: this.cursorPosition,
				history: this.searchHistory,
				tags: mailbox.tags,
				messages: mailbox.messages,
				colorHash: this.colorHash,
			});
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
				this.searchHistory = parseHistory(localStorage.getItem("mp-search-history"));
			} catch {
				this.searchHistory = [];
			}
		},

		/**
		 * Adds a searched term to the history list (deduplicating case-insensitively)
		 * and persists up to 50 items to localStorage.
		 */
		addToSearchHistory(term) {
			const history = addToHistory(this.searchHistory, term);
			if (history === this.searchHistory) {
				return;
			}
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
				// Tab populates the input only, it does not run the search.
				if (this.showDropdown && this.selectedIndex >= 0 && this.selectedIndex < this.flatSuggestions.length) {
					e.preventDefault();
					this.selectItem(this.flatSuggestions[this.selectedIndex], false);
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
		selectItem(item, search = true) {
			if (item.isHistory) {
				this.search = item.value;
				this.showDropdown = false;
				if (search) {
					this.addToSearchHistory(item.value);
					this.doSearch();
				}
				return;
			}

			const input = this.$refs.searchInput;
			const tokenInfo = this.currentTokenInfo;

			const before = this.search.slice(0, tokenInfo.startIndex);
			const after = this.search.slice(tokenInfo.endIndex);

			let insertText = item.insertText;
			const isCompleteValue =
				item.type === "value" ||
				item.type === "negation" ||
				item.type === "tag-value" ||
				item.type === "field-value";

			if (isCompleteValue && !after.startsWith(" ")) {
				insertText += " ";
			}

			this.search = before + insertText + after;
			this.showDropdown = false;

			// A complete filter (e.g. "is:unread", not a "from:" prefix awaiting a value) is searched immediately,
			// unless the caller only wants the input populated (Tab completion)
			if (search && isCompleteValue && !item.insertText.endsWith(":")) {
				this.doSearch();
			}

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
					class="search-autocomplete dropdown-menu show shadow-sm p-0"
					role="listbox"
					@mousedown.prevent
					@mouseleave="selectedIndex = -1"
				>
					<div class="search-autocomplete-body py-1 small">
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
										:class="
											selectedIndex === item.flatIndex ? 'text-white-50' : 'text-body-secondary'
										"
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
					</div>

					<!-- Static dropdown footer with keyboard hints and syntax help -->
					<div
						class="search-autocomplete-footer dropdown-item-text pt-1 pb-2 px-3 d-flex justify-content-between align-items-center text-body-secondary small"
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

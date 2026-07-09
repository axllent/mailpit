// State Management

import { reactive, watch } from "vue";

// Parse and validate a string[] from localStorage, returning [] on any invalid value.
const storageToStringArray = (key) => {
	try {
		const raw = localStorage.getItem(key);
		if (!raw) return [];
		const parsed = JSON.parse(raw);
		if (Array.isArray(parsed) && parsed.every((v) => typeof v === "string")) {
			return parsed;
		}
	} catch {
		// ignore malformed JSON
	}
	return [];
};

// global mailbox info
export const mailbox = reactive({
	total: 0, // total number of messages in database
	unread: 0, // total unread messages in database
	count: 0, // total in mailbox or search
	messages: [], // current messages
	tags: [], // all tags
	selected: [], // currently selected
	connected: false, // websocket connection
	searching: false, // current search, false for none
	refresh: false, // to listen from MessagesMixin
	autoPaginating: true, // allows temporary bypass of loadMessages() via auto-pagination
	notificationsSupported: false, // browser supports notifications
	notificationsEnabled: false, // user has enabled notifications
	skipConfirmations: false, // skip modal confirmations for "Delete all" & "mark all read"
	appInfo: {}, // application information
	uiConfig: {}, // configuration for UI
	lastMessage: false, // return scrolling
	defaultReleaseAddresses: storageToStringArray("mp-default-release-addresses"), // default release addresses for released messages

	// settings
	showTagColors: !localStorage.getItem("mp-hide-tag-colors"),
	showHTMLCheck: !localStorage.getItem("mp-hide-html-check"),
	showLinkCheck: !localStorage.getItem("mp-hide-link-check"),
	showSpamCheck: !localStorage.getItem("mp-hide-spam-check"),
	timeZone: localStorage.getItem("mp-time-zone")
		? localStorage.getItem("mp-time-zone")
		: Intl.DateTimeFormat().resolvedOptions().timeZone,
	showAttachmentDetails: localStorage.getItem("mp-show-attachment-details"), // show attachment details
});

watch(
	() => mailbox.count,
	() => {
		mailbox.selected = [];
	},
);

watch(
	() => mailbox.showTagColors,
	(v) => {
		if (v) {
			localStorage.removeItem("mp-hide-tag-colors");
		} else {
			localStorage.setItem("mp-hide-tag-colors", "true");
		}
	},
);

watch(
	() => mailbox.showHTMLCheck,
	(v) => {
		if (v) {
			localStorage.removeItem("mp-hide-html-check");
		} else {
			localStorage.setItem("mp-hide-html-check", "true");
		}
	},
);

watch(
	() => mailbox.showLinkCheck,
	(v) => {
		if (v) {
			localStorage.removeItem("mp-hide-link-check");
		} else {
			localStorage.setItem("mp-hide-link-check", "true");
		}
	},
);

watch(
	() => mailbox.showSpamCheck,
	(v) => {
		if (v) {
			localStorage.removeItem("mp-hide-spam-check");
		} else {
			localStorage.setItem("mp-hide-spam-check", "true");
		}
	},
);

watch(
	() => mailbox.defaultReleaseAddresses,
	(v) => {
		if (v.length) {
			localStorage.setItem("mp-default-release-addresses", JSON.stringify(v));
		} else {
			localStorage.removeItem("mp-default-release-addresses");
		}
	},
);

watch(
	() => mailbox.timeZone,
	(v) => {
		if (v === Intl.DateTimeFormat().resolvedOptions().timeZone) {
			localStorage.removeItem("mp-time-zone");
		} else {
			localStorage.setItem("mp-time-zone", v);
		}
	},
);

watch(
	() => mailbox.showAttachmentDetails,
	(v) => {
		if (v) {
			localStorage.setItem("mp-show-attachment-details", "true");
		} else {
			localStorage.removeItem("mp-show-attachment-details");
		}
	},
);

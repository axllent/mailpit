// State Management

import { reactive, watch } from 'vue'

// global mailbox info
export const mailbox = reactive({
	total: 0,						// total number of messages in database
	unread: 0,						// total unread messages in database
	count: 0,						// total in mailbox or search
	messages: [],					// current messages
	tags: [], 						// all tags
	selected: [], 					// currently selected
	connected: false, 				// websocket connection
	searching: false,				// current search, false for none
	refresh: false, 				// to listen from MessagesMixin
	autoPaginating: true, 			// allows temporary bypass of loadMessages() via auto-pagination
	notificationsSupported: false,
	notificationsEnabled: false,
	appInfo: {},					// application information
	uiConfig: {},					// configuration for UI
	lastMessage: false,				// return scrolling

	// settings
	showTagColors: !localStorage.getItem('hideTagColors') == '1',
	showHTMLCheck: !localStorage.getItem('hideHTMLCheck') == '1',
	showLinkCheck: !localStorage.getItem('hideLinkCheck') == '1',
	showSpamCheck: !localStorage.getItem('hideSpamCheck') == '1',
	timeZone: localStorage.getItem('timeZone') ? localStorage.getItem('timeZone') : Intl.DateTimeFormat().resolvedOptions().timeZone,
})

watch(
	() => mailbox.count,
	(v) => {
		mailbox.selected = []
	}
)

watch(
	() => mailbox.showTagColors,
	(v) => {
		if (v) {
			localStorage.removeItem('hideTagColors')
		} else {
			localStorage.setItem('hideTagColors', '1')
		}
	}
)

watch(
	() => mailbox.showHTMLCheck,
	(v) => {
		if (v) {
			localStorage.removeItem('hideHTMLCheck')
		} else {
			localStorage.setItem('hideHTMLCheck', '1')
		}
	}
)

watch(
	() => mailbox.showLinkCheck,
	(v) => {
		if (v) {
			localStorage.removeItem('hideLinkCheck')
		} else {
			localStorage.setItem('hideLinkCheck', '1')
		}
	}
)

watch(
	() => mailbox.showSpamCheck,
	(v) => {
		if (v) {
			localStorage.removeItem('hideSpamCheck')
		} else {
			localStorage.setItem('hideSpamCheck', '1')
		}
	}
)

watch(
	() => mailbox.timeZone,
	(v) => {
		if (v == Intl.DateTimeFormat().resolvedOptions().timeZone) {
			localStorage.removeItem('timeZone')
		} else {
			localStorage.setItem('timeZone', v)
		}
	}
)

// State Management

import { reactive, watch } from 'vue'

// global mailbox info
export const mailbox = reactive({
	total: 0, 				// total number of messages in database
	unread: 0, 				// total unread messages in database
	count: 0, 				// total in mailbox or search
	messages: [],			// current messages
	tags: [], 				// all tags
	showTagColors: true, 	// show/hide tag colors
	selected: [], 			// currently selected
	connected: false, 		// websocket connection
	searching: false,		// current search, false for none
	refresh: false, 		// to listen from MessagesMixin
	notificationsSupported: false,
	notificationsEnabled: false,
	appInfo: {},			// application information
	uiConfig: {},			// configuration for UI
	lastMessage: false,		// return scrolling
	timeZone: '', 			// browser timezone
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

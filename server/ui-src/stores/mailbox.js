// State Management

import { reactive, watch } from 'vue'
import Tinycon from 'tinycon'

Tinycon.setOptions({
	height: 11,
	background: '#dd0000',
	fallback: false,
	font: '9px arial',
})

// global mailbox info
export const mailbox = reactive({
	total: 0, 				// total number of messages in database
	unread: 0, 				// total unread messages in database
	count: 0, 				// total in mailbox or search
	messages: [],			// current messages
	tags: [], 				// all tags
	showTagColors: false, 	// show tag colors?
	selected: [], 			// currently selected
	connected: false, 		// websocket connection
	searching: false,		// current search, false for none
	refresh: false, 		// to listen from MessagesMixin
	notificationsSupported: false,
	notificationsEnabled: false,
	appInfo: {},			// application information
	uiConfig: {},			// configuration for UI
	lastMessage: false,		// return scrolling
})

watch(
	() => mailbox.unread,
	(v) => {
		if (v == 0) {
			Tinycon.reset()
		} else {
			Tinycon.setBubble(v)
		}
	}
)

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
			localStorage.setItem('showTagsColors', '1')
		} else {
			localStorage.removeItem('showTagsColors')
		}
	}
)
